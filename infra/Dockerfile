FROM golang:1.15-alpine
LABEL maintainer="The Datafuselabs Authors"

RUN apk add git make

COPY ./bin/infra /bin/infra

ENTRYPOINT ["/bin/infra"]
