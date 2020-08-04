FROM golang:1.14 as builder

WORKDIR /app
COPY . .

RUN make \
 && git diff -- *.go \
 && git diff --quiet -- *.go

ARG CODECOV_TOKEN
RUN curl -q -sSL --max-time 30 https://codecov.io/bash | bash

FROM alpine as fetcher

WORKDIR /app

RUN apk --update add curl \
 && curl -q -sSL --max-time 30 -o /app/cacert.pem https://curl.haxx.se/ca/cacert.pem

FROM jrottenberg/ffmpeg:4.3-scratch

EXPOSE 1080

HEALTHCHECK --retries=10 CMD [ "/vith", "-url", "http://localhost:1080/health" ]
ENTRYPOINT [ "/vith" ]

ARG APP_VERSION
ENV VERSION=${APP_VERSION}

VOLUME /tmp

COPY --from=fetcher /app/cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/bin/vith /vith
