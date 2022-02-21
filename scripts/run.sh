#!/bin/bash

( { /taro-bot/taro -debug "$DEBUG"; } 2>&1 1>&3 3>&- ) > /tmp/taro-bot.log || {
    /taro-bot/taro -exited $?
}
