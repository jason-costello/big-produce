FROM golang:1.17.0-alpine3.13 AS builder
RUN apk update && apk add ca-certificates

ADD ./ /appdir/
RUN cd /appdir && \
    go mod tidy && \
    go mod vendor && \
    GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -a -ldflags=-X=main.version=${VERSION} -tags netgo -ldflags="-w -s" -o app

## Build scratch container and only copy over binary and certs
FROM scratch
COPY --from=builder /appdir/app /usr/local/bin/app

USER 1001
EXPOSE :8088
ENTRYPOINT [ "app" ]
