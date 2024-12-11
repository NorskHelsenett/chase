FROM python:3.11-alpine

RUN apk add --no-cache \
    firefox \
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
    # Add write permission to /dev/shm for Selenium/Firefox
    chmod 1777 /dev/shm

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY screenshot.py .
COPY server.py .

ENV MOZ_HEADLESS=1
ENV FIREFOX_BIN=/usr/bin/firefox
ENV PORT=8080

# Create screenshots directory with proper permissions
RUN mkdir -p screenshots && chown -R appuser:appuser screenshots && chmod 777 screenshots

# Switch to appuser
USER appuser

# Start just the FastAPI server
CMD ["uvicorn", "server:app", "--host", "0.0.0.0", "--port", "8080"]

EXPOSE 8080