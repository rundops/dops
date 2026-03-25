#!/bin/sh
echo "==> Building Docker image: $REPO:$TAG"
echo ""
echo "  Step 1/6: FROM golang:1.26-alpine"
sleep 0.3
echo "  Step 2/6: COPY go.mod go.sum ./"
sleep 0.3
echo "  Step 3/6: RUN go mod download"
sleep 1
echo "  Step 4/6: COPY . ."
sleep 0.3
echo "  Step 5/6: RUN go build -o /app"
sleep 1.5
echo "  Step 6/6: ENTRYPOINT [\"/app\"]"
sleep 0.2
echo ""
echo "  Image built: $REPO:$TAG (142MB)"
if [ "$PUSH" = "true" ]; then
  echo ""
  echo "  Pushing to registry..."
  sleep 1.5
  echo "  ✓ Pushed $REPO:$TAG"
  echo "  Digest: sha256:$(LC_ALL=C tr -dc 'a-f0-9' </dev/urandom | head -c 12)"
fi
