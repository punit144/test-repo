FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -o deployments-service .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /src/deployments-service /app/deployments-service
EXPOSE 8081
ENTRYPOINT ["/app/deployments-service"]
