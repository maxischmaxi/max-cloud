FROM golang:1.23-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o api .

FROM gcr.io/distroless/static-debian12

COPY --from=builder /build/api /api

EXPOSE 8080

ENTRYPOINT ["/api"]
