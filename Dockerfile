FROM linuxserver/ffmpeg

ENV VITH_READ_TIMEOUT 1m
ENV VITH_WRITE_TIMEOUT 1m

EXPOSE 1080

HEALTHCHECK --retries=10 CMD [ "/vith", "-url", "http://localhost:1080/health" ]
ENTRYPOINT [ "/vith" ]

ARG APP_VERSION
ENV VERSION=${APP_VERSION}

VOLUME /tmp

ARG TARGETOS
ARG TARGETARCH

COPY cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY release/vith_${TARGETOS}_${TARGETARCH} /vith
