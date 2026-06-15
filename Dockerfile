FROM golang:1.25-alpine

WORKDIR /app
RUN apk add --no-cache bash curl
COPY . .
RUN go mod download

ENV PORT=80 DB_PATH=/tmp/whiteboard.db GIN_MODE=debug
EXPOSE 80

CMD ["go", "run", "./cmd/server"]
