FROM python:3.11-alpine

# Install Firefox and dependencies
RUN apk add --no-cache \
    firefox-esr \
    dbus \
    ttf-liberation \
    mesa-gl \
    pciutils-libs \
    bash \
    && wget https://github.com/mozilla/geckodriver/releases/download/v0.35.0/geckodriver-v0.35.0-linux64.tar.gz \
    && tar -xzf geckodriver-v0.35.0-linux64.tar.gz \
    && mv geckodriver /usr/local/bin/ \
    && chmod +x /usr/local/bin/geckodriver \
    && rm geckodriver-v0.35.0-linux64.tar.gz

# Add user appuser with UID and GID 1001
RUN addgroup -g 1001 appuser && \
    adduser -D -u 1001 -G appuser appuser && \
    # Create necessary directories
    mkdir -p /home/appuser/.mozilla && \
    mkdir -p /home/appuser/.cache/mozilla && \
    mkdir -p /app/screenshots && \
    # Set permissions
    chown -R appuser:appuser /home/appuser && \
    chown -R appuser:appuser /usr/bin/firefox* && \
    chown appuser:appuser /usr/local/bin/geckodriver && \
    chown -R appuser:appuser /app && \
    chmod 755 /usr/bin/firefox* && \
    chmod 755 /usr/local/bin/geckodriver

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY screenshot.py .
COPY server.py .

ENV MOZ_HEADLESS=1
ENV FIREFOX_BIN=/usr/bin/firefox-esr
ENV MOZ_DISABLE_CONTENT_SANDBOX=1
ENV GECKO_DRIVER_PORT=4444
ENV GECKO_DRIVER_HOST=127.0.0.1
# Add these Firefox-specific settings
ENV MOZ_FORCE_DISABLE_E10S=1
ENV HOME=/home/appuser

USER appuser

CMD ["uvicorn", "server:app", "--host", "0.0.0.0", "--port", "8080"]

EXPOSE 8080