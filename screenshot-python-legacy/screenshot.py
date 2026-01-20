import logging
import os
import shutil
import tempfile
import time
from os import devnull
from typing import Optional

from selenium import webdriver
from selenium.common.exceptions import TimeoutException, WebDriverException
from selenium.webdriver.firefox.options import Options
from selenium.webdriver.firefox.service import Service
from selenium.webdriver.support.ui import WebDriverWait

logger = logging.getLogger(__name__)


class ScreenshotUtility:
    def __init__(self, page_load_timeout: int = 30) -> None:
        self.driver = None
        self.page_load_timeout = page_load_timeout
        self.driver_initialized = False
        self.profile_dir: Optional[str] = None
        self.driver_path: Optional[str] = None
        self.options: Optional[Options] = None
        self._init_driver()

    def _init_driver(self) -> None:
        options = Options()
        firefox_bin = os.getenv("FIREFOX_BIN") or shutil.which("firefox")
        if firefox_bin:
            options.binary_location = firefox_bin

        options.add_argument("--headless")
        options.add_argument("--no-sandbox")
        options.add_argument("--disable-dev-shm-usage")
        options.add_argument("--disable-gpu")

        profile_dir = tempfile.mkdtemp(prefix="firefox_profile_")
        options.add_argument("-profile")
        options.add_argument(profile_dir)
        self.profile_dir = profile_dir

        try:
            options.set_preference("javascript.enabled", True)
            options.set_preference("browser.display.use_document_fonts", 1)
            options.set_preference("gfx.downloadable_fonts.enabled", True)
            options.set_preference("permissions.default.image", 1)
            options.set_preference("webdriver.log.file", "/dev/null")
            options.set_preference("network.http.connection-timeout", 30)
            options.set_preference("network.http.connection-retry-timeout", 30)
            options.set_preference("dom.disable_beforeunload", True)
        except Exception as exc:
            logger.warning("Error setting Firefox preferences: %s", exc)

        driver_path = os.getenv("GECKODRIVER_PATH")
        if driver_path and not os.path.exists(driver_path):
            raise Exception(f"GECKODRIVER_PATH set but file not found: {driver_path}")
        if not driver_path:
            driver_path = shutil.which("geckodriver")
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
        except Exception as exc:
            self.driver_initialized = False
            self._cleanup_profile()
            raise Exception(f"Failed to initialize Firefox driver: {exc}") from exc

    def _cleanup_profile(self) -> None:
        if self.profile_dir:
            try:
                shutil.rmtree(self.profile_dir, ignore_errors=True)
            except Exception:
                pass
            self.profile_dir = None

    def _ensure_driver(self) -> None:
        if not self.driver or not self.driver_initialized:
            logger.info("Reinitializing Firefox driver")
            try:
                if self.driver:
                    self.driver.quit()
            except Exception:
                pass
            self._cleanup_profile()
            self._init_driver()

    def _load_page(self, url: str, wait_time: int) -> None:
        try:
            self.driver.get(url)
        except WebDriverException as exc:
            error_msg = str(exc).lower()
            if any(
                err in error_msg
                for err in (
                    "dnsnotfound",
                    "neterror",
                    "net::err_name_not_resolved",
                    "connection refused",
                    "timeout",
                )
            ):
                logger.warning("Network/DNS error for %s: %s", url, exc)
                raise Exception(
                    f"Network error: Unable to reach {url}. Check DNS or network connectivity."
                ) from exc
            raise

        ready_timeout = max(wait_time, self.page_load_timeout, 1)
        try:
            WebDriverWait(self.driver, ready_timeout).until(
                lambda driver: driver.execute_script("return document.readyState") == "complete"
            )
        except TimeoutException:
            logger.warning("Timeout waiting for page ready state for %s", url)

        try:
            WebDriverWait(self.driver, max(wait_time, 3)).until(
                lambda d: d.execute_script(
                    """
                    let images = document.getElementsByTagName('img');
                    return Array.from(images).every(img => img.complete);
                    """
                )
            )
        except TimeoutException:
            logger.warning("Timeout waiting for images to load for %s", url)

        if wait_time > 0:
            time.sleep(wait_time)

    def capture_png(
        self,
        url: str,
        wait_time: int = 0,
        fullpage: bool = False,
        width: int = 1920,
        height: int = 1080,
        max_retries: int = 2,
    ) -> Optional[bytes]:
        for attempt in range(max_retries + 1):
            try:
                self._ensure_driver()

                if width and height:
                    try:
                        self.driver.set_window_size(width, height)
                    except Exception as exc:
                        logger.warning("Failed to set window size: %s", exc)

                logger.info(
                    "Attempting to screenshot %s (attempt %d/%d)",
                    url,
                    attempt + 1,
                    max_retries + 1,
                )
                self._load_page(url, wait_time)

                if fullpage and hasattr(self.driver, "get_full_page_screenshot_as_png"):
                    image = self.driver.get_full_page_screenshot_as_png()
                else:
                    image = self.driver.get_screenshot_as_png()

                if not image:
                    raise Exception("Screenshot data empty")

                logger.info("Successfully captured screenshot of %s", url)
                return image
            except Exception as exc:
                if attempt < max_retries:
                    logger.warning("Screenshot attempt %d failed, retrying: %s", attempt + 1, exc)
                    time.sleep(1)
                    self.driver_initialized = False
                    continue
                logger.error(
                    "Failed to capture screenshot of %s after %d attempts",
                    url,
                    max_retries + 1,
                    exc_info=True,
                )
                return None
        return None

    def capture_html(
        self,
        url: str,
        wait_time: int = 0,
        width: int = 1920,
        height: int = 1080,
        max_retries: int = 2,
    ) -> Optional[str]:
        for attempt in range(max_retries + 1):
            try:
                self._ensure_driver()

                if width and height:
                    try:
                        self.driver.set_window_size(width, height)
                    except Exception as exc:
                        logger.warning("Failed to set window size: %s", exc)

                logger.info(
                    "Attempting to capture HTML %s (attempt %d/%d)",
                    url,
                    attempt + 1,
                    max_retries + 1,
                )
                self._load_page(url, wait_time)
                html = self.driver.page_source
                if not html:
                    raise Exception("HTML empty")
                return html
            except Exception as exc:
                if attempt < max_retries:
                    logger.warning("HTML capture attempt %d failed, retrying: %s", attempt + 1, exc)
                    time.sleep(1)
                    self.driver_initialized = False
                    continue
                logger.error(
                    "Failed to capture HTML of %s after %d attempts",
                    url,
                    max_retries + 1,
                    exc_info=True,
                )
                return None
        return None

    def is_healthy(self) -> bool:
        if not self.driver or not self.driver_initialized:
            return False
        try:
            _ = self.driver.title
            return True
        except Exception:
            return False

    def close(self) -> None:
        if self.driver:
            try:
                self.driver.quit()
            except Exception as exc:
                logger.warning("Error closing driver: %s", exc)
            finally:
                self.driver = None
                self.driver_initialized = False
                self._cleanup_profile()

    def __enter__(self) -> "ScreenshotUtility":
        return self

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:
        self.close()
