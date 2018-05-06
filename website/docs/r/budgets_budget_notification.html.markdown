---
layout: "aws"
page_title: "AWS: aws_budgets_budget_notification"
sidebar_current: "docs-aws-resource-budgets-budget_notification"
description: |-
  Provides a budgets budget notification resource.
---

# aws_budgets_budget_notification

Provides a budgets budget notification resource. Use this to define notifications to be sent via E-Mail or SNS once certain conditions in a budget are met, e.g. forcasted cost exceeds a percentage of your budget.

## Example Usage

```hcl
resource "aws_budgets_budget" "cost" {
	name = "` + budgetName + `"
	budget_type = "COST"
	limit_amount = 1000
	limit_unit = "USD"
	time_period_start = "2006-01-02_15:04"
	time_unit = "MONTHLY"
}

resource "aws_budgets_budget_notification" "cost_budget_consumed" {
	budget_name = "${aws_budgets_budget.cost.name}"
	comparison_operator = "GREATER_THAN"
	threshold = 100
	threshold_type = "PERCENTAGE"
	notification_type = "ACTUAL"
	subscriber_email_addresses = ["budget@example.com"]
	subscriber_sns_topic_arns = ["arn:aws:sns:us-west-2:123456789:some-topic"]
}

## Argument Reference

For more detailed documentation about each argument, refer to the [AWS official
documentation](http://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/data-type-budget.html).

The following arguments are supported:

* `account_id` - (Optional) The ID of the target account for budget. Will use current user's account_id by default if omitted.
* `budget_name` - (Required) The name of a budget to be used.
* `comparison_operator` - (Required) Comparison operator to use to evaluate the condition. Can be `LESS_THAN`, `EQUAL_TO` or `GREATER_THAN`.
* `threshold` - (Required) Threshold when the notification should be sent.
* `threshold_type` - (Required) What kind of threshold is defined. Can be `PERCENTAGE` OR `ABSOLUTE_VALUE`.
* `notification_type` - (Required) What kind of budget value to notify on. Can be `ACTUAL` or `FORECASTED`
* `subscriber_email_addresses` - (Optional) E-Mail addresses to notify. Either this or `subscriber_sns_topic_arns` is required.
* `subscriber_sns_topic_arns` - (Optional) SNS topics to notify. Either this or `subscriber_email_addresses` is required.

## Attributes Reference

The following attributes are exported:

* `id` - id of resource.

## Import

Importing Budget Notifications is not supported at the moment
