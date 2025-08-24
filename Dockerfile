FROM alpine

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

RUN addgroup -g 1000 proxy && \
    adduser -u 1000 -G proxy -s /bin/sh -D proxy

RUN mkdir -p /app /config && \
    chown -R proxy:proxy /app /config

WORKDIR /app

COPY k8-proxy .
RUN chmod +x k8-proxy

USER proxy
ENTRYPOINT ["./k8-proxy"]