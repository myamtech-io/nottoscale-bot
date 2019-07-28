ARG SIMC_VERSION="820-01"

FROM golang:1.12

ARG protocVersion="3.9.0"

RUN pwd

WORKDIR /tmp

RUN set -ex && \
  apt-get update && \
  apt-get install -y wget unzip && \
  wget --quiet "https://github.com/protocolbuffers/protobuf/releases/download/v$protocVersion/protoc-$protocVersion-linux-x86_64.zip" && \
  rm -rf /var/lib/apt/lists/* && \
  unzip "protoc-$protocVersion-linux-x86_64.zip" && \
  mv include/* /usr/include && \
  mv bin/* /usr/bin && \
  go get -u github.com/golang/protobuf/protoc-gen-go

COPY build-proto.sh .
COPY proto proto

RUN ./build-proto.sh

FROM golang:1.12-alpine

ARG SIMC_VERSION

RUN set -ex && \
    apk add --no-cache bash git openssh curl-dev alpine-sdk

RUN set -ex && \
    curl --silent -LO https://github.com/simulationcraft/simc/archive/release-$SIMC_VERSION.tar.gz && \
    tar -xf release-$SIMC_VERSION.tar.gz

RUN set -ex && \
    cd simc-release-$SIMC_VERSION/engine && \
    make optimized

RUN mv simc-release-$SIMC_VERSION/engine/simc /bin/simc

RUN set -ex && \
    go get github.com/bwmarrin/discordgo && \
    go get github.com/sirupsen/logrus && \
    go get -u github.com/aws/aws-sdk-go/... && \
    go get github.com/golang/protobuf/proto && \
    go get github.com/google/uuid

RUN mkdir -p src/github.com/webmakersteve/myamtech-bot

WORKDIR /go/src/github.com/webmakersteve/myamtech-bot

COPY . .
COPY --from=0 /go/src/github.com/webmakersteve/myamtech-bot/proto proto

RUN go build -o /tmp/myamtech-bot

FROM golang:1.12-alpine

RUN set -ex && \
    apk add --no-cache curl-dev alpine-sdk

COPY --from=1 /bin/simc /usr/bin/simc
COPY --from=1 /tmp/myamtech-bot bin/myamtech-bot

CMD [ "./bin/myamtech-bot" ]
