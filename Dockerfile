FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o webinar-app .

FROM alpine:3.21
RUN adduser -D -u 1000 appuser
COPY --from=builder /app/webinar-app /usr/local/bin/webinar-app
USER 1000
EXPOSE 8080
ENTRYPOINT ["webinar-app"]
