# Build stage
FROM golang:1.25-alpine AS builder

RUN apk --no-cache add git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -buildvcs=false -ldflags '-extldflags "-static" -s -w' -o github-forgejo-mirror .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/github-forgejo-mirror .

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

ENTRYPOINT ["./github-forgejo-mirror"]
