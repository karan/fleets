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
    "github.com/aws/aws-lambda-go/lambdacontext"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/cloudwatch"
    coinbasepro "github.com/preichenberger/go-coinbasepro/v2"
    "github.com/shopspring/decimal"
)

func HandleRequest(ctx context.Context) (string, error) {
    client := coinbasepro.NewClient()

    // Create new cloudwatch client.
    sess := session.Must(session.NewSessionWithOptions(session.Options{
        SharedConfigState: session.SharedConfigEnable,
    }))
    svc := cloudwatch.New(sess)

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

    balanceF64, _ := balance.Float64()
    availableF64, _ := available.Float64()
    // Publish metric with balance and available
    _, err = svc.PutMetricData(&cloudwatch.PutMetricDataInput{
        Namespace: aws.String("HODL/Moon"),
        MetricData: []*cloudwatch.MetricDatum{
            {
                MetricName: aws.String("Balance"),
                Unit:       aws.String("Count"),
                Value:      aws.Float64(balanceF64),
                Dimensions: []*cloudwatch.Dimension{
                    {
                        Name:  aws.String("FunctionName"),
                        Value: aws.String(lambdacontext.FunctionName),
                    },
                },
            },
            {
                MetricName: aws.String("Available"),
                Unit:       aws.String("Count"),
                Value:      aws.Float64(availableF64),
                Dimensions: []*cloudwatch.Dimension{
                    {
                        Name:  aws.String("FunctionName"),
                        Value: aws.String(lambdacontext.FunctionName),
                    },
                },
            },
        },
    })
    if err != nil {
        log.Println("Error adding metrics:", err.Error())
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

    lastPriceF64, _ := lastPrice.Float64()
    sizeDecimalF64, _ := sizeDecimal.Float64()
    // Publish metric with balance and available
    _, err = svc.PutMetricData(&cloudwatch.PutMetricDataInput{
        Namespace: aws.String("HODL/Moon"),
        MetricData: []*cloudwatch.MetricDatum{
            {
                MetricName: aws.String("BTC Last Price"),
                Unit:       aws.String("Count"),
                Value:      aws.Float64(lastPriceF64),
                Dimensions: []*cloudwatch.Dimension{
                    {
                        Name:  aws.String("FunctionName"),
                        Value: aws.String(lambdacontext.FunctionName),
                    },
                },
            },
            {
                MetricName: aws.String("Order Size"),
                Unit:       aws.String("Count"),
                Value:      aws.Float64(sizeDecimalF64),
                Dimensions: []*cloudwatch.Dimension{
                    {
                        Name:  aws.String("FunctionName"),
                        Value: aws.String(lambdacontext.FunctionName),
                    },
                },
            },
        },
    })
    if err != nil {
        log.Println("Error adding metrics:", err.Error())
    }

    log.Printf("Successully bought BTC")
    return "Successully bought BTC", nil
}

func main() {
    lambda.Start(HandleRequest)
}
