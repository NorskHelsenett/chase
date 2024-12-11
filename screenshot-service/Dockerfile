FROM python:3.11-alpine

RUN apk add --no-cache \
    firefox \
    xvfb \
    wget \
    dbus-x11 \
    ttf-liberation \
    mesa-gl \
    pciutils-libs \
    && wget https://github.com/mozilla/geckodriver/releases/download/v0.33.0/geckodriver-v0.33.0-linux64.tar.gz \
    && tar -xzf geckodriver-v0.33.0-linux64.tar.gz \
    && mv geckodriver /usr/local/bin/ \
    && chmod +x /usr/local/bin/geckodriver \
    && rm geckodriver-v0.33.0-linux64.tar.gz

# Add user appuser with UID and GID 1001
RUN addgroup -g 1001 appuser && \
    adduser -D -u 1001 -G appuser appuser && \
    # Give appuser access to firefox and related directories
    chown -R appuser:appuser /usr/bin/firefox && \
    chown appuser:appuser /usr/local/bin/geckodriver && \
    mkdir -p /home/appuser/.mozilla && \
    chown -R appuser:appuser /home/appuser && \
    # Create and set permissions for X11 directory
    mkdir -p /tmp/.X11-unix && \
    chmod 1777 /tmp/.X11-unix && \
    # Add write permission to /dev/shm for Selenium/Firefox
    chmod 1777 /dev/shm

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY screenshot.py .
COPY server.py .

ENV DISPLAY=:99
ENV MOZ_HEADLESS=1
ENV FIREFOX_BIN=/usr/bin/firefox
ENV PORT=8080

# Create screenshots directory with proper permissions
RUN mkdir -p screenshots && chown -R appuser:appuser screenshots && chmod 777 screenshots

# Switch to appuser
USER appuser

# Start Xvfb and the FastAPI server
CMD ["sh", "-c", "Xvfb :99 -screen 0 1920x1080x24 -ac +extension GLX +render & uvicorn server:app --host 0.0.0.0 --port 8080"]

EXPOSE 8080