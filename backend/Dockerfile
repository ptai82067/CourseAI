# Multi-stage Dockerfile for backend (Go + Fiber)
# Production-ready Dockerfile for Render deployment

# builder
FROM golang:1.21-alpine AS builder
WORKDIR /app

# install git and ca-certificates for private modules if needed
RUN apk add --no-cache git ca-certificates && update-ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags netgo -ldflags "-s -w" -o server ./cmd/server/main.go

# final image
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/server ./server

# Declare environment variables that will be injected at runtime
# These are required for production deployment on Render/cloud platforms
# Render will override these with actual values from Environment Variables
ENV DATABASE_URL=""
ENV ENV="production"
ENV PORT="8080"
ENV JWT_SECRET=""
ENV FRONTEND_URL=""

EXPOSE 8080
CMD ["./server"]
