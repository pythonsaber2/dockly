# Security Policy

## Supported versions

Dockly is currently pre-1.0. Security fixes are applied to the latest commit on `main` and included in the next release. Older development snapshots are not supported.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability. Use GitHub's **Report a vulnerability** flow in the Security tab of this repository. Include the affected version or commit, reproduction steps, impact, and any suggested mitigation.

You should receive an acknowledgement within 72 hours and a status update within seven days. Please allow reasonable time for a fix and coordinated disclosure.

## Deployment guidance

Dockly controls the Docker daemon and therefore has host-level power. A secure installation should:

- restrict access to trusted operators;
- terminate TLS in front of Dockly and set `DOCKLY_SECURE_COOKIES=true`;
- protect or narrowly proxy `/var/run/docker.sock`;
- keep the host, Docker Engine, Git, and Dockly updated;
- firewall the dashboard and application ports intentionally;
- back up `/var/lib/dockly` securely because it contains environment values;
- set a strong, unique `DOCKLY_API_TOKEN` if API authentication is enabled;
- rotate the API and webhook tokens after suspected disclosure.

Dockly never returns stored environment values through its API. It does pass them to Docker through an owner-only temporary env file that is removed after container creation.
