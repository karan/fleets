# hodl

🚀To the moon 🚀

DCA on BTC and ETH using Coinbase Pro.

## How it works

Depending on the schedule you set, this bot will:

- Read config file
- Check if USDC account on cPro has enough balance to buy BTC and ETH
  - If not enough, transfer USDC from Coinbase.
  - NOTE: if Coinbase does not have enough balance, the script will fail!
- Convert USDC to USD
- Buy BTC-USD
- Buy ETH-USD
- (Optional) Ping healthcheck URLs

Note that if your Coinbase Pro account already has USD, it won't use it. It does not optimize for that.

## Setup

1. Copy and create a `sandbox.env` and a `prod.env` from `template.env`. Look at comments for values.

1. Set up recurring USDC purchases in Coinbase.com (not Pro). Make sure to always have enough in there. For example, if you're buying $20 of Crypto once a week, then make sure you always keep at least $20 of USDC in your Coinbase.com account.

## Build and Run

### Run

Set `ENV_FILE_PATH` to the file that contains the env vars.

```
$ ENV_FILE_PATH=sandbox.env go run main.go
```

## Build

Set `ENV_FILE_PATH` to the file that contains the env vars. Set `GOOS` to your target platform.

```
$ GOOS=linux go build -ldflags="-d -s -w" -o hodl main.go && chmod +x hodl
$ ENV_FILE_PATH=sandbox.env ./hodl
```

## Cron

You can use cron to run the script. Example to run it every other day:

```
DATEVAR=date +%Y-%m-%d
0 * */2 * * ENV_FILE_PATH=/home/prod.env /home/hodl >> /home/cron-$($DATEVAR).log 2>&1
```

## TODO

- [ ] When transferring, account for trading fees. If you start with 0 USD, then exact USDC won't be enough.

- [ ] Optimize USD use by checking that too.
