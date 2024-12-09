# syntax=docker/dockerfile:experimental

FROM golang:1.23.4-alpine3.21 as build

RUN apk add --no-cache --update \
  zeromq-dev \
   git \
  make

WORKDIR tm-events

COPY . . 
RUN make build

FROM alpine:3.21

COPY --from=build /go/tm-events/bin/* /usr/local/bin/

