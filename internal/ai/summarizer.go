package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/chrisuzor/commerce-ops-console/internal/domain"
)

type Summarizer interface {
	Summarize(context.Context, domain.Order) (string, error)
}

// EvidenceSummarizer is deterministic, testable, and safe as the default.
// A model-backed implementation can replace it without changing order logic.
type EvidenceSummarizer struct{}

func (EvidenceSummarizer) Summarize(_ context.Context, order domain.Order) (string, error) {
	if len(order.Issues) == 0 {
		return fmt.Sprintf("Order %s is %s. No active operational issue was detected from the available events.", order.ID, strings.ToLower(string(order.Status))), nil
	}

	parts := make([]string, 0, len(order.Issues))
	for _, issue := range order.Issues {
		parts = append(parts, issue.Explanation)
	}
	return fmt.Sprintf(
		"Order %s requires attention. %s Recommended next step: review the cited timeline evidence before taking an operational action.",
		order.ID,
		strings.Join(parts, " "),
	), nil
}
