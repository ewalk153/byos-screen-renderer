# Stage 1: build the Go app
FROM golang:1.24 AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o app

# Stage 2: runtime container
FROM debian:bookworm-slim

# Install dependencies: Chromium + ImageMagick
RUN apt-get update && apt-get install -y \
  chromium chromium-driver \
  imagemagick \
  ca-certificates \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

# Set up user
RUN useradd -m appuser
USER appuser
WORKDIR /home/appuser

# Copy binary and template
COPY --from=builder /app/app .
COPY --from=builder /app/template.liquid .

# Expose server port
EXPOSE 8080

# Specify chromium path
ENV CHROMIUM_PATH=/usr/bin/chromium
ENV OUTPUT_PATH=/output/

# Launch server
CMD ["./app"]
