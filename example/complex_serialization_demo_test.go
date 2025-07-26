package example

import (
	"reflect"
	"testing"
	"time"
	"fmt"
	"encoding/json"
)

// Test data builders for complex types
func BuildTestUser() User {
	return User{
		ID:       12345,
		Username: "testuser",
		Email:    "test@example.com",
		Active:   true,
	}
}

func BuildTestAPIError() APIError {
	return APIError{
		Code:    404,
		Message: "User not found",
		Details: "The requested user does not exist in the system",
	}
}

func BuildComplexFetchResult() FetchResult {
	user := BuildTestUser()
	return &FetchSuccess{
		Value: &SomeUser{Value: user},
	}
}

func BuildErrorFetchResult() FetchResult {
	apiErr := BuildTestAPIError()
	return &FetchError{
		Error: apiErr,
	}
}

func BuildComplexRequestLog() RequestLog {
	return RequestLog{
		RequestID: "req-123-456-789",
		Timestamp: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		Result:    BuildComplexFetchResult(),
		Metadata: map[string]string{
			"source":    "api-gateway",
			"version":   "v1.2.3",
			"trace_id":  "trace-abc-def",
		},
	}
}

func BuildComplexUserSearchResult() UserSearchResult {
	users := []User{
		BuildTestUser(),
		{ID: 67890, Username: "user2", Email: "user2@example.com", Active: false},
	}
	
	pagedResult := &PagedUserSuccess{
		Data:       users,
		TotalCount: 2,
		Page:       1,
		PageSize:   10,
	}
	
	return &SearchSuccess{
		Value: pagedResult,
	}
}

func BuildComplexSearchResponse() SearchResponse {
	return SearchResponse{
		Query:      "username:test*",
		Results:    BuildComplexUserSearchResult(),
		ExecutedAt: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
	}
}

func BuildComplexNestedResult() NestedResult {
	return &NestedSuccess{
		Primary:   BuildComplexFetchResult(),
		Secondary: BuildComplexUserSearchResult(),
		Metadata: map[string]interface{}{
			"operation_type": "user_lookup_with_search",
			"priority":       "high",
			"retry_count":    0,
		},
	}
}

func BuildComplexOperation() ComplexOperation {
	endTime := time.Date(2024, 1, 15, 10, 35, 20, 0, time.UTC)
	return ComplexOperation{
		OperationID: "op-789-abc-def",
		StartTime:   time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		EndTime:     &endTime,
		Result:      BuildComplexNestedResult(),
		Metrics: map[string]float64{
			"duration_ms":     4750.5,
			"cpu_usage_pct":   23.4,
			"memory_mb":       128.7,
			"cache_hit_rate":  0.85,
		},
	}
}

// Invariance testing helper
func AssertInvariance[T any](t *testing.T, original, transformed T, description string) {
	t.Helper()
	if !reflect.DeepEqual(original, transformed) {
		t.Errorf("Invariance violation in %s: original and transformed objects differ", description)
		t.Logf("Original: %+v", original)
		t.Logf("Transformed: %+v", transformed)
	}
}

