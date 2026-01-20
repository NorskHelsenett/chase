# Bookmarker Crawler Service

A standalone Go-based web crawler service that renders JavaScript-heavy pages and captures screenshots without cookie consent banners.

## Features

- **Go + Headless Chromium**: Uses Chrome DevTools Protocol via [rod](https://github.com/go-rod/rod) to render SPAs and dynamic content
- **Flexible Screenshots**: PNG screenshots with full-page mode plus configurable viewport size
- **Consent Cleanup**: Removes common cookie consent banners before capture
- **HTML Rendering**: Get rendered HTML after JavaScript execution
- **Reddit-style URLs**: Simple file extension-based routing

## API

All endpoints use file extensions for format selection (like Reddit):

### GET /{url}/.png

Get a PNG screenshot of the page.

**Examples:**
```bash
# Simple domain (auto-adds https://)
curl http://localhost:11235/vg.no/.png > screenshot.png

# Full URL
curl http://localhost:11235/https://example.com/.png > screenshot.png

# Use in HTML
<img src="http://crawler:11235/example.com/.png" />
```

### GET /{url}/.html

Get the rendered HTML after JavaScript execution.

**Example:**
```bash
curl http://localhost:11235/example.com/.html
```

### Query parameters

- `fullscreen` or `fullpage`: Capture a full-page screenshot
- `width`: Viewport width in pixels (default: 1920)
- `height`: Viewport height in pixels (default: 1080)

**Examples:**
```bash
# Full-page screenshot
curl "http://localhost:11235/example.com/.png?fullpage" > screenshot.png

# Custom viewport size
curl "http://localhost:11235/example.com/.png?width=1280&height=720" > screenshot.png
```

### GET /, /health, or /healthz

Health check endpoint.

## Development

```bash
# Install dependencies
go mod download

# Run locally
go run cmd/server/main.go

# Build
go build -o crawler cmd/server/main.go

# Run tests
go test ./...

# Lint
go vet ./...
```

## Docker

```bash
# Build image
docker build -t bookmarker-crawler .

# Run container
docker run -p 11235:11235 bookmarker-crawler
```

## Environment Variables

- `PORT`: Server port (default: 11235)
- `CHROME_BIN`: Path to Chrome/Chromium binary (auto-detected)
- `CHROME_PATH`: Path to Chrome/Chromium libraries (auto-detected)

## Architecture

The service consists of three main components:

1. **HTTP Server** (`cmd/server/main.go`): Handles incoming crawl requests
2. **Crawler** (`internal/crawler.go`): Manages browser instances and page rendering
3. **Consent Remover** (`internal/consent.go`): Removes cookie consent banners

## Cookie Consent Handling

The service uses a multi-strategy approach:

1. **Pre-set cookies**: Sets common consent cookies before page load
2. **Element removal**: Removes consent banners after page load using CSS selectors
3. **Overlay cleanup**: Removes modal backdrops and overlays that block content

Supported consent frameworks:
- OneTrust
- CookieBot
- TrustArc
- Quantcast
- SourcePoint
- Didomi
- Osano
- Generic cookie/consent/GDPR patterns

## Deployment

This service is designed to be deployed as a separate microservice:

- **Kubernetes**: Use the provided Helm chart
- **Docker Compose**: See `.devcontainer/docker-compose.yml` for example
- **Standalone**: Run the binary with required Chrome/Chromium installation

## Performance

- **Memory**: ~150-200MB per concurrent browser instance
- **CPU**: Low when idle, spikes during page rendering
- **Startup**: ~1-2 seconds per crawl (includes browser launch)
- **Concurrency**: Handles multiple requests in parallel

## License

MIT
