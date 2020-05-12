// package main buys some BTC and ETH.

package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/joho/godotenv"
    coinbasepro "github.com/preichenberger/go-coinbasepro/v2"
    "github.com/shopspring/decimal"
)

var (
    client *coinbasepro.Client
)

type DepositWithdrawalInfo struct {
    ID       string  `json:"id"`
    Amount   float64 `json:"amount,string"`
    Currency string  `json:"currency"`
    PayoutAt string  `json:"payout_at"`
}

type ConversionInfo struct {
    ID     string  `json:"id"`
    From   string  `json:"from"`
    To     float64 `json:"amount,to"`
    Amount string  `json:"amount"`
}

// Return balance and available for the account
// COINBASE_USDC_ACCOUNT_ID must be set.
func coinbaseBalanace() (decimal.Decimal, decimal.Decimal) {
    account, err := client.GetAccount(os.Getenv("COINBASE_USDC_ACCOUNT_ID"))
    if err != nil {
        log.Fatalf("error while getting account: %+v", err)
    }

    log.Printf("account = %+v", account)

    accountBalance, err := decimal.NewFromString(account.Balance)
    if err != nil {
        log.Fatalf("could not convert balance to decimal: %+v", err)
    }
    accountAvailable, err := decimal.NewFromString(account.Available)
    if err != nil {
        log.Fatalf("could not convert balance to decimal: %+v", err)
    }
    log.Printf("balance: %s, available: %s", accountBalance.String(), accountAvailable.String())
    return accountBalance, accountAvailable
}

// Start a transfer from Coinbase account for amount
// COINBASE_SOURCE_ACCOUNT_ID env var must be set.
func coinbaseTransfer(transferAmount decimal.Decimal) error {
    log.Printf("Transfering %s USDC from Coinbase to Pro", transferAmount.String())

    transferAmountF64, _ := transferAmount.Float64()

    resp := DepositWithdrawalInfo{}
    req := make(map[string]interface{})
    req["amount"] = transferAmountF64
    req["currency"] = "USDC"
    req["coinbase_account_id"] = os.Getenv("COINBASE_SOURCE_ACCOUNT_ID")

    log.Printf("req = %+v", req)
    res, err := client.Request("POST", fmt.Sprintf("/deposits/coinbase-account"), req, &resp)
    if err != nil {
        log.Printf("res = %+v", res)
        log.Printf("resp = %+v", resp)
        return err
    }
    return nil
}

// Convert USDC to USD.
func coinbaseConversion(amount decimal.Decimal) error {
    log.Printf("Converting %s USDC to USD", amount.String())

    resp := ConversionInfo{}
    req := make(map[string]interface{})
    req["amount"] = amount.String()
    req["from"] = "USDC"
    req["to"] = "USD"

    log.Printf("req = %+v", req)
    res, err := client.Request("POST", fmt.Sprintf("/conversions"), req, &resp)
    if err != nil {
        log.Printf("res = %+v", res)
        log.Printf("resp = %+v", resp)
        return err
    }
    return nil
}

// Rounds size based on baseIncrement
// 53.0234, 1.00 => 53
// 53.2343255, 1.001 => 53.234
func roundToDecimal(size decimal.Decimal, baseIncrement string) decimal.Decimal {
    splits := strings.Split(baseIncrement, ".")
    decimals := splits[len(splits)-1]

    // Default to 0 decimals (whole ints)
    decimalsCount := 0
    // If the decimal part has 1, we round to that last known place.
    if strings.Contains(decimals, "1") {
        decimalsCount = 1 + len(strings.Split(decimals, "1")[0])
    }

    tenDecimal := decimal.NewFromFloat(10.0)
    decimalsCountDecimal := decimal.NewFromInt(int64(decimalsCount))

    // int(size * 10**decimals_count) / 10**decimals_count
    numerator := decimal.NewFromInt(size.Mul(tenDecimal.Pow(decimalsCountDecimal)).IntPart())
    return numerator.Div(tenDecimal.Pow(decimalsCountDecimal))
}

