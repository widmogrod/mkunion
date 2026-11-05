package testasset

import (
	"time"

	"github.com/widmogrod/mkunion/f"
)

// Query and Answer are a request and response pair in the shape of an HTTP
// API: every request embeds f.Returns to name its answer, fields carry json
// tags, one answer holds the Status union and a time, one field is hidden.
// The Swift exporter's golden test renders this file.

//go:tag mkunion:"Query"
type (
	Ping struct{ f.Returns[*Pong] }
	Find struct {
		f.Returns[*Found]
		Text  string   `json:"text"`
		Limit int      `json:"limit,omitempty"`
		Tags  []string `json:"tags,omitempty"`
	}
	Open struct {
		f.Returns[*Document]
		URL string `json:"url"`
	}
)

//go:tag mkunion:"Answer"
type (
	Pong struct {
		Version string `json:"version"`
	}
	Found struct {
		Hits []Hit   `json:"hits"`
		Took float64 `json:"took"`
	}
	Document struct {
		Name    string         `json:"name"`
		Cover   *string        `json:"cover,omitempty"`
		Pages   map[string]int `json:"pages"`
		Raw     []byte         `json:"raw,omitempty"`
		Updated time.Time      `json:"updated"`
		Status  Status         `json:"status"`
		Secret  string         `json:"-"`
	}
	Failure struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
)

//go:tag mkunion:"Status"
type (
	Draft     struct{}
	Published struct {
		At time.Time `json:"at"`
	}
	Label string
)

// Hit is a plain struct an answer carries.
type Hit struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}
