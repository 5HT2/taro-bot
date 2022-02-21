FROM golang:alpine

RUN mkdir /taro-bot \
 && mkdir /taro-files
ADD . /taro-bot
WORKDIR /taro-bot

RUN apk add --no-cache bash
RUN go build -o taro .

ENV DEBUG "false"
WORKDIR /taro-files
CMD DEBUG="$DEBUG" /taro-bot/scripts/run.sh
