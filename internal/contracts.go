package internal

import (
	"context"
)

//go:generate /Users/lionariman/go/bin/mockgen -source=contracts.go -destination=internal/mocks/mock_contracts.go -package=mocks OrderCache OrderRepository

type OrderCache interface {
	Get(string) (*Order, bool)
	Set(*Order)
	Warm([]*Order)
	Delete(string)
	DeleteAllItems()
}

type OrderRepository interface {
	Get(context.Context, string) (*Order, bool, error)
	ListRecentIDs(context.Context, int) ([]string, error)
	LoadRecent(context.Context, int) ([]*Order, error)
	Upsert(context.Context, *Order) error
}
