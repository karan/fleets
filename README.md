# fleets

*Nothing is forever. Except the Internet.*

Automatically delete tweets, retweets, and favorites.

Heavily based on https://github.com/victoriadrake/ephemeral.

## How it works

Depending on the schedule you set, this bot will:

- Read config file
- Get your timeline
- Delete all tweets older than `MAX_TWEET_AGE`
- Unfavorite tweets older than `MAX_TWEET_AGE`
- (Optional) Ping healthcheck URLs

You can set `DRY_RUN=true` to print all actions it will take instead of actually deleting tweets.

The tool does not take into account Twitter rate limits, but it does sleep for a few seconds between each Twitter API call.

## Setup

Copy and create a `prod.env` from `template.env`. Look at comments for values (then delete the comments).

## Build and Run

### Run

Set `ENV_FILE_PATH` to the file that contains the env vars.

```
$ ENV_FILE_PATH=sandbox.env go run main.go
```

## Build

Set `ENV_FILE_PATH` to the file that contains the env vars. Set `GOOS` to your target platform.

```
$ GOOS=linux go build -ldflags="-d -s -w" -o fleets main.go && chmod +x fleets
$ ENV_FILE_PATH=sandbox.env ./fleets
```

## Cron

You can use cron to run the script. Example to run it every day at 4pm:

```
DATEVAR=date +%Y-%m-%d
0 16 * * * ENV_FILE_PATH=/home/prod.env /home/fleets >> /home/cron-$($DATEVAR).log 2>&1
```

## Running using Nix

If you have Nix with [Flakes](https://nixos.wiki/wiki/Flakes) enabled, you can run this program directly from GitHub:

```sh
nix run github:karan/fleets
```
