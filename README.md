# hodl

🚀To the moon 🚀

AWS Lambda functions to do DCA (Dollar Cost Averaging) for BTC (Bitcoin) on Coinbase Pro.

## Modules

### fiat

Deposits some USD using ACH

### moon

Buys BTC

## Setup

1. Create an AWS account, and Coinbase Pro account. Add a bank account to the latter.

2. Find out your Coinbase USD account ID and ACH bank payment method id (hint: look at the Developer Console).

3. Look at the `main.go` file of `fiat` and `moon` for the environment variables that need to be set on each function.

4. For each function, run `deploy.sh`.

5. 🚀To the moon 🚀
