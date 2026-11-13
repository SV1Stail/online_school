FROM golang:1.27.3-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bootstrap ./cmd/bootstrap

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app

COPY --from=builder /out/api /usr/local/bin/api
COPY --from=builder /out/bootstrap /usr/local/bin/bootstrap

USER app
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/api"]
