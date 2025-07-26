// Package example demonstrates complex serialization chains with invariance testing
package example

import (
	"time"
)

// Complex nested types demonstration using concrete types
// This demonstrates the pattern FetchResult = Result[Option[User], APIError]
// Note: User and APIError types are defined in datas.go

// Option[User] - demonstrating Option pattern with concrete User type
//go:tag mkunion:"OptionUser"
type (
	SomeUser struct {
		Value User `json:"value"`
	}
	NoneUser struct{}
)

// Result[OptionUser, APIError] - demonstrating Result pattern
//go:tag mkunion:"DemoFetchResult"
type (
	FetchSuccess struct {
		Value OptionUser `json:"value"`
	}
	FetchError struct {
		Error APIError `json:"error"`
	}
)

//go:tag serde:"request-log"
type RequestLog struct {
	RequestID string           `json:"request_id"`
	Timestamp time.Time        `json:"timestamp"`
	Result    DemoFetchResult  `json:"result"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// PagedResult[User] - demonstrating paged results pattern
//go:tag mkunion:"PagedUserResult"
type (
	PagedUserSuccess struct {
		Data       []User `json:"data"`
		TotalCount int64  `json:"total_count"`
		Page       int    `json:"page"`
		PageSize   int    `json:"page_size"`
	}
	PagedUserError struct {
		Error APIError `json:"error"`
	}
)

// Result[PagedUserResult, APIError] - even more complex nesting
//go:tag mkunion:"UserSearchResult"
type (
	SearchSuccess struct {
		Value PagedUserResult `json:"value"`
	}
	SearchError struct {
		Error APIError `json:"error"`
	}
)

//go:tag serde:"search-response"
type SearchResponse struct {
	Query      string           `json:"query"`
	Results    UserSearchResult `json:"results"`
	ExecutedAt time.Time        `json:"executed_at"`
}

// Additional complex nested structure for demonstration
//go:tag mkunion:"NestedResult"
type (
	NestedSuccess struct {
		Primary   DemoFetchResult      `json:"primary"`
		Secondary UserSearchResult `json:"secondary"`
		Metadata  map[string]interface{} `json:"metadata"`
	}
	NestedFailure struct {
		Errors []APIError `json:"errors"`
		PartialData map[string]interface{} `json:"partial_data,omitempty"`
	}
)

//go:tag serde:"complex-operation"
type ComplexOperation struct {
	OperationID string       `json:"operation_id"`
	StartTime   time.Time    `json:"start_time"`
	EndTime     *time.Time   `json:"end_time,omitempty"`
	Result      NestedResult `json:"result"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
}