// Test: Complex type serialization chains with invariance testing
func TestComplexSerializationChains(t *testing.T) {
	// Create complex test data
	originalRequest := BuildComplexRequestLog()
	originalSearch := BuildComplexSearchResponse()
	originalOperation := BuildComplexOperation()
	
	t.Run("FetchResult_JSON_Chain", func(t *testing.T) {
		// go -> json -> go
		jsonBytes, err := json.Marshal(originalRequest.Result)
		if err != nil {
			t.Fatalf("Failed to marshal to JSON: %v", err)
		}
		
		var restored FetchResult
		err = json.Unmarshal(jsonBytes, &restored)
		if err != nil {
			t.Fatalf("Failed to unmarshal from JSON: %v", err)
		}
		
		AssertInvariance(t, originalRequest.Result, restored, "FetchResult JSON chain")
	})
	
	t.Run("RequestLog_Complete_JSON_Chain", func(t *testing.T) {
		// Full RequestLog: go -> json -> go
		jsonBytes, err := json.Marshal(originalRequest)
		if err != nil {
			t.Fatalf("Failed to marshal RequestLog to JSON: %v", err)
		}
		
		var restoredRequest RequestLog
		err = json.Unmarshal(jsonBytes, &restoredRequest)
		if err != nil {
			t.Fatalf("Failed to unmarshal RequestLog from JSON: %v", err)
		}
		
		AssertInvariance(t, originalRequest, restoredRequest, "RequestLog JSON chain")
	})
	
	t.Run("UserSearchResult_JSON_Chain", func(t *testing.T) {
		// go -> json -> go
		jsonBytes, err := json.Marshal(originalSearch.Results)
		if err != nil {
			t.Fatalf("Failed to marshal UserSearchResult to JSON: %v", err)
		}
		
		var restored UserSearchResult
		err = json.Unmarshal(jsonBytes, &restored)
		if err != nil {
			t.Fatalf("Failed to unmarshal UserSearchResult from JSON: %v", err)
		}
		
		AssertInvariance(t, originalSearch.Results, restored, "UserSearchResult JSON chain")
	})
	
	t.Run("SearchResponse_Complete_JSON_Chain", func(t *testing.T) {
		// Full SearchResponse: go -> json -> go
		jsonBytes, err := json.Marshal(originalSearch)
		if err != nil {
			t.Fatalf("Failed to marshal SearchResponse to JSON: %v", err)
		}
		
		var restoredSearch SearchResponse
		err = json.Unmarshal(jsonBytes, &restoredSearch)
		if err != nil {
			t.Fatalf("Failed to unmarshal SearchResponse from JSON: %v", err)
		}
		
		AssertInvariance(t, originalSearch, restoredSearch, "SearchResponse JSON chain")
	})
	
	t.Run("ComplexOperation_Full_Chain", func(t *testing.T) {
		// Most complex nested structure: go -> json -> go
		jsonBytes, err := json.Marshal(originalOperation)
		if err != nil {
			t.Fatalf("Failed to marshal ComplexOperation to JSON: %v", err)
		}
		
		var restoredOperation ComplexOperation
		err = json.Unmarshal(jsonBytes, &restoredOperation)
		if err != nil {
			t.Fatalf("Failed to unmarshal ComplexOperation from JSON: %v", err)
		}
		
		AssertInvariance(t, originalOperation, restoredOperation, "ComplexOperation JSON chain")
	})
}

// Demonstration of format-specific serialization chains
func TestProtobufSerializationChains(t *testing.T) {
	t.Skip("Protobuf chains require generated *_protobuf_gen.go files")
	
	// This test would demonstrate:
	// go -> protobuf -> go chains
	// once the protobuf generators are run
}

func TestSQLSerializationChains(t *testing.T) {
	t.Skip("SQL chains require generated *_sql_gen.go files")
	
	// This test would demonstrate:
	// go -> sql -> go chains  
	// once the SQL generators are run
}

func TestGraphQLSerializationChains(t *testing.T) {
	t.Skip("GraphQL chains require generated *_graphql_gen.go files")
	
	// This test would demonstrate:
	// go -> graphql -> go chains
	// once the GraphQL generators are run
}

// Cross-format transformation chains
func TestCrossFormatChains(t *testing.T) {
	t.Skip("Cross-format chains require all generated files")
	
	// This test would demonstrate:
	// json -> protobuf -> sql -> go
	// protobuf -> json -> graphql -> go
	// etc.
}

// Benchmark complex serialization performance
func BenchmarkComplexSerialization(b *testing.B) {
	request := BuildComplexRequestLog()
	operation := BuildComplexOperation()
	
	b.Run("JSON_RequestLog", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data, _ := json.Marshal(request)
			var restored RequestLog
			json.Unmarshal(data, &restored)
		}
	})
	
	b.Run("JSON_ComplexOperation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data, _ := json.Marshal(operation)
			var restored ComplexOperation
			json.Unmarshal(data, &restored)
		}
	})
	
	// Additional benchmarks would be added for protobuf, sql, graphql
	// once the generators are run
}

// Demonstration function that shows the types in action
func ExampleComplexTypes() {
	// Build test data
	user := BuildTestUser()
	fmt.Printf("User: %+v\n", user)
	
	// Show OptionUser usage
	someUser := &SomeUser{Value: user}
	fmt.Printf("SomeUser: %+v\n", someUser)
	
	noneUser := &NoneUser{}
	fmt.Printf("NoneUser: %+v\n", noneUser)
	
	// Show FetchResult usage (equivalent to Result[Option[User], APIError])
	successResult := &FetchSuccess{Value: someUser}
	fmt.Printf("Success FetchResult: %+v\n", successResult)
	
	errorResult := &FetchError{Error: BuildTestAPIError()}
	fmt.Printf("Error FetchResult: %+v\n", errorResult)
	
	// Show full RequestLog with nested types
	requestLog := BuildComplexRequestLog()
	fmt.Printf("Request Log: %+v\n", requestLog)
	
	// Show even more complex nested types
	searchResponse := BuildComplexSearchResponse()
	fmt.Printf("Search Response: %+v\n", searchResponse)
	
	// Show most complex nested operation
	complexOp := BuildComplexOperation()
	fmt.Printf("Complex Operation: %+v\n", complexOp)
	
	// Output demonstrates the complex type hierarchy working
}