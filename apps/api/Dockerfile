# Build stage - must build from monorepo root for replace directives
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files from root and shared package first
COPY go.mod go.sum ./
COPY packages/shared/go.mod packages/shared/go.sum ./packages/shared/
COPY apps/api/go.mod apps/api/go.sum ./apps/api/

# Download dependencies
WORKDIR /app/apps/api
RUN go mod download

# Copy all source code
COPY packages/shared ./packages/shared
COPY apps/api ./apps/api

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api .

# Runtime stage
FROM gcr.io/distroless/static-debian12

COPY --from=builder /api /api

EXPOSE 8080

ENTRYPOINT ["/api"]
