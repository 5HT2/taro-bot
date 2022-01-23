NAME   := l1ving/taro-bot
TAG    := $(shell git log -1 --pretty=%h)
IMG    := ${NAME}:${TAG}
LATEST := ${NAME}:latest

taro-bot: clean
	go build -o taro .

deps:
	go get -u github.com/diamondburned/arikawa/v3
	go get -u golang.org/x/net

clean:
	rm -f taro

run: taro-bot
	./taro

docker-build:
	@docker build -t ${IMG} .
	@docker tag ${IMG} ${LATEST}

docker-push:
	@docker push ${NAME}
