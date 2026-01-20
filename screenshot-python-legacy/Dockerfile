FROM python:3.11-alpine

RUN apk add --no-cache \
    firefox-esr \
    dbus \
    ttf-liberation \
    mesa-gl \
    pciutils-libs \
    bash \
    wget \
    && ARCH=$(apk --print-arch) \
    && case "$ARCH" in \
        x86_64) GECKO_ARCH=linux64 ;; \
        aarch64) GECKO_ARCH=linux-aarch64 ;; \
        armv7) GECKO_ARCH=linux-arm7 ;; \
        *) echo "Unsupported architecture: $ARCH" && exit 1 ;; \
    esac \
    && GECKO_VERSION=v0.36.0 \
    && GECKO_FILE="geckodriver-${GECKO_VERSION}-${GECKO_ARCH}.tar.gz" \
    && wget "https://github.com/mozilla/geckodriver/releases/download/${GECKO_VERSION}/${GECKO_FILE}" \
    && tar -xzf "${GECKO_FILE}" \
    && mv geckodriver /usr/local/bin/ \
    && chmod +x /usr/local/bin/geckodriver \
    && rm "${GECKO_FILE}"

RUN addgroup -g 1001 appuser && \
    adduser -D -u 1001 -G appuser appuser && \
    mkdir -p /home/appuser/.mozilla && \
    mkdir -p /home/appuser/.cache/mozilla && \
    chown -R appuser:appuser /home/appuser && \
    chown -R appuser:appuser /usr/bin/firefox* && \
    chown appuser:appuser /usr/local/bin/geckodriver && \
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
ENV MOZ_FORCE_DISABLE_E10S=1
ENV HOME=/home/appuser

USER appuser

ENV PORT=11235

EXPOSE 11235

CMD ["python", "server.py"]
