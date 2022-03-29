# syntax=docker/dockerfile:experimental

FROM golang:1.18-alpine3.15 as build

RUN apk add --no-cache --update \
  zeromq-dev \
   git \
  make

WORKDIR tm-events

COPY . . 
RUN make build

FROM alpine:3.15

COPY --from=build /go/tm-events/bin/* /usr/local/bin/

