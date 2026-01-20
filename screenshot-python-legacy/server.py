import logging
import os
import queue
import threading
from typing import Optional, Tuple
from urllib.parse import parse_qs, urlencode

import requests
from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import PlainTextResponse, Response

from screenshot import ScreenshotUtility

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="Screenshot Service")


class UtilityPool:
    def __init__(self, size: int) -> None:
        self.size = max(size, 1)
        self.pool: queue.Queue[ScreenshotUtility] = queue.Queue(maxsize=self.size)
        self.created = 0
        self.lock = threading.Lock()

    def acquire(self, timeout: float) -> ScreenshotUtility:
        try:
            return self.pool.get_nowait()
        except queue.Empty:
            pass

        with self.lock:
            if self.created < self.size:
                self.created += 1
                return ScreenshotUtility(page_load_timeout=page_load_timeout())

        try:
            return self.pool.get(timeout=timeout)
        except queue.Empty as exc:
            raise TimeoutError("No crawler available") from exc

    def release(self, utility: Optional[ScreenshotUtility], healthy: bool) -> None:
        if not utility:
            return

        if not healthy:
            utility.close()
            with self.lock:
                if self.created > 0:
                    self.created -= 1
            return

        try:
            self.pool.put_nowait(utility)
        except queue.Full:
            utility.close()
            with self.lock:
                if self.created > 0:
                    self.created -= 1


def page_load_timeout() -> int:
    raw = os.getenv("PAGE_LOAD_TIMEOUT", "30").strip()
    try:
        value = int(raw)
        if value > 0:
            return value
    except ValueError:
        pass
    return 30


def crawl_wait_time() -> int:
    raw = os.getenv("WAIT_TIME", "3").strip()
    try:
        value = int(raw)
        if value >= 0:
            return value
    except ValueError:
        pass
    return 3


def max_retries() -> int:
    raw = os.getenv("MAX_RETRIES", "2").strip()
    try:
        value = int(raw)
        if value >= 0:
            return value
    except ValueError:
        pass
    return 2


def pool_size() -> int:
    raw = os.getenv("CRAWLER_POOL_SIZE", "1").strip()
    try:
        value = int(raw)
        if value > 0:
            return value
    except ValueError:
        pass
    return 1


def pool_timeout() -> float:
    raw = os.getenv("CRAWLER_POOL_TIMEOUT", "10").strip()
    try:
        value = float(raw)
        if value > 0:
            return value
    except ValueError:
        pass
    return 10.0


def status_timeout() -> float:
    raw = os.getenv("STATUS_TIMEOUT", "5").strip()
    try:
        value = float(raw)
        if value > 0:
            return value
    except ValueError:
        pass
    return 5.0


def fetch_status_code(url: str) -> Optional[int]:
    try:
        response = requests.get(
            url,
            allow_redirects=True,
            timeout=status_timeout(),
            stream=True,
            headers={"User-Agent": "Mozilla/5.0"},
        )
        response.close()
        return response.status_code
    except requests.Timeout as exc:
        raise HTTPException(status_code=504, detail=str(exc)) from exc
    except requests.RequestException as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc

def is_truthy(values: dict, key: str) -> bool:
    if key not in values:
        return False
    raw_list = values.get(key) or []
    if not raw_list:
        return True
    raw = raw_list[0].strip().lower()
    if raw == "":
        return True
    return raw in {"1", "true", "yes", "on"}


def pop_positive_int(values: dict, key: str, default: int) -> int:
    raw_list = values.pop(key, None)
    if not raw_list:
        return default
    raw = raw_list[0].strip()
    if raw == "":
        return default
    try:
        value = int(raw)
        if value > 0:
            return value
    except ValueError:
        pass
    return default


def extract_target(path: str, query: str) -> Tuple[str, str, str]:
    if query.endswith("/.png"):
        remaining_query = query[: -len("/.png")]
        return path, "png", remaining_query
    if query.endswith("/.html"):
        remaining_query = query[: -len("/.html")]
        return path, "html", remaining_query
    if path.endswith("/.png"):
        return path[: -len("/.png")], "png", query
    if path.endswith("/.html"):
        return path[: -len("/.html")], "html", query
    raise ValueError("Invalid format. Use: /.png or /.html")

