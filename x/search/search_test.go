package search

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"testing"
)

var corpus = map[string]string{
	"go":    "Recent changes to the Go programming language have made it possible to eliminate all garbage collection",
	"apple": "Apple has released a new version of its Swift programming language that is designed to replace the Objective-C programming language.",
	"math":  "Linear equation y= mx + b is a linear equation where m is the slope of the line and b is the y-intercept.",
	"noice": "Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
}

func TestSearch_Search(t *testing.T) {
	index := NewIndex(NewSimpleTokenizer())
	for key, text := range corpus {
		err := index.Add(key, text)
		assert.NoError(t, err)
	}

	t.Run("term search should return documents that have term", func(t *testing.T) {
		result, err := index.Search(&Term{
			Term: "programming",
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)

		expectedOrder := &SearchResult{
			Results: []Hit{
				{
					DocID: "apple",
					Score: 0.06931471805599453,
					Raw:   corpus["apple"],
				},
				{
					DocID: "go",
					Score: 0.04332169878499658,
					Raw:   corpus["go"],
				},
			},
		}

		if diff := cmp.Diff(expectedOrder, result); diff != "" {
			t.Errorf("Search() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("fulltext search should return documents that match most terms", func(t *testing.T) {
		result, err := index.Search(&Fulltext{
			Query: "Go programming language",
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)

		expectedOrder := &SearchResult{
			Results: []Hit{
				{
					DocID: "go",
					Score: 0.028881132523331052,
					Raw:   corpus["go"],
				},
				{
					DocID: "apple",
					Score: 0.01732867951399863,
					Raw:   corpus["apple"],
				},
			},
		}

		if diff := cmp.Diff(expectedOrder, result); diff != "" {
			t.Errorf("Search() mismatch (-want +got):\n%s", diff)
		}
	})
}
