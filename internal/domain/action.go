package domain

import "time"

type Action struct {
	ID             string    `json:"id"`
	OrderID        string    `json:"orderId"`
	Type           string    `json:"type"`
	Actor          string    `json:"actor"`
	IdempotencyKey string    `json:"idempotencyKey"`
	CreatedAt      time.Time `json:"createdAt"`
}
