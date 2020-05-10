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

    return "Successully deposited money", nil
}

func main() {
    lambda.Start(HandleRequest)
}
