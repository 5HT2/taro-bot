FROM golang:1.17.1

RUN mkdir /taro-bot \
 && mkdir /taro-files
ADD . /taro-bot
WORKDIR /taro-bot

RUN go build -o taro .

ENV DEBUG "false"
WORKDIR /taro-files
CMD { /taro-bot/taro -debug "$DEBUG" > >(tee -a /tmp/taro-bot-stdout.log) 2> >(tee -a /tmp/taro-bot.log >&2); } || { /taro-bot/taro -exited $?; }
