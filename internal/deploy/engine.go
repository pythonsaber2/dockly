package deploy

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pythonsaber2/dockly/internal/core"
)

type Runner interface {
	Run(context.Context, string, ...string) (string, error)
}
type HealthChecker interface {
	Check(context.Context, string) error
}

type CommandRunner struct{}

func (CommandRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

type HTTPHealthChecker struct {
	Timeout  time.Duration
	Interval time.Duration
}

func (h HTTPHealthChecker) Check(ctx context.Context, url string) error {
	timeout := h.Timeout
	if timeout == 0 {
		timeout = 45 * time.Second
	}
	interval := h.Interval
	if interval == 0 {
		interval = time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := http.Client{Timeout: 3 * time.Second}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if resp, err := client.Do(req); err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("health check failed for %s: %w", url, ctx.Err())
		case <-ticker.C:
		}
	}
}

type Engine struct {
	workDir    string
	runner     Runner
	health     HealthChecker
	healthHost string
}

func NewEngine(workDir string, runner Runner, health HealthChecker, host ...string) *Engine {
	healthHost := "127.0.0.1"
	if len(host) > 0 && host[0] != "" {
		healthHost = host[0]
	}
	return &Engine{workDir: workDir, runner: runner, health: health, healthHost: healthHost}
}
func DefaultEngine(workDir, healthHost string) *Engine {
	return NewEngine(workDir, CommandRunner{}, HTTPHealthChecker{}, healthHost)
}

func (e *Engine) Deploy(ctx context.Context, app core.Application, d core.Deployment) (core.Deployment, error) {
	d.Status = "running"
	if d.StartedAt.IsZero() {
		d.StartedAt = time.Now().UTC()
	}
	source := filepath.Join(e.workDir, "builds", d.ID)
	_ = os.RemoveAll(source)
	defer os.RemoveAll(source)
	if err := os.MkdirAll(filepath.Dir(source), 0700); err != nil {
		return e.fail(d, err)
	}
	out, err := e.runner.Run(ctx, "git", "clone", "--depth", "1", "--branch", app.Branch, "--single-branch", app.Repository, source)
	d.Logs += out
	if err != nil {
		return e.fail(d, err)
	}
	out, err = e.runner.Run(ctx, "git", "-C", source, "rev-parse", "HEAD")
	d.Logs += out
	if err != nil {
		return e.fail(d, err)
	}
	d.Commit = strings.TrimSpace(out)
	image := "dockly/" + app.ID + ":" + d.ID
	dockerfile, err := safeJoin(source, app.Dockerfile)
	if err != nil {
		return e.fail(d, fmt.Errorf("invalid Dockerfile path: %w", err))
	}
	contextDir, err := safeJoin(source, app.Context)
	if err != nil {
		return e.fail(d, fmt.Errorf("invalid build context: %w", err))
	}
	out, err = e.runner.Run(ctx, "docker", "build", "--pull", "-f", dockerfile, "-t", image, contextDir)
	d.Logs += out
	if err != nil {
		return e.fail(d, err)
	}
	d.Image = image
	if err = e.startImage(ctx, app, image); err != nil {
		return e.fail(d, err)
	}
	d.Status = "successful"
	now := time.Now().UTC()
	d.FinishedAt = &now
	return d, nil
}

func (e *Engine) StartImage(ctx context.Context, app core.Application, image string, d core.Deployment) (core.Deployment, error) {
	d.Status = "running"
	d.Image = image
	if d.StartedAt.IsZero() {
		d.StartedAt = time.Now().UTC()
	}
	if err := e.startImage(ctx, app, image); err != nil {
		return e.fail(d, err)
	}
	d.Status = "successful"
	now := time.Now().UTC()
	d.FinishedAt = &now
	return d, nil
}

