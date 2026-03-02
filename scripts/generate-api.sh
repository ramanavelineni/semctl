#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PKG_DIR="$PROJECT_ROOT/pkg/semapi"
SPEC_FILE="$PROJECT_ROOT/api/api-docs.yml"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.generate.yaml"
SEM_PORT=3000

# Step 1: Start Semaphore via docker compose
echo "Starting Semaphore..."
docker compose -f "$COMPOSE_FILE" up -d

# Step 2: Wait for Semaphore to be ready
echo "Waiting for Semaphore to become healthy..."
for i in $(seq 1 60); do
  if curl -sf "http://localhost:$SEM_PORT/api/ping" > /dev/null 2>&1; then
    echo "Semaphore is ready."
    break
  fi
  if [ "$i" -eq 60 ]; then
    echo "ERROR: Semaphore did not become ready in time."
    docker compose -f "$COMPOSE_FILE" down -v
    exit 1
  fi
  sleep 2
done

# Step 3: Fetch OpenAPI spec from the running instance
echo "Fetching API spec..."
mkdir -p "$(dirname "$SPEC_FILE")"
curl -sf "http://localhost:$SEM_PORT/swagger/api-docs.yml" -o "$SPEC_FILE"
echo "Spec saved to $SPEC_FILE"

# Step 4: Stop Semaphore (no longer needed)
echo "Stopping Semaphore..."
docker compose -f "$COMPOSE_FILE" down -v

# Step 5: Generate Go client using go-swagger
echo "Generating Go client..."
rm -rf "$PKG_DIR"
mkdir -p "$PKG_DIR"

# Requires: go install github.com/go-swagger/go-swagger/cmd/swagger@latest
swagger generate client \
  -f "$SPEC_FILE" \
  -t "$PKG_DIR" \
  --default-scheme http \
  -A semapi

echo "Done! Generated client in pkg/semapi/"
