#!/bin/bash

# shellcheck disable=SC1091
source "$HOME/.env"
if [[ -z "$TARO_PATH" ]]; then
  echo "TARO_PATH not set!"
  exit 1
fi

docker pull l1ving/taro-bot:latest

docker stop taro || echo "Could not stop missing container taro"
docker rm taro || echo "Could not remove missing container taro"

docker run --name taro \
  --mount type=bind,source="$TARO_PATH",target=/taro-files \
  --network host -d \
  l1ving/taro-bot
