#!/bin/bash
# ./deploy.sh sandbox|prod

set -e

deployEnv=$1

# If variable is not set, exit
if [ -z "$deployEnv" ]; then
    echo "No env set. Call with sandbox or prod"
    exit 1
fi

functionName=hodl-fiat-$deployEnv

function sandboxenv() {
    echo -e "Variables={$(cat sandbox.env | paste -sd "," -)}"
}

function prodenv() {
    echo -e "Variables={$(cat prod.env | paste -sd "," -)}"
}

function build() {
    GOOS=linux go build -ldflags="-d -s -w" -o main main.go && chmod +x main
}

function pack() {
    zip -j $functionName.zip main
}

build
if [ $? -eq 1 ]; then
    exit 1
fi

pack
if [ $? -eq 1 ]; then
    exit 1
fi

if aws lambda get-function --function-name $functionName > /dev/null; then
    echo "Function exists.. Updating..."
    # Update function code
    aws lambda update-function-code --function-name $functionName \
        --zip-file fileb://$functionName.zip

    # Update function config
    aws lambda update-function-configuration --function-name $functionName \
        --description "Buys BTC if there's enough available cash in the account." \
        --environment file://$deployEnv.env.json
else
    echo "Creating function..."
    aws lambda create-function --function-name $functionName --runtime go1.x \
        --description "Buys BTC if there's enough available cash in the account." \
        --zip-file fileb://$functionName.zip --handler main \
        --environment file://$deployEnv.env.json \
        --role arn:aws:iam::414242556682:role/lambda-role
fi

rm $functionName.zip main
