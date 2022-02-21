FROM golang:1.17.1

RUN mkdir /taro-bot \
 && mkdir /taro-files
ADD . /taro-bot
WORKDIR /taro-bot

RUN go build -o taro .

ENV DEBUG "false"
WORKDIR /taro-files
CMD /taro-bot/taro -debug $DEBUG 2> /tmp/taro-bot.log || /taro-bot/taro -exited $?
