# Multi-stage build for the dops MCP server

# Stage 1: Build web UI
FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/internal/web/dist/ ./internal/web/dist/
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /dops .

FROM alpine:3.20

RUN apk add --no-cache git bash

COPY --from=builder /dops /opt/dops/bin/dops
ENV PATH="/opt/dops/bin:${PATH}"

# DOPS_HOME is where dops looks for config.json, catalogs/, and themes/.
# Mount your local .dops directory here:
#   docker run -i -v ~/.dops:/data/dops ghcr.io/rundops/dops
ENV DOPS_HOME=/data/dops

RUN mkdir -p /data/dops

ENTRYPOINT ["dops", "mcp", "serve"]
CMD ["--transport", "stdio"]
