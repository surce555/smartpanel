# ==========================================
# Stage 1: Frontend Build
# ==========================================
FROM node:20-alpine AS frontend-builder
WORKDIR /build/frontend

# Pin pnpm 9: pnpm 10+ aborts the install with ERR_PNPM_IGNORED_BUILDS unless
# dependency build scripts are explicitly approved.
RUN npm install -g pnpm@9

# Copy lockfile + workspace config first for better layer caching. The
# pnpm-workspace.yaml carries the build-script allow-list (esbuild, vue-demi)
# for pnpm 10+; without it the install step fails.
COPY frontend/package.json frontend/pnpm-lock.yaml* frontend/.npmrc* frontend/pnpm-workspace.yaml* ./
RUN pnpm install --prefer-offline

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
