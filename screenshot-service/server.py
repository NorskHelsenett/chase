import os
import base64
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import Optional
import uvicorn
from datetime import datetime
from fastapi.responses import JSONResponse

# Import your existing ScreenshotUtility
from screenshot import ScreenshotUtility

app = FastAPI(title="Screenshot Service")

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

# Initialize the screenshot utility
screenshot_util = ScreenshotUtility()

@app.get("/healthz")
@app.head("/healthz")
async def health_check():
    return JSONResponse(content={"status": "ok"})

@app.post("/screenshot")
async def take_screenshot(request: ScreenshotRequest):
    try:
        # Generate a unique filename
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        filename = f"{timestamp}_{request.url.replace('://', '_').replace('/', '_')}.png"

        # Take screenshot
        result = screenshot_util.take_screenshot(request.url, filename)
        
        if not result:
            raise HTTPException(status_code=500, detail="Failed to capture screenshot")

        # Read the screenshot and convert to base64
        with open(result, 'rb') as f:
            image_data = base64.b64encode(f.read()).decode('utf-8')

        # Clean up the file
        os.remove(result)

        return ScreenshotResponse(
            success=True,
            image=image_data,
            timestamp=datetime.now().isoformat(),
            url=request.url
        )

    except Exception as e:
        return ScreenshotResponse(
            success=False,
            error=str(e),
            timestamp=datetime.now().isoformat(),
            url=request.url
        )

if __name__ == "__main__":
    port = int(os.getenv("PORT", "8080"))
    uvicorn.run(app, host="0.0.0.0", port=port, log_level="info")
