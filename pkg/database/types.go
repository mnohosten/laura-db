package database

import "github.com/mnohosten/laura-db/pkg/query"

// QueryOptions holds options for queries
type QueryOptions struct {
	Projection map[string]bool
	Sort       []query.SortField
	Limit      int
	Skip       int
}
