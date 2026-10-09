# ==========================================
# Stage 1: Frontend Build
# ==========================================
FROM node:20-alpine AS frontend-builder
WORKDIR /build/frontend

# Install pnpm directly
RUN npm install -g pnpm@9

COPY frontend/package.json frontend/pnpm-lock.yaml* ./
RUN pnpm install

COPY frontend/ ./
RUN pnpm run build

# ==========================================
# Stage 2: Backend Build (CGO-free Pure Go)
# ==========================================
FROM golang:alpine AS backend-builder
WORKDIR /build/backend

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

ENV GOTOOLCHAIN=auto

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/smartpanel .

# ==========================================
# Stage 3: Minimal Alpine Runtime
# ==========================================
FROM alpine:3.20
LABEL maintainer="surce555"
LABEL org.opencontainers.image.source="https://github.com/surce555/smartpanel"
LABEL org.opencontainers.image.description="SmartPanel - Intelligent Dual-Stack NAS Navigation Dashboard"

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary and frontend assets
COPY --from=backend-builder /build/smartpanel /app/smartpanel
COPY --from=frontend-builder /build/frontend/dist /app/dist

# Default directories
RUN mkdir -p /app/data /app/uploads /app/data/logs

# Environment variables
ENV PANEL_PORT=5666 \
    DATA_DIR=/app/data \
    UPLOADS_DIR=/app/uploads \
    TZ=Asia/Shanghai

EXPOSE 5666

VOLUME ["/app/data", "/app/uploads"]

ENTRYPOINT ["/app/smartpanel"]
