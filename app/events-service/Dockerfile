FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -o events-service .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /src/events-service /app/events-service
EXPOSE 8082
ENTRYPOINT ["/app/events-service"]
