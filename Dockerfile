FROM --platform=$BUILDPLATFORM alpine
RUN apk add ffmpeg
COPY gotosocial-thumbnailer /usr/bin/gotosocial-thumbnailer
ENTRYPOINT ["/usr/bin/gotosocial-thumbnailer"]
