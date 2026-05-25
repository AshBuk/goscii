FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o goscii .

FROM golang:1.26-alpine
RUN addgroup -S goscii && adduser -S -G goscii goscii
COPY --from=builder /app/goscii /usr/local/bin/goscii
USER goscii
ENTRYPOINT ["goscii"]
