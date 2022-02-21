#!/bin/bash

/taro-bot/taro -debug "$DEBUG" || {
    /taro-bot/taro -exited $?
}
