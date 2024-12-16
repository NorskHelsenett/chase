import os
import base64
import logging
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, HttpUrl
from typing import Optional
import uvicorn
from datetime import datetime
from fastapi.responses import JSONResponse
from contextlib import contextmanager

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
    wait_time: Optional[int] = 2
    fullpage: Optional[bool] = True

    model_config = {
        "json_schema_extra": {
            "example": {
                "url": "https://example.com",
                "width": 1920,
                "height": 1080,
                "wait_time": 2,
                "fullpage": True
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
screenshot_utils = {}

@contextmanager
def get_screenshot_utility():
    """Context manager for handling screenshot utility lifecycle"""
    worker_id = os.getpid()
    try:
        if worker_id not in screenshot_utils:
            screenshot_utils[worker_id] = ScreenshotUtility()
        yield screenshot_utils[worker_id]
    except Exception as e:
        logger.error(f"Error with screenshot utility: {e}")
        # Clean up the failed instance
        if worker_id in screenshot_utils:
            del screenshot_utils[worker_id]
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

        with get_screenshot_utility() as screenshot_util:
            # Configure browser window size if specified
            if request.width and request.height:
                screenshot_util.driver.set_window_size(request.width, request.height)

            # Take screenshot
            filepath = screenshot_util.take_screenshot(str(request.url), filename)

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

    except Exception as e:
        logger.error(f"Screenshot error: {e}")
        return ScreenshotResponse(
            success=False,
            error=str(e),
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
    for worker_id, screenshot_util in screenshot_utils.items():
        try:
            screenshot_util.driver.quit()
        except Exception as e:
            logger.error(f"Error shutting down worker {worker_id}: {e}")
    screenshot_utils.clear()

if __name__ == "__main__":
    cpu_count = len(os.sched_getaffinity(0))
    recommended_workers = min(cpu_count * 2, 8)  # Cap at 8 workers
    workers = int(os.getenv("WORKERS", recommended_workers))
    port = int(os.getenv("PORT", "8080"))

    logger.info(f"Starting with {workers} workers on port {port}")
    uvicorn.run(
        "app:app",
        host="0.0.0.0",
        port=port,
        workers=workers,
        loop="uvloop",
        limit_concurrency=int(os.getenv("MAX_CONNECTIONS", "100")),
        timeout_keep_alive=int(os.getenv("KEEP_ALIVE", "5"))
    )