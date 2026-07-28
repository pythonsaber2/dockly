# Operations guide

## Production placement

Dockly should run on the Docker host it manages. Put an HTTPS reverse proxy in front of port 8080, limit dashboard access with a firewall or private network, and set:

```env
DOCKLY_SECURE_COOKIES=true
DOCKLY_API_TOKEN=<long-random-token>
```

The Compose deployment mounts `/var/run/docker.sock`. Access to that socket is equivalent to root access on the host. A hardened environment can use a restricted Docker socket proxy, but it must permit image builds and container inspect, run, remove, rename, and logs operations.

## Backups

Back up the `dockly-data` volume or `/var/lib/dockly/state.json`. The file includes password material, webhook credentials, repository configuration, and environment values. Encrypt backups and limit access.

A consistent backup can be made while Dockly is stopped:

```bash
docker compose stop dockly
docker run --rm -v dockly_dockly-data:/data -v "$PWD":/backup alpine \
  tar -czf /backup/dockly-backup.tgz -C /data .
docker compose start dockly
```

Restore into an empty data volume, then restart Dockly. Browser sessions are in memory and users must sign in again.

## Upgrades

```bash
git pull --ff-only
docker compose build --pull
docker compose up -d
curl --fail http://127.0.0.1:8080/healthz
```

Back up state before upgrading. Pre-1.0 releases may include documented state migrations.

## Troubleshooting

### Dashboard works but deploys fail

Run `docker compose logs dockly` and verify the Docker socket mount. From the container:

```bash
docker compose exec dockly docker info
```

### Git clone fails

Confirm the repository URL and branch. Public repositories need no configuration. Private repositories require credentials available to Git inside the Dockly runtime; first-class deploy-key management is not in the MVP.

### Health check fails

The provided Compose file maps `host.docker.internal` to the Docker host and configures checks against that name. Direct binary installations use `127.0.0.1`. For custom networking, set `DOCKLY_HEALTH_HOST` to an address from which Dockly can reach the published application ports.

### Port is already allocated

Each application owns one published host port. Change the application port or stop the process currently using it.

### Rollback image is missing

Rollbacks use local Docker images. Avoid aggressive image pruning for images referenced in Dockly's deployment history. Re-deploy from Git if an image has been removed.

## Logs

Dockly writes structured JSON logs to stdout. Application logs are read from Docker and are available in the dashboard or `/api/apps/{id}/logs`.
