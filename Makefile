NAME   := l1ving/taro-bot
TAG    := $(shell git log -1 --pretty=%h)
IMG    := ${NAME}:${TAG}
LATEST := ${NAME}:latest

taro-bot: clean build build-plugins

run: taro-bot
	./taro

build:
	go build -o taro .

deps:
	go get -u github.com/diamondburned/arikawa/v3
	go get -u github.com/5HT2C/http-bash-requests
	go get -u github.com/go-co-op/gocron
	go get -u github.com/mackerelio/go-osstat
	go get -u github.com/forPelevin/gomoji
	go get -u golang.org/x/net
	go get -u golang.org/x/text/language
	go get -u golang.org/x/text/message
	go mod tidy

clean:
	rm -f taro

build-plugins:
	./scripts/build-plugins.sh


docker-build:
	@docker build -t ${IMG} .
	@docker tag ${IMG} ${LATEST}

docker-push:
	@docker push ${NAME}
