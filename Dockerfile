FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /whiteboard ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /whiteboard /whiteboard
ENV PORT=8080 DB_PATH=/data/whiteboard.db
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/whiteboard"]
