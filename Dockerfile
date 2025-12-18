# --- Stage 1: Build ---
FROM golang:1.24.1-bookworm AS builder

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_TIME=unknown

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 makes it a statically linked binary (no libc dependency)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X 'main.Version=${VERSION}' \
              -X 'main.Commit=${COMMIT}' \
              -X 'main.BuildTime=${BUILD_TIME}'" \
    -o gitinfo main.go

# --- Stage 2: Final ---
FROM gcr.io/distroless/static-debian12:latest

WORKDIR /

COPY --from=builder /app/gitinfo /gitinfo
USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/gitinfo"]