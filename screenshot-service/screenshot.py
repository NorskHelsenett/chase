import os
import sys
import time
from selenium import webdriver
from selenium.webdriver.firefox.service import Service
from selenium.webdriver.firefox.options import Options
from selenium.webdriver.support.ui import WebDriverWait

class ScreenshotUtility:
    def __init__(self, output_dir='screenshots', page_load_timeout=30):
        """
        Initialize with GPU disabled but JavaScript enabled.
        """
        self.output_dir = output_dir
        os.makedirs(output_dir, exist_ok=True)

        options = Options()

        # Disable GPU but keep JavaScript
        options.add_argument('--headless')
        options.add_argument('--no-sandbox')
        options.add_argument('--disable-dev-shm-usage')
        options.add_argument('--disable-gpu')
        options.add_argument('--window-size=1920,1080')

        # Explicitly enable JavaScript
        options.set_preference('javascript.enabled', True)

        # Performance preferences
        options.set_preference('permissions.default.image', 1)
        options.set_preference('browser.cache.disk.enable', False)
        options.set_preference('browser.cache.memory.enable', True)
        options.set_preference('browser.cache.offline.enable', False)
        options.set_preference('network.http.use-cache', False)

        # Fast fail if Firefox binary not found
        firefox_bin = os.getenv('FIREFOX_BIN', '/usr/bin/firefox')
        if not os.path.exists(firefox_bin):
            print("Firefox binary not found", file=sys.stderr)
            sys.exit(1)

        options.binary_location = firefox_bin

        # Fast fail if geckodriver not found
        geckodriver_path = '/usr/local/bin/geckodriver'
        if not os.path.exists(geckodriver_path):
            print("Geckodriver not found", file=sys.stderr)
            sys.exit(1)

        service = Service(geckodriver_path)
        self.driver = webdriver.Firefox(service=service, options=options)
        self.driver.set_page_load_timeout(page_load_timeout)

    def take_screenshot(self, url, filename=None):
        """
        Take screenshot with JavaScript enabled.
        """
        if not filename:
            filename = f"{url.replace('://', '_').replace('/', '_')}.png"
        filepath = os.path.join(self.output_dir, filename)

        self.driver.get(url)

        # Wait for page load since JavaScript is enabled
        try:
            WebDriverWait(self.driver, 10).until(
                lambda d: d.execute_script('return document.readyState') == 'complete'
            )

            # Wait for images
            WebDriverWait(self.driver, 10).until(
                lambda d: d.execute_script("""
                    return Array.from(document.getElementsByTagName('img')).every(
                        img => img.complete && img.naturalHeight !== 0
                    );
                """)
            )
        except Exception as e:
            print(f"Warning: Timeout waiting for page load: {e}", file=sys.stderr)

        # Small pause to ensure final rendering
        time.sleep(2)

        self.driver.save_screenshot(filepath)

        # Verify screenshot exists and has content or crash
        if not os.path.exists(filepath) or os.path.getsize(filepath) == 0:
            print("Screenshot failed - empty or missing file", file=sys.stderr)
            sys.exit(1)

        return filepath

    def __del__(self):
        """Ensure driver is closed on exit"""
        try:
            if hasattr(self, 'driver'):
                self.driver.quit()
        except:
            pass

if __name__ == "__main__":
    url = os.getenv('URL', 'https://example.com')
    try:
        screenshot = ScreenshotUtility(
            page_load_timeout=int(os.getenv('PAGE_LOAD_TIMEOUT', '30'))
        )
        result = screenshot.take_screenshot(url)
    except Exception as e:
        print(f"Fatal error: {e}", file=sys.stderr)
        sys.exit(1)