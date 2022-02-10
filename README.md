# taro-bot

A Discord bot built in Go and Arikawa.

Notable features:
- Auto-response system that can call Go code from JSON
- Automatic command registration, with info support such as aliases and descriptions
- Strict arg parsing and selection
- Automatic error handling via friendly messages given to the user :)
- Asynchronous event handling with concurrency-safe configs

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
        },
        {
            "embed": false,
            "title": "",
            "description": "%s",
            "reflect_func": "SpotifyToYoutubeResponse",
            "regexes": [
                "https?:\\/\\/open\\.spotify\\.com\\/track\\/[a-zA-Z0-9][^\\s]{2,}"
            ],
            "match_min": 1
        }
    ]
}
```

## TODO

- [ ] Allow `"stuff here"` in order to create single args that contain spaces
  - [x] Allow selecting a range of args to return as one
- [ ] Each command should be able to register its own config settings
- [x] Asynchronous config states. Right now you need to store it in a variable.
- [x] Better scalability.
