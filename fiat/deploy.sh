#!/bin/bash

set -e

function build() {
    GOOS=linux go build -ldflags="-d -s -w" -o main main.go && chmod +x main
}

function pack() {
    zip -j fiat.zip main
}

build
if [ $? -eq 1 ]; then
    exit 1
fi

pack
if [ $? -eq 1 ]; then
    exit 1
fi

# aws lambda create-function --function-name hodl-fiat --runtime go1.x \
#   --zip-file fileb://fiat.zip --handler main \
#   --role arn:aws:iam::414242556682:role/service-role/hodl-role-311r6kb5

aws lambda update-function-code --function-name hodl-fiat --zip-file fileb://fiat.zip

rm fiat.zip main