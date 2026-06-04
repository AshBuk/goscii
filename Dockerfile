FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o goscii .

# Go stays in the final image: `go run` compiles player code at runtime.
FROM golang:1.26-alpine
RUN adduser -D -h /home/goscii goscii
ENV HOME=/home/goscii
# 256-color terminal so Lip Gloss renders the lavender palette.
ENV TERM=xterm-256color
COPY --from=builder /app/goscii /usr/local/bin/goscii
USER goscii
WORKDIR /home/goscii
ENTRYPOINT ["goscii"]
CMD ["start"]
