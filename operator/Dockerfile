# Multi-stage build for the custom operator.
# Stage 1: build - compiles a static binary for linux/arm64 (Raspberry Pi)
# Stage 2: runtime - minimal distroless image, no shell, no package manager

# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -a -ldflags="-s -w" -o manager ./cmd/main/main.go

# ---- Runtime stage ----
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /workspace/manager .

USER 65532:65532

ENTRYPOINT ["/manager"]


