#!/bin/sh

PLUGINS_FILE="config/plugins.json"

build_all() {
  for d in ./plugins/*/; do
    echo "building $d"
    go build -o "bin/" -buildmode=plugin "$d"
  done
}

plugin_loaded() {
  jq "select(.loaded_plugins != []).loaded_plugins | index(\"$(basename "$1")\")" "$PLUGINS_FILE"
}

DEFAULT_LOADED="$(plugin_loaded "default")"

if [ -z "$(which jq)" ]; then
  echo "jq is not installed, doing unoptimized build..."
  build_all
elif [ ! -f "$PLUGINS_FILE" ]; then
  echo "$PLUGINS_FILE is missing, doing unoptimized build..."
  build_all
elif [ "$(jq ".loaded_plugins == []" "$PLUGINS_FILE")" = "true" ]; then
  echo "loaded_plugins is not set, building all plugins..."
  build_all
elif [ -n "$DEFAULT_LOADED" ] && [ "$DEFAULT_LOADED" != "null" ]; then
  echo "loaded_plugins contains \"default\", building all plugins..."
  build_all
else
  echo "building selected plugins..."
  for d in ./plugins/*/; do
    LOADED="$(plugin_loaded "$d")"

    if [ -n "$LOADED" ] && [ "$LOADED" != "null" ]; then
      echo "building $d"
      go build -o "bin/" -buildmode=plugin "$d"
    fi
  done
fi
