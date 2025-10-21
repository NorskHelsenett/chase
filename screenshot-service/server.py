import asyncio
import os
import base64
import logging
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, HttpUrl
from typing import Optional, Dict, Any
import uvicorn
from datetime import datetime
from fastapi.responses import JSONResponse
from contextlib import asynccontextmanager

# Import your existing ScreenshotUtility
from screenshot import ScreenshotUtility

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="Screenshot Service")

class ScreenshotRequest(BaseModel):
    url: HttpUrl  # Using HttpUrl for URL validation
    width: Optional[int] = 1920
    height: Optional[int] = 1080
    wait_time: Optional[int] = 0
    fullpage: Optional[bool] = False

    model_config = {
        "json_schema_extra": {
            "example": {
                "url": "https://example.com",
                "width": 1920,
                "height": 1080,
                "wait_time": 0,
                "fullpage": False
            }
        }
    }

class ScreenshotResponse(BaseModel):
    success: bool
    image: Optional[str] = None  # base64 encoded image
    error: Optional[str] = None
    timestamp: str
    url: str

# Screenshot utility pool
screenshot_utils: Dict[int, Dict[str, Any]] = {}
pool_lock = asyncio.Lock()

@asynccontextmanager
async def get_screenshot_utility():
    """Async context manager for handling screenshot utility lifecycle"""
    worker_id = os.getpid()
    async with pool_lock:
        entry = screenshot_utils.get(worker_id)
        if not entry:
            entry = {
                "utility": ScreenshotUtility(),
                "lock": asyncio.Lock()
            }
            screenshot_utils[worker_id] = entry

    utility = entry["utility"]
    utility_lock: asyncio.Lock = entry["lock"]

    async with utility_lock:
        try:
            yield utility
        except Exception as exc:
            if not isinstance(exc, HTTPException):
                logger.error("Error with screenshot utility", exc_info=True)
                async with pool_lock:
                    screenshot_utils.pop(worker_id, None)
                utility.close()
            raise

def cleanup_screenshot(filepath: str) -> None:
    """Safely cleanup screenshot file"""
    try:
        if filepath and os.path.exists(filepath):
            os.remove(filepath)
    except Exception as e:
        logger.error(f"Error cleaning up file {filepath}: {e}")

@app.get("/healthz")
@app.head("/healthz")
async def health_check():
    return JSONResponse(content={"status": "ok"})

@app.post("/screenshot", response_model=ScreenshotResponse)
async def take_screenshot(request: ScreenshotRequest):
    filepath = None
    try:
        # Generate a unique filename with sanitized URL
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        safe_url = str(request.url).replace('://', '_').replace('/', '_')[:100]  # Limit length
        filename = f"{timestamp}_{safe_url}.png"

        async with get_screenshot_utility() as screenshot_util:
            # Configure browser window size if specified
            if request.width and request.height:
                screenshot_util.driver.set_window_size(request.width, request.height)

            # Take screenshot
            wait_time = request.wait_time if request.wait_time is not None else 0
            fullpage = True if request.fullpage is None else request.fullpage
            filepath = screenshot_util.take_screenshot(
                str(request.url),
                filename,
                wait_time=wait_time,
                fullpage=fullpage
            )

            if not filepath:
                raise HTTPException(status_code=500, detail="Failed to capture screenshot")

            # Read the screenshot and convert to base64
            with open(filepath, 'rb') as f:
                image_data = base64.b64encode(f.read()).decode('utf-8')

            return ScreenshotResponse(
                success=True,
                image=image_data,
                timestamp=datetime.now().isoformat(),
                url=str(request.url)
            )

    except Exception as exc:
        logger.error("Screenshot error", exc_info=True)
        return ScreenshotResponse(
            success=False,
            error=str(exc),
            timestamp=datetime.now().isoformat(),
            url=str(request.url)
        )

    finally:
        # Clean up the screenshot file
        if filepath:
            cleanup_screenshot(filepath)

@app.on_event("shutdown")
async def shutdown_event():
    """Cleanup resources on shutdown"""
    async with pool_lock:
        entries = list(screenshot_utils.items())
        screenshot_utils.clear()

    for worker_id, entry in entries:
        utility: ScreenshotUtility = entry["utility"]
        try:
            utility.close()
        except Exception:
            logger.error(f"Error shutting down worker {worker_id}", exc_info=True)

if __name__ == "__main__":
    try:
        cpu_count = len(os.sched_getaffinity(0))
    except (AttributeError, NotImplementedError):
        cpu_count = os.cpu_count() or 1
    recommended_workers = min(cpu_count * 2, 8)  # Cap at 8 workers
    workers = int(os.getenv("WORKERS", recommended_workers))
    port = int(os.getenv("PORT", "8080"))

    logger.info(f"Starting with {workers} workers on port {port}")
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=port,
        workers=workers,
        loop="uvloop",
        limit_concurrency=int(os.getenv("MAX_CONNECTIONS", "100")),
        timeout_keep_alive=int(os.getenv("KEEP_ALIVE", "5"))
    )
