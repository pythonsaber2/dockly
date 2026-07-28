# Dockly

A lightweight, self-hosted deployment platform for shipping Dockerized applications from Git repositories.

Dockly gives a single server the workflow developers expect from a hosted platform—deploys, logs, health checks, release history, and rollbacks—without introducing Kubernetes, Redis, a separate database, or a background queue.

> Dockly is an early MVP. It is designed for a single trusted Docker host and is not yet a multi-node orchestrator.

## Why Dockly

Existing self-hosted platforms are excellent at broad infrastructure management, while CLI deployers are excellent at automation. Dockly focuses on the useful space between them:

- **One small control-plane binary** with an embedded dashboard
- **Docker and Git as the runtime contract**, with no proprietary build format
- **Atomic, file-backed state**—easy to inspect, back up, and move
- **Failure-aware releases** that retain the previous image and restore it when a new container fails its health check
- **A quiet operator interface** centered on applications and release activity

## MVP features

- Deploy a Git branch using its Dockerfile
- Build and run applications with Docker
- Password-only console authentication
- Optional bearer-token API authentication
- Server-side environment variable storage (values are never returned by the API)
- Container logs and live status checks
- Deployment history with commit and image references
- Rollback to any successful retained image
- HTTP health checks with automatic recovery of the previous image
- Token-authenticated deployment webhooks
- Responsive, keyboard-accessible dashboard
- JSON API, Docker image, Compose setup, tests, and GitHub Actions CI

## Quick start

Requirements: Linux, Docker Engine, and Docker Compose v2.

```bash
git clone https://github.com/pythonsaber2/dockly.git
cd dockly
cp .env.example .env
docker compose up -d --build
```

Open `http://localhost:8080`, choose a password of at least 12 characters, and add an application.

The application repository must contain a Dockerfile and expose an HTTP endpoint for the configured health path. Its published port must be free on the Dockly host.

### Run the binary directly

Dockly also runs as a single Go binary. The host needs `git` and the Docker CLI, and the current user must be allowed to access Docker.

```bash
make build
DOCKLY_DATA_DIR="$PWD/.dockly" ./bin/dockly
```

## Project website

The standalone Dockly website is currently available at [http://103.73.65.155:5070](http://103.73.65.155:5070). No domain or TLS endpoint has been assigned yet.

Run the website locally with:

```bash
make site-run
```

The marketing site is separate from Dockly's authenticated embedded dashboard. See [`docs/WEBSITE.md`](docs/WEBSITE.md) for its source layout, visual-asset pipeline, production server, and systemd deployment.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `DOCKLY_LISTEN` | `:8080` | HTTP listen address |
| `DOCKLY_DATA_DIR` | `/var/lib/dockly` | State, temporary files, and build workspace |
| `DOCKLY_API_TOKEN` | empty | Optional bearer token for `/api/*` |
| `DOCKLY_SECURE_COOKIES` | `false` | Set `true` when serving Dockly over HTTPS |
| `DOCKLY_HEALTH_HOST` | `127.0.0.1` | Hostname used to check published application ports |

Command-line flags `-listen` and `-data-dir` override their corresponding defaults.

## Deploying an application

1. Add the repository URL, branch, Dockerfile path, container port, published host port, and health path.
2. Add environment variables in the application detail panel.
3. Select **Deploy now**.
4. Dockly clones the selected branch, records its commit, builds an image, replaces the application container, and waits for the health endpoint.
5. If the new release is unhealthy, Dockly removes it and restarts the previous image with the current application configuration.

Containers are named `dockly-<application-id>` and labeled with `dockly.managed=true`.

### Webhooks

Initial setup returns a webhook token. Trigger a deployment with:

```bash
curl -X POST \
  -H "X-Dockly-Token: YOUR_WEBHOOK_TOKEN" \
  http://localhost:8080/hooks/APPLICATION_ID
```

Use your Git provider's push webhook to call this endpoint. Keep the token secret.

### API

With `DOCKLY_API_TOKEN` set:

```bash
curl -H "Authorization: Bearer $DOCKLY_API_TOKEN" \
  http://localhost:8080/api/apps
```

See [`docs/API.md`](docs/API.md) for endpoints and examples.

## Architecture

Dockly intentionally uses a small architecture:

```text
Browser / API client
        │
        ▼
┌──────────────────────────────┐
│ Dockly Go binary             │
│ auth · API · embedded UI     │
│ deployment engine · monitor  │
└──────────────┬───────────────┘
               ├── atomic JSON state
               ├── git CLI
               └── Docker CLI ── Docker Engine
```

Read [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for design decisions and boundaries.

## Security notes

Mounting the Docker socket grants Dockly host-level control. Run Dockly only for trusted administrators, place it behind HTTPS, set `DOCKLY_SECURE_COOKIES=true`, and restrict network access to the dashboard. State is written with owner-only permissions, but backups must be protected because state includes environment values and password material.

Private Git repositories work when Git credentials are configured in the Dockly runtime. First-class deploy keys and encrypted-at-rest secrets are planned beyond the MVP.

Please report vulnerabilities privately as described in [`SECURITY.md`](SECURITY.md).

## Development

```bash
make test
make build
make site-build
make check
```

The project uses only the Go standard library. See [`CONTRIBUTING.md`](CONTRIBUTING.md) before opening a pull request.

## Project status

The MVP is intentionally single-host and direct-port based. It does not yet provide TLS termination, domains, databases, multi-server scheduling, teams, or zero-downtime traffic switching. Those features should only be added when they preserve Dockly's small operational footprint.

## License

Dockly is available under the [MIT License](LICENSE).
