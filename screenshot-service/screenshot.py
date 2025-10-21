import logging
import os
import shutil
import time
from os import devnull
from selenium import webdriver
from selenium.webdriver.firefox.service import Service
from selenium.webdriver.firefox.options import Options
from selenium.webdriver.support.ui import WebDriverWait
from selenium.common.exceptions import TimeoutException

logger = logging.getLogger(__name__)

class ScreenshotUtility:
    def __init__(self, output_dir='screenshots', page_load_timeout=30):
        self.output_dir = output_dir
        self.driver = None
        self.page_load_timeout = page_load_timeout
        os.makedirs(output_dir, exist_ok=True)

        options = Options()
        firefox_bin = os.getenv('FIREFOX_BIN')
        if not firefox_bin:
            firefox_bin = shutil.which('firefox')
        if firefox_bin:
            options.binary_location = firefox_bin
        options.add_argument('--headless')
        options.add_argument('--no-sandbox')
        options.add_argument('--disable-dev-shm-usage')
        options.add_argument('--disable-gpu')
        options.set_preference('javascript.enabled', True)
        options.set_preference('browser.display.use_document_fonts', 1)
        options.set_preference('gfx.downloadable_fonts.enabled', True)
        options.set_preference('permissions.default.image', 1)
        options.set_preference('webdriver.log.file', '/dev/null')

        driver_path = os.getenv('GECKODRIVER_PATH')
        if driver_path and not os.path.exists(driver_path):
            raise Exception(f"GECKODRIVER_PATH set but file not found: {driver_path}")
        if not driver_path:
            driver_path = shutil.which('geckodriver')
        if not driver_path:
            raise Exception("geckodriver not found. Set GECKODRIVER_PATH or ensure it is in PATH")

        try:
            logger.debug("Starting Firefox with geckodriver at %s", driver_path)
            service = Service(driver_path, service_log_path=devnull)
            self.driver = webdriver.Firefox(service=service, options=options)
            self.driver.set_page_load_timeout(page_load_timeout)
            self.driver.set_window_size(1920, 1080)
        except Exception as e:
            raise Exception(f"Failed to initialize Firefox driver: {e}")

    def take_screenshot(self, url, filename=None, wait_time=0, fullpage=False):
        if not self.driver:
            raise Exception("Driver not initialized")

        try:
            if not filename:
                filename = url.replace('://', '_').replace('/', '_') + '.png'
            filepath = os.path.join(self.output_dir, filename)

            self.driver.get(url)

            ready_timeout = max(wait_time, self.page_load_timeout, 1)
            WebDriverWait(self.driver, ready_timeout).until(
                lambda driver: driver.execute_script('return document.readyState') == 'complete'
            )

            try:
                WebDriverWait(self.driver, max(wait_time, 1)).until(
                    lambda d: d.execute_script("""
                        let images = document.getElementsByTagName('img');
                        return Array.from(images).every(img => img.complete);
                    """)
                )
            except TimeoutException:
                logger.warning("Timeout waiting for images to load for %s", url)

            if wait_time > 0:
                time.sleep(wait_time)

            if fullpage and hasattr(self.driver, "get_full_page_screenshot_as_file"):
                if not self.driver.get_full_page_screenshot_as_file(filepath):
                    raise Exception("Full page screenshot capture failed")
            else:
                if not self.driver.save_screenshot(filepath):
                    raise Exception("Standard screenshot capture failed")

            if os.path.exists(filepath) and os.path.getsize(filepath) > 0:
                return filepath
            raise Exception("Screenshot file empty or missing")

        except Exception:
            logger.error("Failed to capture screenshot of %s", url, exc_info=True)
            return None

    def close(self):
        if self.driver:
            self.driver.quit()
            self.driver = None

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
