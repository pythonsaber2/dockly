# Dockly website

Dockly's public marketing site lives in [`site/`](../site). It is intentionally independent from the authenticated dashboard embedded in the main Dockly binary.

## Local development

```bash
make site-run
```

The site server listens on `0.0.0.0:5070` by default. Override its address or document root with `DOCKLY_SITE_LISTEN` and `DOCKLY_SITE_ROOT`.

```bash
DOCKLY_SITE_LISTEN=127.0.0.1:5070 \
DOCKLY_SITE_ROOT="$PWD/site" \
make site-run
```

## Product visuals

The feature artwork is generated from the real Dockly dashboard capture checked into `site/assets/dockly-dashboard.png`:

```bash
python3 site/scripts/build-visuals.py
```

The script requires Pillow. Generated images are written to `site/assets/home/` and should be reviewed at desktop and mobile widths before committing them.

## Production server

Build the standalone static server:

```bash
make site-build
./bin/dockly-site -listen 0.0.0.0:5070 -root ./site
```

The server provides:

- explicit `GET` and `HEAD` handling
- gzip compression for text assets
- cache headers for static assets
- CSP, frame, referrer, MIME-sniffing, and permissions headers
- bounded HTTP timeouts

A hardened systemd template is available at [`deploy/dockly-site.service`](../deploy/dockly-site.service). Its expected release layout is:

```text
/opt/dockly-site/
├── current -> releases/<release-id>
└── releases/<release-id>/
    ├── dockly-site
    └── site/
```

Production uses an isolated nginx virtual host for `usedockly.com` and `www.usedockly.com`. Nginx proxies HTML and ancillary routes to the Go server on `127.0.0.1:5070`, serves versioned static assets directly, and provides HTTP/2 TLS, long-lived caching, Brotli/gzip compression, and security headers. Port `5070` is intentionally unavailable from the public network. The committed template is [`deploy/usedockly.com.nginx`](../deploy/usedockly.com.nginx).

The Let's Encrypt certificate covers both hostnames. Install [`deploy/certbot-deploy-hook.sh`](../deploy/certbot-deploy-hook.sh) under `/etc/letsencrypt/renewal-hooks/deploy/` so successful renewals validate and reload nginx automatically.
