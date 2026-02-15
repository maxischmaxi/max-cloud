# Build stage - must build from monorepo root for replace directives
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files from root and shared package first
COPY go.mod go.sum ./
COPY packages/shared/go.mod packages/shared/go.sum ./packages/shared/
COPY apps/cli/go.mod apps/cli/go.sum ./apps/cli/

# Download dependencies
WORKDIR /app/apps/cli
RUN go mod download

# Copy all source code
COPY packages/shared ./packages/shared
COPY apps/cli ./apps/cli

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /maxcloud .

# Runtime stage
FROM gcr.io/distroless/static-debian12

COPY --from=builder /maxcloud /maxcloud

ENTRYPOINT ["/maxcloud"]
