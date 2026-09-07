FROM golang:alpine AS builder

WORKDIR /app

# Copy all source files along with vendor directory
COPY . .

# Build static binary using local vendor dependencies (no network required)
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-w -s" -o covered-call-tracker ./cmd/server

# Stage 2: Minimal runner image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/covered-call-tracker .
COPY --from=builder /app/templates ./templates

EXPOSE 8080

CMD ["./covered-call-tracker"]