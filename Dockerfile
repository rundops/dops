# Multi-stage build for the dops MCP server
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /dops .

FROM alpine:3.20

RUN apk add --no-cache git bash

COPY --from=builder /dops /opt/dops/bin/dops
ENV PATH="/opt/dops/bin:${PATH}"

# DOPS_HOME is where dops looks for config.json, catalogs/, and themes/.
# Mount your local .dops directory here:
#   docker run -i -v ~/.dops:/data/dops ghcr.io/<owner>/dops-cli
ENV DOPS_HOME=/data/dops

RUN mkdir -p /data/dops

ENTRYPOINT ["dops", "mcp", "serve"]
CMD ["--transport", "stdio"]
