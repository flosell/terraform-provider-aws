package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/budgets"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

func resourceAwsBudgetsBudgetNotification() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsBudgetsBudgetNotificationCreate,
		Read:   resourceAwsBudgetsBudgetNotificationRead,
		Update: resourceAwsBudgetsBudgetNotificationUpdate,
		Delete: resourceAwsBudgetsBudgetNotificationDelete,

		Schema: map[string]*schema.Schema{
			"budget_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"account_id": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsAccountId,
			},
			"comparison_operator": {
				Type:     schema.TypeString,
				Required: true,
			},
			"threshold": {
				Type:     schema.TypeFloat,
				Required: true,
			},
			"threshold_type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"notification_type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"subscriber_email_addresses": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"subscriber_sns_topic_arns": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceAwsBudgetsBudgetNotificationCreate(d *schema.ResourceData, meta interface{}) error {
	budgetName := d.Get("budget_name").(string)
	accountID := budgetNotificationAccountId(d, meta)

	client := meta.(*AWSClient).budgetconn
	subscribers := expandSubscribers(d)

	if len(subscribers) == 0 {
		return fmt.Errorf("Notification must have at least one subscriber!")
	}
	_, err := client.CreateNotification(&budgets.CreateNotificationInput{
		BudgetName:   aws.String(d.Get("budget_name").(string)),
		AccountId:    aws.String(budgetNotificationAccountId(d, meta)),
		Notification: expandNotification(d),
		Subscribers:  subscribers,
	})

	if err != nil {
		return err
	}

	d.SetId(resource.PrefixedUniqueId(fmt.Sprintf("%s:%s:s", accountID, budgetName)))

	return resourceAwsBudgetsBudgetNotificationRead(d, meta)
}

func expandNotification(d *schema.ResourceData) *budgets.Notification {
	comparisonOperator := d.Get("comparison_operator").(string)
	threshold := d.Get("threshold").(float64)
	thresholdType := d.Get("threshold_type").(string)
	notificationType := d.Get("notification_type").(string)
	notification := &budgets.Notification{
		ComparisonOperator: aws.String(comparisonOperator),
		Threshold:          aws.Float64(threshold),
		ThresholdType:      aws.String(thresholdType),
		NotificationType:   aws.String(notificationType),
	}
	return notification
}

func expandSubscribers(d *schema.ResourceData) []*budgets.Subscriber {
	return append(
		expandSubscribersType(d, "subscriber_sns_topic_arns", budgets.SubscriptionTypeSns),
		expandSubscribersType(d, "subscriber_email_addresses", budgets.SubscriptionTypeEmail)...)
}

func expandSubscribersType(d *schema.ResourceData, key string, subscriptionType string) []*budgets.Subscriber {
	result := make([]*budgets.Subscriber, 0)
	value, ok := d.GetOk(key)
	if ok {

		addrs := expandStringSet(value.(*schema.Set))
		for _, addr := range addrs {
			result = append(result, &budgets.Subscriber{
				SubscriptionType: aws.String(subscriptionType),
				Address:          addr,
			})
		}
	}
	return result
}

func resourceAwsBudgetsBudgetNotificationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).budgetconn

	budgetName := d.Get("budget_name").(string)
	accountID := budgetNotificationAccountId(d, meta)

	describeNotifcationsOutput, err := client.DescribeNotificationsForBudget(&budgets.DescribeNotificationsForBudgetInput{
		BudgetName: aws.String(budgetName),
		AccountId:  aws.String(accountID),
	})

	if err != nil {
		return err
	}

	expectedNotification := expandNotification(d)

	// notifications don't have a proper ID so the only way to identify a notification is looking at all of them and
	// finding an exact match. If not found, we assume the notification isn't there
	for _, notification := range describeNotifcationsOutput.Notifications {
		setNotificationDefaults(notification)
		if isSameNotification(notification, expectedNotification) {
			d.Set("account_id", accountID)
			d.Set("budget_name", budgetName)
			return resourceAwsBudgetsBudgetNotificationSubscriptionsRead(notification, d, meta)
		}
	}

	log.Printf("[WARN] Couldn't find notification, removing from state")

	d.SetId("")

	return nil
}

func resourceAwsBudgetsBudgetNotificationSubscriptionsRead(notification *budgets.Notification, d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).budgetconn
	output, err := client.DescribeSubscribersForNotification(&budgets.DescribeSubscribersForNotificationInput{
		Notification: notification,
		BudgetName:   aws.String(d.Get("budget_name").(string)),
		AccountId:    aws.String(budgetNotificationAccountId(d, meta)),
	})

	if err != nil {
		return err
	}
	emails := make([]string, 0)
	snsArns := make([]string, 0)

	for _, subscriber := range output.Subscribers {
		if *subscriber.SubscriptionType == budgets.SubscriptionTypeEmail {
			emails = append(emails, *subscriber.Address)
		} else if *subscriber.SubscriptionType == budgets.SubscriptionTypeSns {
			snsArns = append(snsArns, *subscriber.Address)
		}
	}

	if err := d.Set("subscriber_email_addresses", emails); err != nil {
		return err
	}
	if err := d.Set("subscriber_sns_topic_arns", snsArns); err != nil {
		return err
	}

	return nil
}

func setNotificationDefaults(notification *budgets.Notification) {
	// The AWS API doesn't seem to return a ThresholdType if it's set to PERCENTAGE
	// Set it manually to make behavior more predictable
	if notification.ThresholdType == nil {
		notification.ThresholdType = aws.String(budgets.ThresholdTypePercentage)
	}
}

