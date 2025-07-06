# Stage 1: Build the SvelteKit frontend with Bun
FROM node AS frontend-builder
WORKDIR /app/frontend

# Copy package.json
COPY web/package*.json ./

# Install dependencies with clean npm cache and production only
RUN npm install --include=dev

# Copy the rest of the frontend code
COPY web .

# Build the SvelteKit app
RUN NODE_ENV=production npm run build

# Stage 2: Build the Go backend
FROM golang:1.24-alpine AS backend-builder
RUN apk add --update gcc musl-dev --no-cache
WORKDIR /app/backend

# Copy go mod and sum files
COPY api/go.mod api/go.sum ./

# Download dependencies with verify
RUN go mod download && go mod verify

# Copy the source from the current directory to the working Directory inside the container
COPY api .

# Build with security flags and optimizations
RUN CGO_ENABLED=1 GOOS=linux go build -a \
    -ldflags='-w -s -linkmode external -extldflags "-static"' \
    -o chase .

# Stage 3: Final stage
FROM scratch

ENV GIN_MODE=release

# Copy SSL certificates for HTTPS support
COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary and web files as user 101
COPY --from=backend-builder --chown=101:101 /app/backend/chase /chase
COPY --from=frontend-builder --chown=101:101 /app/frontend/build /web

# Define any necessary volumes
VOLUME ["/data"]

# Set user 101
USER 101

# Expose port 8080
EXPOSE 8080

# Run with explicit entrypoint and cmd
ENTRYPOINT ["/chase"]
CMD []
