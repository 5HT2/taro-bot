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

## TODO

- [ ] Allow `"stuff here"` in order to create single args that contain spaces
  - [ ] Allow selecting a range of args to return as one
