FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/trustmesh-server ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
  && addgroup -S trustmesh \
  && adduser -S -G trustmesh trustmesh

WORKDIR /app

COPY --from=builder /out/trustmesh-server /usr/local/bin/trustmesh-server

USER trustmesh

EXPOSE 8080

ENTRYPOINT ["trustmesh-server"]
