// package main buys some BTC.
// The following env vars must be set:
// - COINBASE_PRO_BASEURL
// - COINBASE_PRO_PASSPHRASE
// - COINBASE_PRO_KEY
// - COINBASE_PRO_SECRET
// - COINBASE_USD_ACCOUNT_ID (Account ID that holds USD in Coinbase)
// - USD_BTC_BUY_AMOUNT (Buy this much $ of BTC)

package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "os"

    "github.com/aws/aws-lambda-go/lambda"
    coinbasepro "github.com/preichenberger/go-coinbasepro/v2"
    "github.com/shopspring/decimal"
)

func HandleRequest(ctx context.Context) (string, error) {
    client := coinbasepro.NewClient()

    log.Printf("Client initialized")

    account, err := client.GetAccount(os.Getenv("COINBASE_USD_ACCOUNT_ID"))
    if err != nil {
        log.Printf("error while getting account: %+v", err)
        return err.Error(), err
    }

    log.Printf("account = %+v", account)

    balance, err := decimal.NewFromString(account.Balance)
    if err != nil {
        return "", err
    }

    available, err := decimal.NewFromString(account.Available)
    if err != nil {
        return "", err
    }

    threshold, err := decimal.NewFromString(os.Getenv("USD_BTC_BUY_AMOUNT"))
    if err != nil {
        return "", err
    }

    // Adding 5% fuzz factor for price fluctuations and float accuracy.
    threshold = threshold.Mul(decimal.NewFromFloat(1.05))

    haveEnoughMoney := available.GreaterThan(threshold)

    log.Printf("balance: %s, available: %s, threshold: %s, haveEnoughMoney: %t", balance.String(), available.String(), threshold.String(), haveEnoughMoney)

    if !haveEnoughMoney {
        log.Printf("I don't have enough money! Crashing...")
        return "", errors.New(fmt.Sprintf("Did not have enough money to buy. balance: %s, available: %s, threshold: %s", balance.String(), available.String(), threshold.String()))
    }

    // Place an order
    book, err := client.GetBook("BTC-USD", 1)
    if err != nil {
        return "", err
    }

    log.Printf("Book: %+v", book)

    lastPrice, err := decimal.NewFromString(book.Bids[0].Price)
    if err != nil {
        return "", err
    }

    log.Printf("Last price: %s", lastPrice.String())

    sizeDecimal := threshold.DivRound(lastPrice, 8)

    log.Printf("Buying %s BTC", sizeDecimal.String())

    order := coinbasepro.Order{
        // Keep a $10 margin for spikes, otherwise this should be like Market.
        Price:     lastPrice.Add(decimal.NewFromFloat(10.00)).String(),
        Size:      sizeDecimal.String(),
        Side:      "buy",
        ProductID: "BTC-USD",
    }

    savedOrder, err := client.CreateOrder(&order)
    if err != nil {
        return "", err
    }

    log.Printf("Order sent successfully %+v", savedOrder)

    return "Successully bought BTC", nil
}

func main() {
    lambda.Start(HandleRequest)
}
