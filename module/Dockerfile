# --- Build stage --------------------------------------------------------------
FROM golang:1.25-alpine AS build

# Install build tools (git for go-get, ca-certificates for TLS during build)
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Enable Go modules and better caching: copy mod files first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build a static binary (no CGO) for Linux; inject version if you want
#ARG TARGETOS=linux
#ARG TARGETARCH=amd64
#ARG APP_NAME=app
#ARG VERSION=dev
#ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

# -s -w strips symbols to reduce size
#RUN go build .

# If your service listens on a port, expose it (optional)
#EXPOSE 6379

# Run as non-root (distroless uses nonroot user)
#USER nonroot:nonroot

CMD ["go", "run", "."]

