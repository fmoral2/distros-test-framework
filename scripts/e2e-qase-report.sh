#!/usr/bin/env bash


LOGS=$1
PS4='+(${LINENO}): '
set -e
trap 'echo "Error on line $LINENO: $BASH_COMMAND"' ERR



if [ -z "$LOGS" ]; then
    echo "Error: Please provide a log file path as argument"
    exit 1
fi


if [ ! -r "$LOGS" ]; then
    echo "Error: File $LOGS does not exist or is not readable"
    exit 1
fi

content=$(<"$LOGS")
echo "File contents: $content"
