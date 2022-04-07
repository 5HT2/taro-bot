FROM golang:1.18.0

RUN mkdir /taro-bot \
 && mkdir /taro-files
ADD . /taro-bot
WORKDIR /taro-bot

RUN for d in ./plugins/*/; do echo "building $d"; go build -o "bin/" -buildmode=plugin "$d"; done \
 && go build -o taro .

ENV TZ "Local"
ENV DEBUG "false"
WORKDIR /taro-files
CMD DEBUG="$DEBUG" TZ="$TZ" PLUGIN_DIR="/taro-bot/bin" /taro-bot/scripts/run.sh
