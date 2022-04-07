#!/bin/bash

/taro-bot/taro -debug "$DEBUG" -plugindir "$PLUGIN_DIR" || {
    /taro-bot/taro -exited $?
}
