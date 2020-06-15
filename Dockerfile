# Go builder container
FROM golang:alpine as builder
# ENV GOLANG_VERSION 1.10.6

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /go/src/github.com/scdoproject/go-scdo

WORKDIR /go/src/github.com/scdoproject/go-scdo

RUN make all

# Alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /go/src/github.com/scdoproject/go-scdo/build /slc

ENV PATH /slc:$PATH

RUN chmod +x /slc/node

EXPOSE 8027 8037 8057

# start a node with your 'config.json' file, this file must be external from a volume
# For example:
#   docker run -v <your config path>:/slc/config:ro -it slc node start -c /slc/config/configfile
