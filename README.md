# hodl

🚀To the moon 🚀

AWS Lambda functions to do DCA (Dollar Cost Averaging) for BTC (Bitcoin) on Coinbase Pro.

## Modules

### fiat

Deposits some USD using ACH. A bank account must exist in the Coinbase Account.

### moon

Buys BTC if there's enough available cash in the account.

## Setup

1. Create an AWS account, and Coinbase Pro account. Add a bank account to the latter.

1. Create a [`lambda-role`](https://docs.aws.amazon.com/lambda/latest/dg/lambda-intro-execution-role.html#permissions-executionrole-console). Copy the `Role ARN`.

    ![](https://i.imgur.com/5GXf25X.png)

1. Find out your Coinbase USD account ID and ACH bank payment method id (hint: look at the Developer Console).

1. Look at the `main.go` file of `fiat` and `moon` for the environment variables that need to be set on each function.

1. For each function, modify `deploy.sh` with the Role ARN.

1. For each function, copy `sandbox.env.json.template` and create `sandbox.env.json` and `prod.env.json`. Fill in the values.

1. For each function, run `deploy.sh sandbox|prod`.

1. For each, [create a schedule](https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/RunLambdaSchedule.html). For `fiat`, I do every day, and for `moon`, I do once a week.

1. 🚀To the moon 🚀

![](https://i.imgur.com/h0nfgDi.png)

## Improvements

* Make `fiat` run every day, and only deposit if there's not been a deposit in the last 7 days.
* Make `moon` run every day, and only buy if there's not been a trade in the last 7 days.
