import os
import base64
from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel
from typing import Optional
import uvicorn
from datetime import datetime
from fastapi.responses import JSONResponse
from contextlib import asynccontextmanager
import uuid
from typing import Dict, Any
import asyncio
from concurrent.futures import ThreadPoolExecutor
import logging

# Import your existing ScreenshotUtility
from screenshot import ScreenshotUtility

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configure max concurrent screenshots
MAX_CONCURRENT_SCREENSHOTS = int(os.getenv("MAX_CONCURRENT_SCREENSHOTS", "3"))
screenshot_semaphore = asyncio.Semaphore(MAX_CONCURRENT_SCREENSHOTS)

class ScreenshotRequest(BaseModel):
    url: str
    width: Optional[int] = 1920
    height: Optional[int] = 1080
    wait_time: Optional[int] = 2
    fullpage: Optional[bool] = True

class ScreenshotResponse(BaseModel):
    success: bool
    image: Optional[str] = None  # base64 encoded image
    error: Optional[str] = None
    timestamp: str
    url: str
    request_id: str

class ScreenshotManager:
    def __init__(self):
        self.executor = ThreadPoolExecutor(max_workers=MAX_CONCURRENT_SCREENSHOTS)
        self.active_tasks: Dict[str, Any] = {}

    async def cleanup_old_files(self, background_tasks: BackgroundTasks):
        """Clean up old screenshot files that weren't properly deleted"""
        screenshots_dir = "screenshots"
        if os.path.exists(screenshots_dir):
            for file in os.listdir(screenshots_dir):
                try:
                    file_path = os.path.join(screenshots_dir, file)
                    if os.path.isfile(file_path):
                        background_tasks.add_task(os.remove, file_path)
                except Exception as e:
                    logger.error(f"Error cleaning up file {file}: {e}")

    async def take_screenshot(self, request: ScreenshotRequest) -> Dict[str, Any]:
        """Take screenshot in a separate thread"""
        async with screenshot_semaphore:
            try:
                # Create a new ScreenshotUtility instance for each request
                screenshot_util = ScreenshotUtility()

                # Generate unique ID and filename
                request_id = str(uuid.uuid4())
                filename = f"{request_id}.png"

                # Take screenshot in thread pool
                result = await asyncio.get_event_loop().run_in_executor(
                    self.executor,
                    screenshot_util.take_screenshot,
                    request.url,
                    filename
                )

                if not result:
                    raise Exception("Failed to capture screenshot")

                # Read the screenshot
                with open(result, 'rb') as f:
                    image_data = base64.b64encode(f.read()).decode('utf-8')

                # Clean up
                try:
                    os.remove(result)
                except Exception as e:
                    logger.error(f"Error removing file {result}: {e}")

                return {
                    "success": True,
                    "image": image_data,
                    "request_id": request_id,
                    "error": None
                }

            except Exception as e:
                logger.error(f"Screenshot error: {e}")
                return {
                    "success": False,
                    "image": None,
                    "request_id": request_id,
                    "error": str(e)
                }
            finally:
                # Ensure browser is closed
                try:
                    screenshot_util.driver.quit()
                except:
                    pass

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Initialize screenshot manager
    app.state.screenshot_manager = ScreenshotManager()
    yield
    # Cleanup on shutdown
    app.state.screenshot_manager.executor.shutdown(wait=True)

app = FastAPI(title="Screenshot Service", lifespan=lifespan)

@app.get("/healthz")
@app.head("/healthz")
async def health_check():
    return JSONResponse(content={"status": "ok"})

@app.post("/screenshot")
async def take_screenshot(
    request: ScreenshotRequest,
    background_tasks: BackgroundTasks
):
    # Clean up any old files
    await app.state.screenshot_manager.cleanup_old_files(background_tasks)

    # Take screenshot
    result = await app.state.screenshot_manager.take_screenshot(request)

    return ScreenshotResponse(
        success=result["success"],
        image=result["image"],
        error=result["error"],
        timestamp=datetime.now().isoformat(),
        url=request.url,
        request_id=result["request_id"]
    )

def start_server():
    """Start the server with proper worker configuration"""
    port = int(os.getenv("PORT", "8080"))
    workers = int(os.getenv("WORKERS", "2"))

    config = uvicorn.Config(
        app,
        host="0.0.0.0",
        port=port,
        workers=workers,
        loop="uvloop",
        limit_concurrency=int(os.getenv("MAX_CONNECTIONS", "100")),
        timeout_keep_alive=int(os.getenv("KEEP_ALIVE", "5")),
        log_level="info"
    )
    server = uvicorn.Server(config)
    server.run()

if __name__ == "__main__":
    start_server()