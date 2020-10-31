FROM linuxserver/ffmpeg

EXPOSE 1080

HEALTHCHECK --retries=10 CMD [ "/vith", "-url", "http://localhost:1080/health" ]
ENTRYPOINT [ "/vith" ]

ARG APP_VERSION
ENV VERSION=${APP_VERSION}

VOLUME /tmp

ARG TARGETOS
ARG TARGETARCH

COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY release/vith_${TARGETOS}_${TARGETARCH} /vith
