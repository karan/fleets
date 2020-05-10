// package main makes sure there's enough money in Coinbase Pro account.
// The following env vars must be set:
// - COINBASE_PRO_BASEURL
// - COINBASE_PRO_PASSPHRASE
// - COINBASE_PRO_KEY
// - COINBASE_PRO_SECRET
// - COINBASE_USD_ACCOUNT_ID (Account ID that holds USD in Coinbase)
// - USD_THRESHOLD_TO_BUY (Minimum amount of money that must exist in the account)
// - COINBASE_PAYMENT_METHOD_ID (Account ID for ACH bank account transfer)
// - USD_AMOUNT_TO_TRANSFER (This much to transfer)

package main

import (
    "context"
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

// Monkey patch Deposit.
type DepositInfo struct {
    ID       string  `json:"id"`
    Amount   float64 `json:"amount,string"`
    Currency string  `json:"currency"`
    PayoutAt string  `json:"payout_at"`
}

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

    balanceF64, _ := balance.Float64()
    availableF64, _ := available.Float64()
    // Publish metric with balance and available
    _, err = svc.PutMetricData(&cloudwatch.PutMetricDataInput{
        Namespace: aws.String("HODL/Fiat"),
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

    threshold, err := decimal.NewFromString(os.Getenv("USD_THRESHOLD_TO_BUY"))
    if err != nil {
        return "", err
    }

    haveEnoughMoney := available.GreaterThan(threshold)

    log.Printf("balance: %s, available: %s, threshold: %s, haveEnoughMoney: %t", balance.String(), available.String(), threshold.String(), haveEnoughMoney)

    if haveEnoughMoney {
        log.Printf("No need to buy more... BYEEE")
        return "Have enough money", nil
    }

    transferAmount, err := decimal.NewFromString(os.Getenv("USD_AMOUNT_TO_TRANSFER"))
    if err != nil {
        return "", err
    }

    transferAmountFloat64, _ := transferAmount.Float64()

    // Start a transfer
    log.Printf("Depositing %s", transferAmount.String())

    resp := DepositInfo{}
    req := make(map[string]interface{})
    req["amount"] = transferAmountFloat64
    req["currency"] = "USD"
    req["payment_method_id"] = os.Getenv("COINBASE_PAYMENT_METHOD_ID")

    log.Printf("req = %+v", req)
    res, err := client.Request("POST", fmt.Sprintf("/deposits/payment-method"), req, &resp)
    if err != nil {
        log.Printf("res = %+v", res)
        log.Printf("resp = %+v", resp)
        return "", nil
    }

    // Publish metric with how much transfer was placed
    _, err = svc.PutMetricData(&cloudwatch.PutMetricDataInput{
        Namespace: aws.String("HODL/Fiat"),
        MetricData: []*cloudwatch.MetricDatum{
            {
                MetricName: aws.String("Deposit Count"),
                Unit:       aws.String("Count"),
                Value:      aws.Float64(1.00),
                Dimensions: []*cloudwatch.Dimension{
                    {
                        Name:  aws.String("FunctionName"),
                        Value: aws.String(lambdacontext.FunctionName),
                    },
                },
            },
            {
                MetricName: aws.String("Deposit Amount"),
                Unit:       aws.String("Count"),
                Value:      aws.Float64(transferAmountFloat64),
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

    log.Printf("Successully deposited money")
    return "Successully deposited money", nil
}

func main() {
    lambda.Start(HandleRequest)
}