func (e *Engine) startImage(ctx context.Context, app core.Application, image string) error {
	name := "dockly-" + app.ID
	oldImage, _ := e.runner.Run(ctx, "docker", "inspect", "--format", "{{.Config.Image}}", name)
	oldImage = strings.TrimSpace(oldImage)
	_, _ = e.runner.Run(ctx, "docker", "rm", "-f", name)
	envPath, err := e.writeEnv(app.Env)
	if err != nil {
		return err
	}
	defer os.Remove(envPath)
	args := []string{"run", "-d", "--name", name, "--restart", "unless-stopped", "--label", "dockly.managed=true", "--label", "dockly.app=" + app.ID, "--env-file", envPath}
	if app.PublishPort > 0 && app.Port > 0 {
		args = append(args, "-p", strconv.Itoa(app.PublishPort)+":"+strconv.Itoa(app.Port))
	}
	args = append(args, image)
	if _, err = e.runner.Run(ctx, "docker", args...); err != nil {
		e.restore(ctx, app, oldImage)
		return err
	}
	if app.PublishPort > 0 && e.health != nil {
		path := app.HealthPath
		if path == "" {
			path = "/"
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		if err = e.health.Check(ctx, "http://"+e.healthHost+":"+strconv.Itoa(app.PublishPort)+path); err != nil {
			_, _ = e.runner.Run(ctx, "docker", "rm", "-f", name)
			e.restore(ctx, app, oldImage)
			return err
		}
	}
	return nil
}
func (e *Engine) restore(ctx context.Context, app core.Application, image string) {
	if image == "" {
		return
	}
	envPath, err := e.writeEnv(app.Env)
	if err != nil {
		return
	}
	defer os.Remove(envPath)
	args := []string{"run", "-d", "--name", "dockly-" + app.ID, "--restart", "unless-stopped", "--label", "dockly.managed=true", "--label", "dockly.app=" + app.ID, "--env-file", envPath}
	if app.PublishPort > 0 && app.Port > 0 {
		args = append(args, "-p", strconv.Itoa(app.PublishPort)+":"+strconv.Itoa(app.Port))
	}
	args = append(args, image)
	_, _ = e.runner.Run(ctx, "docker", args...)
}
func (e *Engine) writeEnv(env map[string]string) (string, error) {
	if err := os.MkdirAll(filepath.Join(e.workDir, "tmp"), 0700); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(filepath.Join(e.workDir, "tmp"), "env-*")
	if err != nil {
		return "", err
	}
	_ = f.Chmod(0600)
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	w := bufio.NewWriter(f)
	for _, k := range keys {
		v := strings.ReplaceAll(env[k], "\n", "\\n")
		if _, err = fmt.Fprintf(w, "%s=%s\n", k, v); err != nil {
			f.Close()
			return "", err
		}
	}
	if err = w.Flush(); err != nil {
		f.Close()
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	return f.Name(), nil
}
func (e *Engine) fail(d core.Deployment, err error) (core.Deployment, error) {
	d.Status = "failed"
	d.Error = err.Error()
	now := time.Now().UTC()
	d.FinishedAt = &now
	return d, err
}
func (e *Engine) Logs(ctx context.Context, appID string, lines int) (string, error) {
	if lines < 1 {
		lines = 200
	}
	return e.runner.Run(ctx, "docker", "logs", "--tail", strconv.Itoa(lines), "dockly-"+appID)
}
func (e *Engine) Status(ctx context.Context, appID string) (string, error) {
	out, err := e.runner.Run(ctx, "docker", "inspect", "--format", "{{.State.Status}}", "dockly-"+appID)
	if err != nil {
		return "stopped", err
	}
	return strings.TrimSpace(out), nil
}
func (e *Engine) Remove(ctx context.Context, appID string) error {
	_, err := e.runner.Run(ctx, "docker", "rm", "-f", "dockly-"+appID)
	if err != nil && strings.Contains(err.Error(), "No such container") {
		return nil
	}
	return err
}
func IsDockerAvailable(ctx context.Context) bool {
	err := exec.CommandContext(ctx, "docker", "info").Run()
	return err == nil
}

func safeJoin(root, relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	root = filepath.Clean(root)
	joined := filepath.Clean(filepath.Join(root, relative))
	rel, err := filepath.Rel(root, joined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path leaves repository")
	}
	return joined, nil
}
