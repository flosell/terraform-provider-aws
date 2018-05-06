package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/budgets"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"regexp"
)

func TestAccAWSBudgetsBudgetNotification_basic(t *testing.T) {
	budgetName := fmt.Sprintf("tf-acc-%d", acctest.RandInt())

	oneEmail := `["terraform-aws-budget-notifications-first@example.com"]`
	oneOtherEmail := `["terraform-aws-budget-notifications@example.com"]`
	twoEmails := `["terraform-aws-budget-notifications@example.com", "terraform-aws-budget-notifications-2@example.com"]`

	oneTopic := `["${aws_sns_topic.budget_notifications.arn}"]`
	noTopic := `[]`
	noEmails := `[]`

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAcmCertificateDestroy,
		Steps: []resource.TestStep{
			// Can't create without at least one subscriber
			resource.TestStep{
				Config:      budgetNotificationConfig(budgetName, noEmails, noTopic),
				ExpectError: regexp.MustCompile(`Notification must have at least one subscriber`),
			},
			// Basic Notification with only email
			resource.TestStep{
				Config: budgetNotificationConfig(budgetName, oneEmail, noTopic),
			},
			// Change only subscriber to a different e-mail
			resource.TestStep{
				Config: budgetNotificationConfig(budgetName, oneOtherEmail, noTopic),
			},
			// Add a second e-mail and a topic
			resource.TestStep{
				Config: budgetNotificationConfig(budgetName, twoEmails, oneTopic),
			},
			// Delete both E-Mails
			resource.TestStep{
				Config: budgetNotificationConfig(budgetName, noEmails, oneTopic),
			},
			// Swap one Topic fo one E-Mail
			resource.TestStep{
				Config: budgetNotificationConfig(budgetName, oneEmail, noTopic),
			},
			// Can't update without at least one subscriber
			resource.TestStep{
				Config:      budgetNotificationConfig(budgetName, noEmails, noTopic),
				ExpectError: regexp.MustCompile(`Notification must have at least one subscriber`),
			},
			// Update all non-subscription parameters
			resource.TestStep{
				Config: budgetNotificationConfigWithCompletelyDifferentParameters(budgetName),
			},
		},
	})

}

func budgetNotificationConfig(budgetName string, emails string, topics string) string {
	return `
resource "aws_sns_topic" "budget_notifications" {
  name_prefix = "user-updates-topic"
}

resource "aws_budgets_budget" "foo" {
	name = "` + budgetName + `"
	budget_type = "` + budgets.BudgetTypeCost + `"
	limit_amount = 1000
	limit_unit = "USD"
	time_period_start = "2006-01-02_15:04"
	time_unit = "MONTHLY"
}

resource "aws_budgets_budget_notification" "foo" {
	budget_name = "${aws_budgets_budget.foo.name}"
	comparison_operator = "` + budgets.ComparisonOperatorGreaterThan + `"
	threshold = 1000
	threshold_type = "` + budgets.ThresholdTypeAbsoluteValue + `"
	notification_type = "` + budgets.NotificationTypeForecasted + `"
	subscriber_email_addresses = ` + emails + `
	subscriber_sns_topic_arns = ` + topics + `
}
`
}

func budgetNotificationConfigWithCompletelyDifferentParameters(budgetName string) string {
	return `
resource "aws_sns_topic" "budget_notifications" {
  name_prefix = "user-updates-topic"
}

resource "aws_budgets_budget" "foo" {
	name = "` + budgetName + `"
	budget_type = "` + budgets.BudgetTypeCost + `"
	limit_amount = 1000
	limit_unit = "USD"
	time_period_start = "2006-01-02_15:04"
	time_unit = "MONTHLY"
}

resource "aws_budgets_budget_notification" "foo" {
	budget_name = "${aws_budgets_budget.foo.name}"
	comparison_operator = "` + budgets.ComparisonOperatorLessThan + `"
	threshold = 2000
	threshold_type = "` + budgets.ThresholdTypePercentage + `"
	notification_type = "` + budgets.NotificationTypeActual + `"
	subscriber_email_addresses = ["terraform-aws-budget-notifications-first@example.com"]
	subscriber_sns_topic_arns = []
}
`
}
