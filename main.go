// Heavily based on https://github.com/victoriadrake/ephemeral/blob/master/main.go
package main

import (
    "fmt"
    "log"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/ChimeraCoder/anaconda"
    "github.com/joho/godotenv"
)

var (
    dryRun bool
)

func pingHealthcheck(url string) {
    if url == "" {
        return
    }

    _, err := http.Head(url)
    if err != nil {
        log.Printf("Failed pinging URL: %s, %s", url, err)
    }
}

func getWhitelist() []string {
    v := os.Getenv("WHITELIST")

    if v == "" {
        return make([]string, 0)
    }

    return strings.Split(v, ":")
}

func getTimeline(api *anaconda.TwitterApi, maxId string) ([]anaconda.Tweet, error) {
    args := url.Values{}
    args.Add("count", "200")        // Twitter only returns most recent 20 tweets by default, so override
    args.Add("include_rts", "true") // When using count argument, RTs are excluded, so include them as recommended
    if len(maxId) > 0 {
        args.Set("max_id", maxId)
    }

    timeline, err := api.GetUserTimeline(args)
    if err != nil {
        log.Printf("error while getting timeline %+v", err)
        return make([]anaconda.Tweet, 0), err
    }
    return timeline, nil
}

func getFaves(api *anaconda.TwitterApi, maxId string) ([]anaconda.Tweet, error) {
    args := url.Values{}
    args.Add("count", "200") // Twitter only returns most recent 20 tweets by default, so override
    if len(maxId) > 0 {
        args.Set("max_id", maxId)
    }

    faves, err := api.GetFavorites(args)
    if err != nil {
        log.Printf("error while getting favorites %+v", err)
        return make([]anaconda.Tweet, 0), err
    }
    return faves, nil
}

func isWhitelisted(id int64, text string) bool {
    whitelist := getWhitelist()
    tweetID := strconv.FormatInt(id, 10)
    for _, w := range whitelist {
        if w == tweetID || strings.Contains(text, w) {
            return true
        }
    }
    return false
}

func deleteFromTimeline(api *anaconda.TwitterApi, ageLimit time.Duration) error {
    deletedCount := 0
    maxId := ""

    for i := 1; i <= 10; i++ {
        timeline, err := getTimeline(api, maxId)
        if err != nil {
            log.Print("could not get timeline", err)
            return err
        }
        log.Printf("timeline length %d", len(timeline))

        for _, t := range timeline {
            createdTime, err := t.CreatedAtTime()
            if err != nil {
                log.Print("could not parse time ", err)
                return err
            } else {
                if time.Since(createdTime) > ageLimit && !isWhitelisted(t.Id, t.Text) {
                    deletedCount += 1
                    var err error
                    if t.Retweeted {
                        log.Printf("UNRETWEETING TWEET (was %vh old): %v #%d - %s\n", time.Since(createdTime).Hours(), createdTime, t.Id, t.Text)
                        if !dryRun {
                            _, err = api.UnRetweet(t.Id, true)
                            time.Sleep(2 * time.Second)
                        }
                    } else if !t.Favorited {
                        log.Printf("DELETING TWEET (was %vh old): %v #%d - %s\n", time.Since(createdTime).Hours(), createdTime, t.Id, t.Text)
                        if !dryRun {
                            _, err = api.DeleteTweet(t.Id, true)
                            time.Sleep(2 * time.Second)
                        }
                    }

                    if err != nil {
                        log.Print("failed to clean up: ", err)
                        return err
                    }
                }
            }
            maxId = fmt.Sprintf("%d", t.Id)
        }
    }

    log.Printf("=====>>> deleted %d tweets", deletedCount)
    return nil
}

func unFavorite(api *anaconda.TwitterApi, ageLimit time.Duration) error {
    deletedCount := 0
    maxId := ""

    for i := 1; i <= 10; i++ {
        faves, err := getFaves(api, maxId)
        if err != nil {
            log.Print("could not get favorites", err)
            return err
        }
        log.Printf("favorites length %d", len(faves))

        for _, t := range faves {
            createdTime, err := t.CreatedAtTime()
            if err != nil {
                log.Print("could not parse time ", err)
                return err
            } else {
                if time.Since(createdTime) > ageLimit && !isWhitelisted(t.Id, t.Text) {
                    deletedCount += 1
                    var err error
                    if t.Favorited {
                        log.Printf("UNFAVORITING TWEET (was %vh old): %v #%d - %s\n", time.Since(createdTime).Hours(), createdTime, t.Id, t.Text)
                        if !dryRun {
                            _, err = api.Unfavorite(t.Id)
                            time.Sleep(2 * time.Second)
                        }
                    }
                    if err != nil {
                        if err, ok := err.(*anaconda.ApiError); ok {
                            if err.StatusCode == 404 {
                                log.Print("tweet not found. got 404, skipping: %+v", err)
                                continue
                            }
                        } else {
                            log.Print("failed to clean up: ", err)
                            return err
                        }
                    }
                }
            }
            maxId = fmt.Sprintf("%d", t.Id)
        }
    }
    log.Printf("=====>>> unfavorited %d tweets", deletedCount)
    return nil
}

func main() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    log.Printf("==============================================")
    defer log.Printf("==============================================")

    var err error

    err = godotenv.Load(os.Getenv("ENV_FILE_PATH"))
    if err != nil {
        log.Fatal("Error loading .env file")
    }

    dryRun, err = strconv.ParseBool(os.Getenv("DRY_RUN"))
    if err != nil {
        log.Fatalf("could not parse DRY_RUN")
    }
    log.Printf("dryRun is set to %t", dryRun)

    anaconda.SetConsumerKey(os.Getenv("TWITTER_CONSUMER_KEY"))
    anaconda.SetConsumerSecret(os.Getenv("TWITTER_CONSUMER_SECRET"))
    api := anaconda.NewTwitterApi(os.Getenv("TWITTER_ACCESS_TOKEN"), os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"))
    api.SetLogger(anaconda.BasicLogger)

    h, err := time.ParseDuration(os.Getenv("MAX_TWEET_AGE"))
    if err != nil {
        log.Fatal("invalid value of MAX_TWEET_AGE specified")
    }

    log.Printf("START DELETING TWEETS")
    err1 := deleteFromTimeline(api, h)

    log.Printf("START REMOVING LIKES")
    err2 := unFavorite(api, h)

    if err1 == nil && err2 == nil {
        pingHealthcheck(os.Getenv("HEALTHCHECK_URL"))
    }

    log.Printf("Done with this round...")
}
