FROM golang:1.24.5 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o configmap-watcher

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/configmap-watcher /

USER nonroot:nonroot
CMD ["/configmap-watcher"]
