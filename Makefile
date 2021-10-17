taro-bot: clean
	go build -o taro .

deps:
	go get -u github.com/diamondburned/arikawa/v3

clean:
	rm -f discord-pretty-audit

run: taro-bot
	./taro

docker-build:
	@docker build -t ${IMG} .
	@docker tag ${IMG} ${LATEST}

docker-push:
	@docker push ${NAME}
