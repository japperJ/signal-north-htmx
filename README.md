# Signal North

Signal North is a standalone HTMX showcase built around a small Go HTTP server. It demonstrates real server-driven interactions where the server returns HTML fragments instead of JSON.

The interface uses a dark navy operations-console visual language with warm typography, electric lime actions, cyan telemetry, and visible warning states.

## Features

- `hx-get`, `hx-post`, `hx-put`, and `hx-delete`
- Explicit `hx-target` and `hx-swap` regions
- `hx-trigger` with `changed delay:300ms`
- `hx-indicator` loading states
- `hx-confirm` activity deletion
- `hx-boost` navigation
- `revealed` lazy loading
- `every 5s` health polling
- `hx-swap-oob` request metric updates
- Server-sent events through the local SSE extension
- Progressive HTML form and link fallbacks
- Success, empty, validation, error, and loading states
- Responsive keyboard-accessible layout

## Requirements

- Go 1.24 or newer
- Node.js and npm
- Docker
- Chromium for Playwright tests
- Wincontainer runtime for deployment

HTMX 2.0.4 and the SSE extension 2.2.3 are vendored under `static/vendor`. The application does not depend on a CDN at runtime.

## Local Development

Install browser-test dependencies:

```text
npm install
npx playwright install chromium
```

Start the server on port 8080:

```text
go run ./cmd/server
```

To use another port:

```powershell
$env:PORT = "18080"
go run ./cmd/server
```

## Go Verification

```text
go test ./...
go test -race ./...
go vet ./...
```

On Windows hosts without a C compiler, run the race suite in a Linux Go container:

```text
docker run --rm -v "${PWD}:/src" -w /src golang:1.24.5-bookworm go test -race ./...
```

## Browser Verification

The default Playwright command starts a temporary local server on port 18080:

```text
npm run test:browser
npm run test:browser:headed
```

To test an already-running deployment:

```powershell
$env:BASE_URL = "http://127.0.0.1:8080"
npm run test:browser
```

## Docker

This project does not use Docker Compose.

Build the production image:

```text
docker build -t htmx-showcase:demo .
```

Run it locally:

```text
docker run --rm -p 8080:8080 htmx-showcase:demo
```

The image embeds templates and static assets in the Go binary, runs as a non-root user, exposes port 8080, and has a `/healthz` Docker healthcheck.

## Wincontainer Deployment

The production container is deployed directly to Wincontainer. Docker Compose is not used.

Export the image:

```text
docker save htmx-showcase:demo -o htmx-showcase.tar
```

Load the archive with the Wincontainer `wincontainer_load_image` operation, then run:

```text
image: htmx-showcase:demo
name: htmx-showcase
ports: 8080:8080
env: PORT=8080
```

Inspect the image and container, wait for the healthcheck to report healthy, and review logs for startup errors or restart loops.

## Vercel Hobby Deployment

The application can also run as a separate Vercel Hobby project. Vercel builds the Go server directly and does not use the Dockerfile for this deployment.

The Vercel build configuration is in `vercel.json`. The server detects the Vercel runtime and emits a bounded SSE burst so the function can complete within serverless execution limits. The browser reconnects for subsequent signal bursts. In-memory demo state remains ephemeral.

Deploy with the Vercel CLI:

```text
vercel project add signal-north-htmx
vercel build --prod --project signal-north-htmx --yes
vercel deploy --prebuilt --prod --yes --project signal-north-htmx
```

The Wincontainer and Vercel deployments are independent.

GitHub Actions runs the Go tests, race tests, static analysis, and Playwright suite on pull requests and pushes. A push to `main` deploys the prebuilt output to the Vercel project. Configure these repository secrets:

- `VERCEL_TOKEN`
- `VERCEL_ORG_ID`
- `VERCEL_PROJECT_ID`

## HTTP Routes

| Method | Route | Purpose |
|---|---|---|
| GET | `/` | Full homepage |
| GET | `/healthz` | Plain-text health check |
| GET | `/demo/telemetry` | Telemetry fragment |
| GET | `/demo/search?q=deploy` | Search fragment |
| POST | `/demo/command` | Command result and OOB metric |
| POST | `/demo/activity` | Activity fragment |
| DELETE | `/demo/activity/{id}` | Delete activity |
| GET | `/demo/profile` | Inline edit form |
| PUT | `/demo/profile` | Save profile |
| GET | `/demo/status` | Polled health fragment |
| GET | `/demo/lazy` | Revealed architecture fragment |
| GET | `/demo/explain?demo=telemetry` | HTMX/server/browser explanation fragment |
| GET | `/events` | SSE signal stream |

Missing fragment and static asset paths return 404 rather than the homepage.

## Troubleshooting

- If a control does nothing, confirm `/static/vendor/htmx.min.js` returns 200 and that the browser reports HTMX version `2.0.4`.
- If a fragment renders a full page, inspect the endpoint response and confirm it does not contain `<html>` or `<body>`.
- If local Playwright cannot start, check whether port 18080 is available or set `BASE_URL` to an existing server.
- If Wincontainer reports an unhealthy container, inspect the container logs and request `/healthz` directly.
