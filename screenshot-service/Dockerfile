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

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY screenshot.py .
COPY server.py .

ENV DISPLAY=:99
ENV MOZ_HEADLESS=1
ENV FIREFOX_BIN=/usr/bin/firefox
ENV PORT=8080

# Create screenshots directory
RUN mkdir -p screenshots && chmod 777 screenshots

# Start Xvfb and the FastAPI server
CMD ["sh", "-c", "Xvfb :99 -screen 0 1920x1080x24 -ac +extension GLX +render & uvicorn server:app --host 0.0.0.0 --port 8080"]

EXPOSE 8080
