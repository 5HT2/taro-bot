# taro-bot

A Discord bot built in Go and Arikawa.

Notable features:
- Automatic command registration, with info support such as aliases and descriptions
- Strict arg parsing and selection
- Flexible auto-response and auto-scheduling system that can call Go code
- Automatic error handling via friendly messages given to the user :)
- Asynchronous event handling with concurrency-safe configs
- Fully-fledged plugin support

## Usage

```
git clone git@github.com:5HT2/taro-bot.git && cd taro-bot
make
./taro
```

You can also do `./update.sh` to run or update the Docker image, provided you have Docker installed.

#### Config

This is the simplest example of the `config/config.json` file, you only need `bot_token` to be set.

```json
{
    "bot_token": "bot token goes here"
}
```
