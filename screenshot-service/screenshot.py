import os
from os import devnull
import time
from selenium import webdriver
from selenium.webdriver.firefox.service import Service
from selenium.webdriver.firefox.options import Options
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.common.by import By

class ScreenshotUtility:
    def __init__(self, output_dir='screenshots', page_load_timeout=30):
        self.output_dir = output_dir
        self.driver = None
        os.makedirs(output_dir, exist_ok=True)
        
        options = Options()
        options.binary_location = os.getenv('FIREFOX_BIN', '/usr/bin/firefox')
        options.add_argument('--headless')
        options.add_argument('--no-sandbox')
        options.add_argument('--disable-dev-shm-usage')
        options.add_argument('--disable-gpu')
        options.set_preference('javascript.enabled', True)
        options.set_preference('font.name.sans-serif', 'Liberation Sans')
        options.set_preference('permissions.default.image', 1)
        options.set_preference('webdriver.log.file', '/dev/null')
        
        try:
            service = Service('/usr/local/bin/geckodriver', service_log_path=devnull)
            self.driver = webdriver.Firefox(service=service, options=options)
            self.driver.set_page_load_timeout(page_load_timeout)
            self.driver.set_window_size(1920, 1080)
        except Exception as e:
            raise Exception(f"Failed to initialize Firefox driver: {e}")

    def take_screenshot(self, url, filename=None):
        if not self.driver:
            raise Exception("Driver not initialized")
            
        try:
            if not filename:
                filename = url.replace('://', '_').replace('/', '_') + '.png'
            filepath = os.path.join(self.output_dir, filename)
            
            self.driver.get(url)
            
            WebDriverWait(self.driver, 10).until(
                lambda driver: driver.execute_script('return document.readyState') == 'complete'
            )
            
            try:
                WebDriverWait(self.driver, 10).until(
                    lambda d: d.execute_script("""
                        let images = document.getElementsByTagName('img');
                        return Array.from(images).every(img => img.complete);
                    """)
                )
            except:
                print("Warning: Timeout waiting for images")
            
            time.sleep(2)
            
            self.driver.save_screenshot(filepath)
            
            if os.path.exists(filepath) and os.path.getsize(filepath) > 0:
                return filepath
            raise Exception("Screenshot file empty or missing")
            
        except Exception as e:
            print(f"Failed to capture screenshot of {url}: {e}")
            return None
            
    def __del__(self):
        if self.driver:
            self.driver.quit()

if __name__ == "__main__":
    url = os.getenv('url', 'https://example.com')
    screenshot = ScreenshotUtility()
    result = screenshot.take_screenshot(url)
    if result:
        print(f"Screenshot saved to: {result}")