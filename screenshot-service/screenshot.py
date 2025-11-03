import logging
import os
import shutil
import time
from os import devnull
from selenium import webdriver
from selenium.webdriver.firefox.service import Service
from selenium.webdriver.firefox.options import Options
from selenium.webdriver.support.ui import WebDriverWait
from selenium.common.exceptions import TimeoutException, WebDriverException
from typing import Optional
import tempfile

logger = logging.getLogger(__name__)

class ScreenshotUtility:
    def __init__(self, output_dir='screenshots', page_load_timeout=30):
        self.output_dir = output_dir
        self.driver = None
        self.page_load_timeout = page_load_timeout
        self.driver_initialized = False
        os.makedirs(output_dir, exist_ok=True)
        
        # Store options for potential re-initialization
        self.driver_path = None
        self.options = None
        self._init_driver()

    def _init_driver(self):
        """Initialize or reinitialize the Firefox driver"""
        options = Options()
        firefox_bin = os.getenv('FIREFOX_BIN')
        if not firefox_bin:
            firefox_bin = shutil.which('firefox')
        if firefox_bin:
            options.binary_location = firefox_bin
        
        # Headless and performance options
        options.add_argument('--headless')
        options.add_argument('--no-sandbox')
        options.add_argument('--disable-dev-shm-usage')
        options.add_argument('--disable-gpu')
        
        # Use temporary profile directory to avoid conflicts
        profile_dir = tempfile.mkdtemp(prefix='firefox_profile_')
        options.add_argument(f'-profile')
        options.add_argument(profile_dir)
        
        # Set preferences safely with error handling
        try:
            options.set_preference('javascript.enabled', True)
            options.set_preference('browser.display.use_document_fonts', 1)
            options.set_preference('gfx.downloadable_fonts.enabled', True)
            options.set_preference('permissions.default.image', 1)
            options.set_preference('webdriver.log.file', '/dev/null')
            # Network resilience preferences
            options.set_preference('network.http.connection-timeout', 30)
            options.set_preference('network.http.connection-retry-timeout', 30)
            options.set_preference('dom.disable_beforeunload', True)
        except Exception as e:
            logger.warning(f"Error setting Firefox preferences: {e}")

        driver_path = os.getenv('GECKODRIVER_PATH')
        if driver_path and not os.path.exists(driver_path):
            raise Exception(f"GECKODRIVER_PATH set but file not found: {driver_path}")
        if not driver_path:
            driver_path = shutil.which('geckodriver')
        if not driver_path:
            raise Exception("geckodriver not found. Set GECKODRIVER_PATH or ensure it is in PATH")

        self.driver_path = driver_path
        self.options = options

        try:
            logger.debug("Starting Firefox with geckodriver at %s", driver_path)
            service = Service(driver_path, service_log_path=devnull)
            self.driver = webdriver.Firefox(service=service, options=options)
            self.driver.set_page_load_timeout(self.page_load_timeout)
            self.driver.set_window_size(1920, 1080)
            self.driver_initialized = True
        except Exception as e:
            self.driver_initialized = False
            raise Exception(f"Failed to initialize Firefox driver: {e}")
    
    def _ensure_driver(self):
        """Ensure driver is initialized and working, reinitialize if needed"""
        if not self.driver or not self.driver_initialized:
            logger.info("Reinitializing Firefox driver")
            try:
                if self.driver:
                    self.driver.quit()
            except:
                pass
            self._init_driver()

    def take_screenshot(self, url, filename=None, wait_time=0, fullpage=False, max_retries=2):
        """Take a screenshot with retry logic and better error handling"""
        self._ensure_driver()
        
        if not self.driver:
            raise Exception("Driver not initialized")

        for attempt in range(max_retries + 1):
            try:
                if not filename:
                    filename = url.replace('://', '_').replace('/', '_') + '.png'
                filepath = os.path.join(self.output_dir, filename)

                logger.info(f"Attempting to screenshot {url} (attempt {attempt + 1}/{max_retries + 1})")
                
                # Navigate to URL
                try:
                    self.driver.get(url)
                except WebDriverException as e:
                    error_msg = str(e).lower()
                    # Check for DNS and network errors
                    if any(err in error_msg for err in ['dnsnotfound', 'neterror', 'net::err_name_not_resolved', 
                                                         'connection refused', 'timeout']):
                        logger.warning(f"Network/DNS error for {url}: {e}")
                        raise Exception(f"Network error: Unable to reach {url}. Check DNS or network connectivity.")
                    raise

                # Wait for page to be ready
                ready_timeout = max(wait_time, self.page_load_timeout, 1)
                try:
                    WebDriverWait(self.driver, ready_timeout).until(
                        lambda driver: driver.execute_script('return document.readyState') == 'complete'
                    )
                except TimeoutException:
                    logger.warning(f"Timeout waiting for page ready state for {url}")
                    # Continue anyway - partial page might be better than nothing

                # Wait for images to load (optional, don't fail if timeout)
                try:
                    WebDriverWait(self.driver, max(wait_time, 3)).until(
                        lambda d: d.execute_script("""
                            let images = document.getElementsByTagName('img');
                            return Array.from(images).every(img => img.complete);
                        """)
                    )
                except TimeoutException:
                    logger.warning(f"Timeout waiting for images to load for {url}")

                # Additional wait time if specified
                if wait_time > 0:
                    time.sleep(wait_time)

                # Take the screenshot
                try:
                    if fullpage and hasattr(self.driver, "get_full_page_screenshot_as_file"):
                        if not self.driver.get_full_page_screenshot_as_file(filepath):
                            raise Exception("Full page screenshot capture failed")
                    else:
                        if not self.driver.save_screenshot(filepath):
                            raise Exception("Standard screenshot capture failed")
                except WebDriverException as e:
                    logger.error(f"Screenshot save failed: {e}")
                    if attempt < max_retries:
                        logger.info("Reinitializing driver and retrying...")
                        self.driver_initialized = False
                        continue
                    raise

                # Verify screenshot was created
                if os.path.exists(filepath) and os.path.getsize(filepath) > 0:
                    logger.info(f"Successfully captured screenshot of {url}")
                    return filepath
                else:
                    raise Exception("Screenshot file empty or missing")

            except Exception as e:
                if attempt < max_retries:
                    logger.warning(f"Screenshot attempt {attempt + 1} failed, retrying: {e}")
                    time.sleep(1)  # Brief delay before retry
                    continue
                else:
                    logger.error(f"Failed to capture screenshot of {url} after {max_retries + 1} attempts", exc_info=True)
                    return None
        
        return None

    def close(self):
        """Safely close the driver"""
        if self.driver:
            try:
                self.driver.quit()
            except Exception as e:
                logger.warning(f"Error closing driver: {e}")
            finally:
                self.driver = None
                self.driver_initialized = False

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()

if __name__ == "__main__":
    url = os.getenv('url', 'https://nhn.no')
    screenshot = ScreenshotUtility()
    result = screenshot.take_screenshot(url)
    if result:
        print(f"Screenshot saved to: {result}")
