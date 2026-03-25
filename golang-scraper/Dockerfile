FROM golang:1.22-alpine AS builder
WORKDIR /workspace
COPY go.mod ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -a -ldflags="-s -w" -o scraper .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /workspace/scraper /scraper
USER 65532:65532
ENTRYPOINT ["/scraper"]