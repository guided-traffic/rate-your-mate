# Build stage
FROM golang:1.25-alpine AS builder

# Build arguments for metadata
ARG BUILD_NUMBER
ARG GIT_COMMIT
ARG BUILD_TIME

# Set build arguments for cross-compilation
ARG TARGETOS
ARG TARGETARCH

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with cross-compilation support
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -a -installsuffix cgo \
    -ldflags="-w -s -X main.Version=${BUILD_NUMBER:-dev} -X main.GitCommit=${GIT_COMMIT:-unknown} -X main.BuildTime=${BUILD_TIME:-unknown}" \
    -o main .

# Final stage - using distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

# Build arguments for metadata
ARG BUILD_NUMBER
ARG GIT_COMMIT
ARG BUILD_TIME

# Add OCI labels for better metadata
LABEL org.opencontainers.image.title="LAN Party Manager Backend" \
      org.opencontainers.image.description="Backend for LAN Party Manager - Achievement voting system" \
      org.opencontainers.image.vendor="Guided Traffic" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.documentation="https://github.com/guided-traffic/lan-party-manager" \
      org.opencontainers.image.source="https://github.com/guided-traffic/lan-party-manager" \
      org.opencontainers.image.version="${BUILD_NUMBER:-dev}" \
      org.opencontainers.image.revision="${GIT_COMMIT:-unknown}" \
      org.opencontainers.image.created="${BUILD_TIME:-unknown}"

# distroless images run as non-root user 65532 (nonroot) by default
# distroless includes ca-certificates and tzdata

WORKDIR /app
COPY --from=builder /app/main .

USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/app/main"]
