// Package page holds the pagination types used by every list endpoint.
package page

const (
	DefaultSize = 20
	MaxSize     = 100
)

// Query is a normalized page request.
type Query struct {
	Page     int
	PageSize int
}

// New clamps page and size to sane bounds.
func New(page, size int) Query {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = DefaultSize
	}
	if size > MaxSize {
		size = MaxSize
	}
	return Query{Page: page, PageSize: size}
}

// Offset is the number of rows to skip.
func (q Query) Offset() int { return (q.Page - 1) * q.PageSize }

// Result is one page of items plus the total count.
type Result[T any] struct {
	Items    []T
	Page     int
	PageSize int
	Total    int64
}

// TotalPages returns the number of pages for the total count.
func (r Result[T]) TotalPages() int {
	if r.PageSize == 0 {
		return 0
	}
	return int((r.Total + int64(r.PageSize) - 1) / int64(r.PageSize))
}
