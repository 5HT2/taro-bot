# taro-bot

[![](https://img.shields.io/badge/discord%20bot-invite!-5865F2?logo=discord&logoColor=white)](https://discord.com/oauth2/authorize?client_id=893216230410952785&permissions=278404582464&scope=bot)<br>
[![CodeQL](https://github.com/5HT2/taro-bot/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/5HT2/taro-bot/actions/workflows/codeql-analysis.yml)
[![docker-build](https://github.com/5HT2/taro-bot/actions/workflows/docker-build.yml/badge.svg)](https://github.com/5HT2/taro-bot/actions/workflows/docker-build.yml)

A Discord bot built in Go and Arikawa.

**Notable features:**
- Strict arg parsing and selection
- Automatic error handling via friendly messages given to the user :)
- Asynchronous event handling with concurrency-safe configs
- Fully-fledged plugin support

**A feature (plugin) is able to:**
- Return comprehensive commands, with [info support such as aliases and descriptions](https://github.com/5HT2/taro-bot/blob/99b929ac18d583a38a332405b45dd53d57143b17/plugins/base/base.go#L19).
- Return "auto responses", with [flexible message matching to call Go code](https://github.com/5HT2/taro-bot/blob/99b929ac18d583a38a332405b45dd53d57143b17/plugins/tenor-delete/tenor-delete.go#L28).
- Return scheduled jobs, to be [called at an interval or based on conditions](https://github.com/5HT2/taro-bot/blob/99b929ac18d583a38a332405b45dd53d57143b17/plugins/vintagestory/vintagestory.go#L30).
- Register event handlers to Discord's gateway, such as [when a reaction is added to a message](https://github.com/5HT2/taro-bot/blob/99b929ac18d583a38a332405b45dd53d57143b17/plugins/starboard/starboard.go#L123).

**All bot features are plugins**, and can be enabled or disabled on demand, with hot-reloading being added soon ([#8](https://github.com/5HT2/taro-bot/issues/8)).

More information about using and creating plugins is described in the [plugin documentation](https://github.com/5HT2/taro-bot/blob/master/plugins).

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
