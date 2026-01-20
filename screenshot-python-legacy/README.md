# Screenshot Service

A standalone Python + Selenium service that renders JavaScript-heavy pages and captures screenshots or rendered HTML.

## Features

- **Python + Headless Firefox**: Uses Selenium + geckodriver to render SPAs and dynamic content
- **Flexible Screenshots**: PNG screenshots with full-page mode plus configurable viewport size
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

### GET /, /healthz

Health check endpoint.

## Development

```bash
pip install -r requirements.txt
python server.py
```

## Docker

```bash
docker build -t screenshot-service .
docker run -p 11235:11235 screenshot-service
```

## Environment Variables

- `PORT`: Server port (default: 11235)
- `FIREFOX_BIN`: Path to Firefox binary (auto-detected)
- `GECKODRIVER_PATH`: Path to geckodriver (auto-detected)
- `CRAWLER_POOL_SIZE`: Number of Firefox instances to keep (default: 1)
- `CRAWLER_POOL_TIMEOUT`: Seconds to wait for a free instance (default: 10)
- `PAGE_LOAD_TIMEOUT`: Selenium page load timeout seconds (default: 30)
- `WAIT_TIME`: Extra seconds to wait before capture (default: 3)
- `MAX_RETRIES`: Retry count per request (default: 2)

## Architecture

The service consists of two main components:

1. **HTTP Server** (`server.py`): Handles incoming crawl requests
2. **Screenshot Utility** (`screenshot.py`): Manages Firefox instances and page rendering

## Deployment

This service is designed to be deployed as a separate microservice:

- **Kubernetes**: Use the provided Helm chart
- **Docker Compose**: See `.devcontainer/docker-compose.yml` for example
- **Standalone**: Run the service with required Firefox/geckodriver installation

## Performance

- **Memory**: ~200-250MB per concurrent browser instance
- **CPU**: Low when idle, spikes during page rendering
- **Startup**: ~2-4 seconds per browser launch
- **Concurrency**: Controlled by `CRAWLER_POOL_SIZE`

## License

MIT
