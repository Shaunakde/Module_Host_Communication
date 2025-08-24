#!/usr/bin/env bash
set -euo pipefail

# -------- CONFIG --------
GUI_DIR="./gui"          # Docker build context (has Dockerfile)
MODULE_DIR="./module"    # Go module (contains main)
IMAGE_NAME="gui-app"     # Docker image name
CONTAINER_NAME="gui-app-run"

# Optional: pass args to 'go run' and to the container CMD
GO_ARGS=${GO_ARGS:-}
APP_ARGS=${APP_ARGS:-}

# -------- CHECKS --------
[[ -d "$GUI_DIR" ]] || { echo "Missing $GUI_DIR"; exit 1; }
[[ -d "$MODULE_DIR" ]] || { echo "Missing $MODULE_DIR"; exit 1; }

# Detect OS to choose networking (host networking is Linux-only)
#OS="$(uname -s)"
#if [[ "$OS" == "Linux" ]]; then
#  DOCKER_NET=(--network host)
#else
  # On macOS/Windows, host networking isn't available.
  # If you need ports, export PORTS="-p 8000:8000 -p 5000:5000" before running.
  DOCKER_NET=()
#fi

# If user provides PORTS env (e.g., "-p 8000:8000"), include it
DOCKER_PORTS=()
if [[ -n "${PORTS:-}" ]]; then
  # shellcheck disable=SC2206
  DOCKER_PORTS=(${PORTS})
fi

# -------- CLEANUP HANDLER --------
cleanup() {
  echo ""
  echo "[CLEANUP] Stopping background processes…"
  # Kill Go program if still running
  if [[ -n "${GO_PID:-}" ]] && ps -p "$GO_PID" >/dev/null 2>&1; then
    kill "$GO_PID" 2>/dev/null || true
    wait "$GO_PID" 2>/dev/null || true
  fi
  # Stop running container (if still alive)
  if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    docker stop "$CONTAINER_NAME" >/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

# -------- BUILD DOCKER IMAGE --------
echo "[GUI] Building Docker image: ${IMAGE_NAME}"
docker build -t "$IMAGE_NAME" "$GUI_DIR"

# -------- RUN GO PROGRAM (background, non-TTY) --------
(
  echo "[Module] Starting Go app in ${MODULE_DIR} ..."
  cd "$MODULE_DIR"
  # Logs will stream to this terminal; process is non-interactive
  GO111MODULE=on go run . $GO_ARGS
) &
GO_PID=$!

# -------- RUN DOCKER APP (foreground, TTY) --------
echo "[GUI] Running Docker container (interactive)"
# If you need X11 GUI: add -e DISPLAY and -v /tmp/.X11-unix:/tmp/.X11-unix:ro here.
docker run --rm -it \
  --name "$CONTAINER_NAME" \
  -p 6379:6379 \
#  "${DOCKER_NET[@]}" \
#  "${DOCKER_PORTS[@]}" \
  "$IMAGE_NAME" $APP_ARGS

# When the container exits, the trap will clean up the Go process.

