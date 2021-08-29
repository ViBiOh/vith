FROM alpine

EXPOSE 1080
USER 405

HEALTHCHECK --retries=10 CMD [ "/vith", "-url", "http://localhost:1080/health" ]
ENTRYPOINT [ "/vith" ]

ARG APP_VERSION
ENV VERSION=${APP_VERSION}

VOLUME /tmp

ARG TARGETOS
ARG TARGETARCH

COPY ffmpeg/${TARGETOS}/${TARGETARCH}/ffmpeg /usr/bin/ffmpeg
COPY ffmpeg/${TARGETOS}/${TARGETARCH}/ffprobe /usr/bin/ffprobe

COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY release/vith_${TARGETOS}_${TARGETARCH} /vith
