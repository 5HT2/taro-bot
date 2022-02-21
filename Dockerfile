FROM golang:alpine

RUN mkdir /taro-bot \
 && mkdir /taro-files
ADD . /taro-bot
WORKDIR /taro-bot

RUN apk add --no-cache bash
RUN go build -o taro .

ENV DEBUG "false"
WORKDIR /taro-files
CMD /bin/bash -c "{ ( { /taro-bot/taro -debug \"$DEBUG\"; } 2>&1 1>&3 3>&- ) > /tmp/taro-bot.log; } || { /taro-bot/taro -exited $?; }"
