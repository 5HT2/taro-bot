# taro-bot

A Discord bot built in Go and Arikawa.

Notable features:
- Auto-response system that can call Go code from JSON
- Automatic command registration, with info support such as aliases and descriptions
- Strict arg parsing and selection
- Automatic error handling via friendly messages given to the user :)

## Usage

```
git clone git@github.com:l1ving/taro-bot.git && cd taro-bot
make
./taro
```

You can also do `./update.sh` to run or update the Docker image, provided you have Docker installed.

#### Config

This is the simplest example of the `config/config.json` file, you only need `bot_token` as `global_responses` is completely optional.

Per-guild responses will be configurable with a command in the future, while `global_responses` will have to be changed by the bot owner in the config.

```json
{
    "bot_token": "bot token goes here",
    "global_responses": [
        {
            "title": "",
            "description": "The current prefix is `%s`",
            "reflect_func": "PrefixResponse",
            "regexes": [
                "<@!?your bot ID goes here>",
                "prefix"
            ],
            "match_min": 2
        }
    ]
}
```

## TODO

- [ ] Allow `"stuff here"` in order to create single args that contain spaces
  - [ ] Allow selecting a range of args to return as one
