# API reference

All responses are JSON. Errors use `{ "error": "message" }`.

Browser requests authenticate with the `dockly_session` cookie. Automation requests can use:

```http
Authorization: Bearer <DOCKLY_API_TOKEN>
```

## Public endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/healthz` | Process health and version |
| `GET` | `/api/setup/status` | Whether first-run setup is required |
| `POST` | `/api/setup` | Set the initial password; only available once |
| `POST` | `/api/login` | Create a browser session |
| `POST` | `/api/logout` | Delete the current browser session |
| `POST` | `/hooks/{appID}` | Queue a webhook deployment |

Setup and login bodies:

```json
{ "password": "a strong password of at least 12 characters" }
```

The setup response includes the webhook token once. The token is also stored in the protected state file.

## Applications

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/api/apps` | List applications without environment values |
| `POST` | `/api/apps` | Create an application |
| `GET` | `/api/apps/{id}` | Get one application |
| `PATCH` | `/api/apps/{id}` | Update supplied settings |
| `DELETE` | `/api/apps/{id}` | Remove the container and application record |
| `PUT` | `/api/apps/{id}/env` | Replace all environment variables |
| `GET` | `/api/apps/{id}/status` | Inspect the Docker container state |
| `GET` | `/api/apps/{id}/logs?lines=300` | Read recent container output |

Create example:

```bash
curl -X POST http://localhost:8080/api/apps \
  -H "Authorization: Bearer $DOCKLY_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hello-api",
    "repository": "https://github.com/example/hello-api.git",
    "branch": "main",
    "dockerfile": "Dockerfile",
    "context": ".",
    "port": 3000,
    "publishPort": 3000,
    "healthPath": "/health"
  }'
```

Environment replacement:

```json
{ "env": { "NODE_ENV": "production", "DATABASE_URL": "..." } }
```

Environment values are accepted and persisted but never included in application responses. Responses expose only sorted `envKeys` metadata.

## Deployments

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/api/deployments` | List all deployments, newest first |
| `GET` | `/api/apps/{id}/deployments` | List one application's deployments |
| `POST` | `/api/apps/{id}/deploy` | Queue a deployment; returns `202` |
| `POST` | `/api/apps/{id}/rollback/{deploymentID}` | Queue rollback to a successful release |

Deployment status progresses through `queued`, `running`, and either `successful` or `failed`. Application status progresses through `idle`, `deploying`, and either `running` or `failed`.

## Webhook authentication

Send the setup-generated token in the header:

```http
X-Dockly-Token: <token>
```

A `token` query parameter is supported for providers that cannot set custom headers, but the header is preferred because URLs are commonly logged.
