FROM golang:1.25-alpine AS builder

ARG CLAWSYNAPSE_REPO=https://github.com/yuanjun5681/clawsynapse.git
ARG CLAWSYNAPSE_REF=main

RUN apk add --no-cache git ca-certificates

WORKDIR /src

RUN git clone --depth 1 --branch "${CLAWSYNAPSE_REF}" "${CLAWSYNAPSE_REPO}" .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/clawsynapsed ./cmd/clawsynapsed

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata su-exec \
  && addgroup -S clawsynapse \
  && adduser -S -G clawsynapse clawsynapse \
  && mkdir -p /var/lib/clawsynapse \
  && mkdir -p /var/lib/trustmesh-transfers \
  && chown -R clawsynapse:clawsynapse /var/lib/clawsynapse \
  && chown -R clawsynapse:clawsynapse /var/lib/trustmesh-transfers

WORKDIR /app

COPY --from=builder /out/clawsynapsed /usr/local/bin/clawsynapsed
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod 755 /usr/local/bin/docker-entrypoint.sh

VOLUME ["/var/lib/clawsynapse"]
VOLUME ["/var/lib/trustmesh-transfers"]

EXPOSE 18080

ENTRYPOINT ["docker-entrypoint.sh"]
