FROM golang:1.17.1-alpine AS build
WORKDIR /go/src/github.com/isucon/isucon11-final/webapp/go
COPY ./go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=~/go/pkg/mod --mount=type=cache,target=~/.cache/go-build CGO_ENABLED=0 go build -o /isucholar -ldflags "-s -w"

FROM ubuntu:20.04

RUN apt-get update && apt-get install -y wget zip

ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

COPY --from=build /isucholar /bin/
WORKDIR /webapp/go

ENTRYPOINT ["/bin/isucholar"]
