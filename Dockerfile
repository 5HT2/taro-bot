FROM golang:1.17.1

RUN mkdir /taro-bot \
 && mkdir /taro-files
ADD . /taro-bot
WORKDIR /taro-bot

RUN go build -o taro .

WORKDIR /taro-files
CMD /taro-bot/taro
