// Command dockly-site serves Dockly's public marketing site.
package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type siteHandler struct{ root string }

func main() {
	listen := flag.String("listen", env("DOCKLY_SITE_LISTEN", "0.0.0.0:5070"), "listen address")
	root := flag.String("root", env("DOCKLY_SITE_ROOT", "./site"), "site directory")
	flag.Parse()

	absolute, err := filepath.Abs(*root)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(absolute, "index.html")); err != nil {
		log.Fatalf("site root is not ready: %v", err)
	}

	server := &http.Server{
		Addr:              *listen,
		Handler:           logging(security(siteHandler{root: absolute})),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("Dockly site listening on %s from %s", *listen, absolute)
	log.Fatal(server.ListenAndServe())
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func (h siteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	requestPath := r.URL.Path
	if requestPath == "/" {
		requestPath = "/index.html"
	}
	clean := filepath.Clean(strings.TrimPrefix(requestPath, "/"))
	if clean == "." || strings.HasPrefix(clean, "..") {
		http.NotFound(w, r)
		return
	}
	full := filepath.Join(h.root, clean)
	if !strings.HasPrefix(full, h.root+string(os.PathSeparator)) {
		http.NotFound(w, r)
		return
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(full)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	contentType := mime.TypeByExtension(filepath.Ext(full))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if clean == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=604800")
	}
	if acceptsGzip(r) && compressible(contentType) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			gz := gzip.NewWriter(w)
			_, _ = io.Copy(gz, file)
			_ = gz.Close()
		}
		return
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func acceptsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

func compressible(contentType string) bool {
	return strings.HasPrefix(contentType, "text/") || strings.Contains(contentType, "javascript") || strings.Contains(contentType, "json") || strings.Contains(contentType, "svg")
}

func security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'none'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started).Round(time.Millisecond))
	})
}

func init() {
	if err := mime.AddExtensionType(".js", "text/javascript; charset=utf-8"); err != nil {
		panic(fmt.Sprintf("register JavaScript MIME type: %v", err))
	}
}
