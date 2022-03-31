FROM golang:1.18.0-alpine3.15

RUN mkdir /taro-bot \
 && mkdir /taro-files
ADD . /taro-bot
WORKDIR /taro-bot

RUN apk add --no-cache bash git \
 && go build -o taro .

ENV TZ "Local"
ENV DEBUG "false"
WORKDIR /taro-files
CMD DEBUG="$DEBUG" TZ="$TZ" /taro-bot/scripts/run.sh