def is_likely_non_html(url: str) -> bool:
    lower = url.lower()
    segments = [segment for segment in lower.split("/") if segment]
    for ext in (
        ".js",
        ".css",
        ".json",
        ".xml",
        ".png",
        ".jpg",
        ".jpeg",
        ".gif",
        ".webp",
        ".svg",
        ".ico",
        ".woff",
        ".woff2",
        ".ttf",
        ".otf",
        ".eot",
        ".pdf",
        ".zip",
        ".gz",
        ".tgz",
        ".rar",
        ".7z",
        ".mp4",
        ".mp3",
        ".avi",
        ".mov",
        ".m4a",
    ):
        if lower.endswith(ext):
            return True
        for segment in segments:
            if segment.endswith(ext):
                return True
    return False


pool = UtilityPool(pool_size())


@app.get("/health")
@app.head("/health")
@app.get("/healthz")
@app.head("/healthz")
def health_check() -> PlainTextResponse:
    return PlainTextResponse("ok")


@app.get("/readyz")
@app.head("/readyz")
def readiness_check() -> PlainTextResponse:
    utility = None
    healthy = True
    try:
        utility = pool.acquire(timeout=pool_timeout())
        healthy = utility.is_healthy()
    except Exception as exc:
        logger.warning("Readiness check failed: %s", exc)
        healthy = False
    finally:
        pool.release(utility, healthy)

    if not healthy:
        raise HTTPException(status_code=503, detail="Crawler unhealthy")
    return PlainTextResponse("ready")


@app.get("/")
def root() -> PlainTextResponse:
    return PlainTextResponse("Screenshot service running")


@app.get("/{path:path}")
def capture(path: str, request: Request) -> Response:
    if path in {"health", "healthz", "readyz"}:
        raise HTTPException(status_code=404, detail="Not found")

    query = request.url.query or ""
    try:
        target_url, fmt, remaining_query = extract_target(path, query)
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc

    if not target_url:
        raise HTTPException(status_code=400, detail="URL required")

    if not target_url.startswith("http://") and not target_url.startswith("https://"):
        target_url = f"https://{target_url}"

    if fmt == "html" and is_likely_non_html(target_url):
        raise HTTPException(status_code=415, detail="Target is not HTML")

    values = parse_qs(remaining_query, keep_blank_values=True)
    fullpage = is_truthy(values, "fullscreen") or is_truthy(values, "fullpage")
    values.pop("fullscreen", None)
    values.pop("fullpage", None)

    width = pop_positive_int(values, "width", 1920)
    height = pop_positive_int(values, "height", 1080)

    remaining_query = urlencode(values, doseq=True)
    if remaining_query:
        target_url = f"{target_url}?{remaining_query}"

    upstream_status = fetch_status_code(target_url)

    utility = None
    healthy = True
    try:
        utility = pool.acquire(timeout=pool_timeout())
        wait_time = crawl_wait_time()
        retries = max_retries()

        if fmt == "png":
            image = utility.capture_png(
                target_url,
                wait_time=wait_time,
                fullpage=fullpage,
                width=width,
                height=height,
                max_retries=retries,
            )
            if not image:
                if upstream_status and upstream_status >= 400:
                    raise HTTPException(
                        status_code=upstream_status,
                        detail=f"Upstream returned status {upstream_status}",
                    )
                raise HTTPException(status_code=500, detail="Failed to capture screenshot")
            return Response(
                content=image,
                media_type="image/png",
                status_code=upstream_status or 200,
            )

        html = utility.capture_html(
            target_url,
            wait_time=wait_time,
            width=width,
            height=height,
            max_retries=retries,
        )
        if html is None:
            if upstream_status and upstream_status >= 400:
                raise HTTPException(
                    status_code=upstream_status,
                    detail=f"Upstream returned status {upstream_status}",
                )
            raise HTTPException(status_code=500, detail="Failed to capture HTML")
        return Response(
            content=html,
            media_type="text/html; charset=utf-8",
            status_code=upstream_status or 200,
        )
    except HTTPException:
        healthy = False
        raise
    except Exception as exc:
        healthy = False
        logger.error("Error crawling %s: %s", target_url, exc)
        raise HTTPException(status_code=500, detail="Failed to crawl URL") from exc
    finally:
        pool.release(utility, healthy)


if __name__ == "__main__":
    import uvicorn

    port = int(os.getenv("PORT", "11235"))
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=port,
        workers=int(os.getenv("WORKERS", "1")),
        limit_concurrency=int(os.getenv("MAX_CONNECTIONS", "100")),
        timeout_keep_alive=int(os.getenv("KEEP_ALIVE", "5")),
    )