func isSameNotification(notification *budgets.Notification, expectedNotification *budgets.Notification) bool {
	return *notification.ThresholdType == *expectedNotification.ThresholdType &&
		*notification.Threshold == *expectedNotification.Threshold &&
		*notification.NotificationType == *expectedNotification.NotificationType &&
		*notification.ComparisonOperator == *expectedNotification.ComparisonOperator
}

func resourceAwsBudgetsBudgetNotificationUpdate(d *schema.ResourceData, meta interface{}) error {
	if len(expandSubscribers(d)) == 0 {
		return fmt.Errorf("Notification must have at least one subscriber!")
	}

	err := resourceAwsBudgetsBudgetNotificationUpdateSubscriptions(d, meta)
	if err != nil {
		return err
	}

	client := meta.(*AWSClient).budgetconn

	if d.HasChange("comparison_operator") ||
		d.HasChange("threshold") ||
		d.HasChange("threshold_type") ||
		d.HasChange("notification_type") {
		oldComparisionOperator, _ := d.GetChange("comparison_operator")
		oldThreshold, _ := d.GetChange("threshold")
		oldThresholdType, _ := d.GetChange("threshold_type")
		oldNotificationType, _ := d.GetChange("notification_type")

		oldNotification := &budgets.Notification{
			ComparisonOperator: aws.String(oldComparisionOperator.(string)),
			Threshold:          aws.Float64(oldThreshold.(float64)),
			ThresholdType:      aws.String(oldThresholdType.(string)),
			NotificationType:   aws.String(oldNotificationType.(string)),
		}

		_, err = client.UpdateNotification(&budgets.UpdateNotificationInput{
			BudgetName:      aws.String(d.Get("budget_name").(string)),
			AccountId:       aws.String(budgetNotificationAccountId(d, meta)),
			NewNotification: expandNotification(d),
			OldNotification: oldNotification,
		})
	}

	return err
}

func resourceAwsBudgetsBudgetNotificationUpdateSubscriptions(d *schema.ResourceData, meta interface{}) error {
	removeEmail, addEmail := diffSubscriptionList(d, "subscriber_email_addresses")
	removeTopic, addTopic := diffSubscriptionList(d, "subscriber_sns_topic_arns")

	if err := resourceAwsBudgetsBudgetNotificationAddSubscriber(addEmail, budgets.SubscriptionTypeEmail, d, meta); err != nil {
		return err
	}

	if err := resourceAwsBudgetsBudgetNotificationAddSubscriber(addTopic, budgets.SubscriptionTypeSns, d, meta); err != nil {
		return err
	}

	if err := resourceAwsBudgetsBudgetNotificationDeleteSubscriber(removeEmail, budgets.SubscriptionTypeEmail, d, meta); err != nil {
		return err
	}

	if err := resourceAwsBudgetsBudgetNotificationDeleteSubscriber(removeTopic, budgets.SubscriptionTypeSns, d, meta); err != nil {
		return err
	}

	return nil
}

func resourceAwsBudgetsBudgetNotificationAddSubscriber(add *schema.Set, subscriptionType string, d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).budgetconn
	for _, addr := range add.List() {
		_, err := client.CreateSubscriber(&budgets.CreateSubscriberInput{
			BudgetName:   aws.String(d.Get("budget_name").(string)),
			AccountId:    aws.String(budgetNotificationAccountId(d, meta)),
			Notification: expandNotification(d),
			Subscriber: &budgets.Subscriber{
				SubscriptionType: aws.String(subscriptionType),
				Address:          aws.String(addr.(string)),
			},
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func resourceAwsBudgetsBudgetNotificationDeleteSubscriber(add *schema.Set, subscriptionType string, d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).budgetconn
	for _, addr := range add.List() {
		_, err := client.DeleteSubscriber(&budgets.DeleteSubscriberInput{
			BudgetName:   aws.String(d.Get("budget_name").(string)),
			AccountId:    aws.String(budgetNotificationAccountId(d, meta)),
			Notification: expandNotification(d),
			Subscriber: &budgets.Subscriber{
				SubscriptionType: aws.String(subscriptionType),
				Address:          aws.String(addr.(string)),
			},
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func diffSubscriptionList(d *schema.ResourceData, key string) (*schema.Set, *schema.Set) {
	oraw, nraw := d.GetChange(key)
	if oraw == nil {
		oraw = new(schema.Set)
	}
	if nraw == nil {
		nraw = new(schema.Set)
	}
	os := oraw.(*schema.Set)
	ns := nraw.(*schema.Set)
	remove := os.Difference(ns)
	add := ns.Difference(os)
	return remove, add
}

func budgetNotificationAccountId(d *schema.ResourceData, meta interface{}) string {
	if v, ok := d.GetOk("account_id"); ok {
		return v.(string)
	} else {
		return meta.(*AWSClient).accountid
	}
}

func resourceAwsBudgetsBudgetNotificationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).budgetconn
	_, err := client.DeleteNotification(&budgets.DeleteNotificationInput{
		BudgetName:   aws.String(d.Get("budget_name").(string)),
		AccountId:    aws.String(budgetNotificationAccountId(d, meta)),
		Notification: expandNotification(d),
	})

	if err != nil {
		return err
	}

	d.SetId("")

	return nil

}