// book: "BTC-USDC"
func placeOrder(bookName string, buyAmountUSD decimal.Decimal, baseIncrement string) {
    book, err := client.GetBook(bookName, 1)
    if err != nil {
        log.Fatalf("Failed trying to get book, ", err)
    }

    lastPrice, err := decimal.NewFromString(book.Bids[0].Price)
    if err != nil {
        log.Fatalf("Failed trying to make last price, ", err)
    }
    log.Printf("Last price: %s", lastPrice.String())

    // DivRound divides and rounds to a given precision for min order size.
    sizeDecimal := buyAmountUSD.DivRound(lastPrice, 8)
    sizeDecimal = roundToDecimal(sizeDecimal, baseIncrement)

    // Account for minimum order size
    if sizeDecimal.LessThan(decimal.NewFromFloat(0.001)) {
        log.Fatalf("Cannot place order for %s %s.. min size is 0.001", sizeDecimal.String(), bookName)
    }

    log.Printf("Buying %s %s", sizeDecimal.String(), bookName)

    order := coinbasepro.Order{
        Type:      "limit",
        Price:     lastPrice.String(),
        Size:      sizeDecimal.String(),
        Side:      "buy",
        ProductID: bookName,
    }

    savedOrder, err := client.CreateOrder(&order)
    if err != nil {
        log.Fatalf("Failed trying to create order, ", err)
    }
    log.Printf("Order sent successfully %+v", savedOrder)
}

func pingHealthcheck(url string) {
    if url == "" {
        return
    }

    _, err := http.Head(url)
    if err != nil {
        log.Printf("Failed pinging URL: %s, %s", url, err)
    }
}

func main() {
    err := godotenv.Load(os.Getenv("ENV_FILE_PATH"))
    if err != nil {
        log.Fatal("Error loading .env file")
    }

    client = coinbasepro.NewClient()
    log.Printf("Client initialized")

    _, accountAvailable := coinbaseBalanace()

    btcBuyAmount, err := decimal.NewFromString(os.Getenv("BTC_BUY_AMOUNT_IN_USD"))
    if err != nil {
        log.Fatalf("could not convert balance to decimal: %+v", err)
    }
    ethBuyAmount, err := decimal.NewFromString(os.Getenv("ETH_BUY_AMOUNT_IN_USD"))
    if err != nil {
        log.Fatalf("could not convert balance to decimal: %+v", err)
    }
    totalBuyAmount := btcBuyAmount.Add(ethBuyAmount)
    // Adding 5% fuzz factor for price fluctuations and float accuracy.
    minAmountNeeded := totalBuyAmount.Mul(decimal.NewFromFloat(1.05))
    log.Printf("totalBuyAmount: %s, minAmountNeeded: %s", totalBuyAmount.String(), minAmountNeeded.String())

    // If there's not enough, start a transfer
    haveEnoughMoney := accountAvailable.GreaterThan(minAmountNeeded)
    if !haveEnoughMoney {
        err := coinbaseTransfer(totalBuyAmount)
        if err != nil {
            log.Fatalf("Failed to transfer: %+v", err)
        }
    }

    log.Printf("Waiting for a few seconds before proceeding..")
    time.Sleep(5 * time.Second)

    log.Printf("Checking balance again...")
    _, accountAvailable = coinbaseBalanace()
    haveEnoughMoney = accountAvailable.GreaterThan(minAmountNeeded)
    if !haveEnoughMoney {
        log.Fatalf("Still not enough money... Dying...")
    }

    err = coinbaseConversion(totalBuyAmount)
    if err != nil {
        log.Fatalf("Failed to convert: %+v", err)
    }
    log.Printf("Waiting for a few seconds before proceeding..")
    time.Sleep(5 * time.Second)

    placeOrder(os.Getenv("BTC_BOOK_NAME"), btcBuyAmount, os.Getenv("BTC_BASE_INCREMENT"))
    pingHealthcheck(os.Getenv("HEALTHCHECK_BTC_BOUGHT"))
    placeOrder(os.Getenv("ETH_BOOK_NAME"), ethBuyAmount, os.Getenv("ETH_BASE_INCREMENT"))
    pingHealthcheck(os.Getenv("HEALTHCHECK_ETH_BOUGHT"))

    log.Printf("Done with this round...")
}
