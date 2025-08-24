FROM python:3.11-slim

WORKDIR /usr/src/app

# System deps for building/grpcio-tools (no heavy toolchains needed)
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash curl ca-certificates \
  && rm -rf /var/lib/apt/lists/*

# Copy deps early for better caching
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy source
COPY . .

# Generate Python code from .proto at build-time (optional; you can also check gen files into repo)
RUN python -m grpc_tools.protoc \
    -I. \
    --python_out=. \
    proto/telemetry.proto

# Default command (overridden by compose)
CMD ["python", "-m", "main"]

