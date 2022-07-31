# plugins

This README describes how to use plugins. See the [README](../README.md) in the root of the repo for a description of what plugins can do.

1. [Default plugins](#default-plugins)
2. [Compiling a plugin](#compiling-a-plugin)
3. [Hot-reloading plugins](#hot-reloading-plugins)
4. [Creating a plugin](#creating-a-plugin)
5. [Docker](#docker)

## Default plugins

Plugins that are loaded by default when running the bot are registered with the `-plugins` flag, and the `plugin-name.go` file maintains a list of plugins which are included by "default" with the bot.

Using this flag you are able to:
- Disable all plugins (use `-plugins=""`)
- Selectively load plugins on bot startup (use `-plugins="base my-plugin"`, and so on)

You are also able to change the directory that the bot looks for plugins in, which is `-pluginDir="bin"` by default. Changing this flag will not compile your plugins for you, the Makefile will check `plugins/` for compiling, and output them to `bin/`.

## Compiling a plugin

Running `make` on its own will compile the bot along with the plugins, by default. This is the code that `make` calls for compiling all plugins:

```bash
for d in ./plugins/*/; do
  echo "building $$d"
  go build -o "bin/" -buildmode=plugin "$$d"
done
```

You can compile a single plugin on your own using
```bash
go build -o "bin/" -buildmode=plugin "plugins/my-plugin/"
```

If you want your plugin to be loaded, you must add it to the `DefaultPlugins` list in `bot/config.go`, or the `config/plugins.json` file (described in the main README).

## Hot-reloading plugins

Currently, hot-reloading is technically possible but there are no commands to do so from the user-end. This README will be updated as issue [#8](https://github.com/5HT2/taro-bot/issues/8) is updated.

## Creating a plugin

All a plugin has to do is
- Have a `plugin-name.go` with a `package main` which declares a `func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin`.
- Be inside the `plugins/` (or other) directory in a directory under its own name, for example, `plugins/base/base.go` or `plugins/base-extra/base-extra.go`

The actual [`plugins.go`](https://github.com/5HT2/taro-bot/blob/master/plugins/plugins.go) code is heavily documented and explains the technical process of how plugins are loaded and work.

An example plugin's `example.go` can be found [in the `plugins` folder](https://github.com/5HT2/taro-bot/blob/master/plugins/example/example.go).

## Docker

You can modify the plugins to be loaded via Docker with the `config/plugins.json` file, as described in the main README.
