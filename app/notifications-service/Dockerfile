FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -o notifications-service .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /src/notifications-service /app/notifications-service
EXPOSE 8083
ENTRYPOINT ["/app/notifications-service"]
