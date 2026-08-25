# syntax=docker/dockerfile:1

# Build the React assets separately so dependency installation remains cached
# when only Go code changes.
FROM node:22-bookworm AS frontend-builder
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Wails on Linux uses GTK and WebKitGTK through cgo.
FROM golang:1.25-bookworm AS builder
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libgtk-3-dev \
    libwebkit2gtk-4.0-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist
RUN CGO_ENABLED=1 go build -trimpath -tags desktop,production -o /out/cash ./cmd/cash

# The final image is intended to run the Linux desktop build through the
# host's X11 display. Application data lives in the mounted /data directory.
FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgtk-3-0 \
    libwebkit2gtk-4.0-37 \
    && rm -rf /var/lib/apt/lists/*
ENV XDG_CONFIG_HOME=/data \
    WEBKIT_DISABLE_DMABUF_RENDERER=1
WORKDIR /app
COPY --from=builder /out/cash /app/cash
VOLUME ["/data"]
ENTRYPOINT ["/app/cash"]

# A lightweight verification target for CI and local validation.
FROM builder AS test
RUN go test ./... && go vet ./...
