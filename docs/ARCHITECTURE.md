# Architecture

Dockly is a single-host deployment control plane. Its architecture is deliberately smaller than general-purpose PaaS products.

## Components

### HTTP server

The Go HTTP server exposes the embedded dashboard, JSON API, login/setup endpoints, health endpoint, and deploy webhooks. It uses Go 1.22 method-aware routes and applies restrictive browser security headers.

### Authentication

The first-run flow stores one password hash using PBKDF2-HMAC-SHA256 with a random salt and 310,000 iterations. Browser sessions are random, in-memory, 24-hour tokens in HttpOnly, SameSite=Strict cookies. Operators can separately provide `DOCKLY_API_TOKEN` for bearer authentication.

A process restart invalidates browser sessions but not the password. This is intentional: no reusable session secret is persisted.

### State store

`internal/store` maintains applications, deployments, the password hash, and the webhook token in one JSON document. Each update is serialized by a mutex, written to an owner-only temporary file, and atomically renamed over the previous document.

This is appropriate for the MVP's single process and low write volume. It is not intended for shared storage or multiple Dockly replicas.

### Deployment engine

The deployment engine shells out to Git and Docker through a narrow runner interface:

1. shallow-clone the configured branch into an isolated build directory;
2. resolve and record the Git commit;
3. build an immutable, deployment-specific Docker image;
4. record the currently running image;
5. replace the named application container;
6. poll the configured HTTP health endpoint;
7. restore the previous image when start-up or health verification fails.

Environment values are written to a mode `0600` temporary env file, passed with Docker's `--env-file`, and removed immediately after container creation. This keeps values out of process arguments and Dockly API responses.

### Dashboard

The dashboard is plain HTML, CSS, and JavaScript embedded in the binary. It has no Node build step or remote runtime assets. The UI polls every 15 seconds and uses the same API exposed to automation clients.

## Deliberate MVP boundaries

- One Dockly process and one Docker Engine
- Direct host-port publishing; no integrated reverse proxy or domains
- Brief restart window when replacing an application container
- No database provisioning, team authorization, or multi-server scheduling
- Public Git repositories or runtime-provided Git credentials
- Retained Docker images are the rollback source; external image cleanup must preserve images still referenced by desired rollback history

## Extension points

The `server.Engine` and `deploy.Runner` interfaces isolate the Docker/Git boundary. Future work can add a proxy-based zero-downtime switch, alternate source authentication, or remote Docker targets without coupling those concerns to HTTP handlers or state persistence.
