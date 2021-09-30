taro-bot: clean
	go get -u github.com/diamondburned/arikawa/v3
	go build -o taro .

clean:
	rm -f discord-pretty-audit

run: taro-bot
	./taro
