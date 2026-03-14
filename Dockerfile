FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /reel-life ./cmd/reel-life

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /reel-life /reel-life
EXPOSE 8080
ENTRYPOINT ["/reel-life"]
