package alerter

import (
	"context"
	"fmt"

	"github.com/yashtandon019/kost/pkg/explainer"
)

// SlackAlerter sends anomaly alerts to a Slack channel via webhook.
type SlackAlerter struct {
	// WebhookURL is the Slack incoming webhook URL.
	WebhookURL string

	// Channel is the optional channel override.
	Channel string
}

// NewSlackAlerter creates a new SlackAlerter.
func NewSlackAlerter(webhookURL, channel string) *SlackAlerter {
	return &SlackAlerter{
		WebhookURL: webhookURL,
		Channel:    channel,
	}
}

// Alert sends the anomaly explanation to Slack.
func (a *SlackAlerter) Alert(_ context.Context, _ explainer.Explanation) error {
	// TODO: Implement Slack webhook integration.
	// 1. Format the explanation into a Slack Block Kit message
	// 2. POST to the webhook URL
	// 3. Handle rate limiting and retries
	return fmt.Errorf("slack alerter not yet implemented")
}
