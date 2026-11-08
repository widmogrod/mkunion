// Package api is the contract between manga-cli serve and the Arcs app.
//
// One request type, one answer type, no routes: a client posts one Request
// value to POST /v1 and gets one Response value back, both in mkunion's JSON
// form {"$type": "api.Search", "api.Search": {...}}. Each request names the
// answer it expects by embedding f.Returns[*Answer]; mkunion turns that into
// the typed RequestHandler the server implements, and the Swift and
// TypeScript clients learn the pairing from the same marker.
//
// Field names on the wire follow the json tags. Every object may gain
// fields; clients ignore unknown keys. Lists live in a struct, never at the
// top level, because a bare list loses its $type.
package api

import (
	"time"

	"github.com/widmogrod/mkunion/f"
)

//go:generate go tool mkunion -i api.go
// The app's ArcsAPI module (reader/ArcsCore/Sources/ArcsAPI) is generated from this file.
// (disabled for the rpc experiment, writes outside the folder) //go:generate go tool mkunion shape-export --language swift -o ../../../reader/ArcsCore/Sources/ArcsAPI api.go

// Request is everything the server can be asked. Info.Accepts lists the
// variants a given server handles, so a client shows only what works.
// answers=Response makes mkunion check that every Request answers with a
// Response variant, and generate DispatchRequest and NewRequestClient from it.
//
//go:tag mkunion:"Request,answers=Response"
type (
	// GetInfo asks who the server is and what it accepts.
	GetInfo struct{ f.Returns[*Info] }
	// Search asks every source for series matching Query. Source, Genres,
	// Status and Limit narrow it; they are passed through from day one even
	// though the app has no controls for them yet.
	Search struct {
		f.Returns[*SearchResults]
		Query  string   `json:"query"`
		Source string   `json:"source,omitempty"`
		Genres []string `json:"genres,omitempty"`
		Status string   `json:"status,omitempty"`
		Limit  int      `json:"limit,omitempty"`
	}
	// GetSeries asks for one series page: name, cover, chapters.
	GetSeries struct {
		f.Returns[*Series]
		URL string `json:"url"`
	}
	// AddToLibrary creates the series folder in the library and starts a
	// sync job that downloads its chapters. A series already in the library
	// gets a sync job and nothing else.
	AddToLibrary struct {
		f.Returns[*JobAccepted]
		URL string `json:"url"`
		// Order is the download order: "oldest" (chapter 1 first), "newest"
		// (the latest first, then the rest) or "" for the site's order. A
		// new folder keeps it in its .manga.yaml, as `init --order` does; a
		// folder that exists keeps its own and uses this one for this sync.
		Order string `json:"order,omitempty"`
	}
	// ListJobs asks for every job the server remembers, newest first.
	ListJobs struct{ f.Returns[*Jobs] }
	// CancelJob stops a running job; the answer carries the job as it is now.
	CancelJob struct {
		f.Returns[*JobAccepted]
		ID string `json:"id"`
	}
)

// Response is everything the server can answer. Error is the answer to any
// request that could not be served for an application reason (unknown
// source, site down); a body that does not decode is HTTP 400 instead.
//
//go:tag mkunion:"Response"
type (
	Info struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		// Library is the folder the server serves, for display.
		Library string `json:"library"`
		// Accepts lists the Request variants this server handles, as
		// "api.Search". It is the capabilities list.
		Accepts []string `json:"accepts"`
	}
	SearchResults struct {
		Results []SearchHit   `json:"results"`
		Errors  []SourceError `json:"errors"`
	}
	Series struct {
		Name     string    `json:"name"`
		URL      string    `json:"url"`
		Source   string    `json:"source"`
		Complete bool      `json:"complete"`
		Cover    string    `json:"cover,omitempty"`
		Chapters []Chapter `json:"chapters"`
		// InLibrary is the folder name when the series is already in the
		// library, else null.
		InLibrary *string `json:"inLibrary"`
	}
	JobAccepted struct {
		Job Job `json:"job"`
	}
	Jobs struct {
		Items []Job `json:"items"`
	}
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
)

// Error is also a Go error, so a client that gets one in place of the answer
// it asked for can errors.As to it (see RequestUnexpectedAnswer.Unwrap).
func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Error codes. A client switches on Code and shows Message.
const (
	ErrUnknownRequest = "unknown_request"
	ErrBadRequest     = "bad_request"
	ErrUnknownSource  = "unknown_source"
	ErrSeriesGone     = "series_gone"
	ErrInternal       = "internal"
)

// SearchHit is one series found by one source.
type SearchHit struct {
	Source string   `json:"source"`
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Author string   `json:"author,omitempty"`
	Genres []string `json:"genres,omitempty"`
	// Status is "ongoing", "completed" or "unknown".
	Status string `json:"status"`
	// Cover is the picture's URL on the source's site; fetch it through
	// GET /v1/image?url=<cover>&series=<url> because the sites want their
	// own referer.
	Cover     string  `json:"cover,omitempty"`
	InLibrary *string `json:"inLibrary"`
}

// SourceError is one source that failed during a search; the others still
// answered.
type SourceError struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

// Chapter is one chapter row on a series page.
type Chapter struct {
	Name   string  `json:"name"`
	URL    string  `json:"url"`
	Number float64 `json:"number"`
}

// Job is one background sync the server runs or ran.
type Job struct {
	ID string `json:"id"`
	// Key is the series folder name in the library.
	Key   string   `json:"key"`
	Name  string   `json:"name"`
	URL   string   `json:"url"`
	State JobState `json:"state"`
	// ChaptersDone and ChaptersTotal come from the folder, the way `status`
	// reads it, so a job and a terminal sync report the same numbers.
	ChaptersDone  int       `json:"chaptersDone"`
	ChaptersTotal int       `json:"chaptersTotal"`
	Started       time.Time `json:"started"`
	// Cancelled is set when CancelJob stopped the job.
	Cancelled bool `json:"cancelled"`
}

// JobState is where a job is. Failed carries the reason; a Done job may
// still have chapters the site never served (see the folder's state).
//
//go:tag mkunion:"JobState"
type (
	Queued  struct{}
	Running struct{}
	Done    struct{}
	Failed  struct {
		Reason string `json:"reason"`
	}
)
