package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shared"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/workflow"
)

func newTestServer(t *testing.T, oaic *openai.Client) (*app, *echo.Echo) {
	t.Helper()
	a := newApp(oaic)
	e := echo.New()
	a.registerRoutes(e)
	return a, e
}

func doRequest(e *echo.Echo, method, path string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func helloFlow() *workflow.Flow {
	return &workflow.Flow{
		Name: "hello_world_flow",
		Arg:  "input",
		Body: []workflow.Expr{
			&workflow.Assign{
				ID:    "assign1",
				VarOk: "res",
				Val: &workflow.Apply{ID: "apply1", Name: "concat", Args: []workflow.Reshaper{
					&workflow.SetValue{Value: schema.MkString("hello ")},
					&workflow.GetValue{Path: "input"},
				}},
			},
			&workflow.End{
				ID:     "end1",
				Result: &workflow.GetValue{Path: "res"},
			},
		},
	}
}

func saveHelloFlow(t *testing.T, e *echo.Echo) {
	t.Helper()
	body, err := shared.JSONMarshal[workflow.Workflow](helloFlow())
	require.NoError(t, err)

	rec := doRequest(e, http.MethodPost, "/flow", body)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestFlowLifecycle(t *testing.T) {
	_, e := newTestServer(t, nil)

	saveHelloFlow(t, e)

	t.Run("saved flow can be fetched", func(t *testing.T) {
		rec := doRequest(e, http.MethodGet, "/flow/hello_world_flow", nil)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "hello_world_flow")
	})

	t.Run("unknown flow is a 404, not a 500", func(t *testing.T) {
		rec := doRequest(e, http.MethodGet, "/flow/i-do-not-exist", nil)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("non-flow workflow payload is rejected", func(t *testing.T) {
		// a FlowRef is a valid workflow.Workflow, but /flow stores only *Flow
		body, err := shared.JSONMarshal[workflow.Workflow](&workflow.FlowRef{FlowID: "x"})
		require.NoError(t, err)

		rec := doRequest(e, http.MethodPost, "/flow", body)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("garbage body is rejected", func(t *testing.T) {
		rec := doRequest(e, http.MethodPost, "/flow", []byte(`{"nope`))
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestFlowsList(t *testing.T) {
	_, e := newTestServer(t, nil)
	saveHelloFlow(t, e)

	t.Run("lists saved flows", func(t *testing.T) {
		rec := doRequest(e, http.MethodPost, "/flows", []byte(`{}`))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "hello_world_flow")
	})

	t.Run("unparsable query falls back to listing everything", func(t *testing.T) {
		rec := doRequest(e, http.MethodPost, "/flows", []byte(`not-json`))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "hello_world_flow")
	})
}

func TestRunCommandAndStates(t *testing.T) {
	a, e := newTestServer(t, nil)
	saveHelloFlow(t, e)

	cmd, err := shared.JSONMarshal[workflow.Command](&workflow.Run{
		Flow:  &workflow.FlowRef{FlowID: "hello_world_flow"},
		Input: schema.MkString("world"),
	})
	require.NoError(t, err)

	rec := doRequest(e, http.MethodPost, "/", cmd)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "workflow.Done")
	assert.Contains(t, rec.Body.String(), "hello world")

	t.Run("state is listed", func(t *testing.T) {
		rec := doRequest(e, http.MethodPost, "/states", []byte(`{}`))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "workflow.Done")
	})

	t.Run("run renders back to workflow syntax", func(t *testing.T) {
		records, err := a.statesRepo.FindingRecords(findAllStates())
		require.NoError(t, err)
		require.NotEmpty(t, records.Items)

		runID := records.Items[0].ID
		rec := doRequest(e, http.MethodGet, "/workflow-to-str-from-run/"+runID, nil)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "flow hello_world_flow")
	})

	t.Run("unknown run id errors", func(t *testing.T) {
		rec := doRequest(e, http.MethodGet, "/workflow-to-str-from-run/i-do-not-exist", nil)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("run of an unknown flow errors", func(t *testing.T) {
		cmd, err := shared.JSONMarshal[workflow.Command](&workflow.Run{
			Flow:  &workflow.FlowRef{FlowID: "i-do-not-exist"},
			Input: schema.MkString("world"),
		})
		require.NoError(t, err)

		rec := doRequest(e, http.MethodPost, "/", cmd)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func findAllStates() schemaless.FindingRecords[schemaless.Record[workflow.State]] {
	return schemaless.FindingRecords[schemaless.Record[workflow.State]]{RecordType: "process"}
}

func TestCallbackEdgeCases(t *testing.T) {
	_, e := newTestServer(t, nil)

	t.Run("non-callback command is rejected", func(t *testing.T) {
		cmd, err := shared.JSONMarshal[workflow.Command](&workflow.Run{
			Flow:  &workflow.FlowRef{FlowID: "hello_world_flow"},
			Input: schema.MkString("world"),
		})
		require.NoError(t, err)

		rec := doRequest(e, http.MethodPost, "/callback", cmd)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("unknown callback id is rejected", func(t *testing.T) {
		cmd, err := shared.JSONMarshal[workflow.Command](&workflow.Callback{
			CallbackID: "callback_that_does_not_exist",
			Result:     schema.MkString("data"),
		})
		require.NoError(t, err)

		rec := doRequest(e, http.MethodPost, "/callback", cmd)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestFuncEndpoint(t *testing.T) {
	_, e := newTestServer(t, nil)

	input := func(t *testing.T, name string, args ...schema.Schema) []byte {
		t.Helper()
		body, err := shared.JSONMarshal[*workflow.FunctionInput](&workflow.FunctionInput{
			Name: name,
			Args: args,
		})
		require.NoError(t, err)
		return body
	}

	t.Run("concat concatenates", func(t *testing.T) {
		rec := doRequest(e, http.MethodPost, "/func",
			input(t, "concat", schema.MkString("a"), schema.MkString("b")))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "ab")
	})

	t.Run("concat with non-string args errors", func(t *testing.T) {
		rec := doRequest(e, http.MethodPost, "/func",
			input(t, "concat", schema.MkMap(), schema.MkMap()))
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("unknown function errors", func(t *testing.T) {
		rec := doRequest(e, http.MethodPost, "/func", input(t, "i-do-not-exist"))
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("garbage body errors", func(t *testing.T) {
		rec := doRequest(e, http.MethodPost, "/func", []byte(`{"nope`))
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestWorkflowToStr(t *testing.T) {
	_, e := newTestServer(t, nil)

	t.Run("valid workflow renders", func(t *testing.T) {
		body, err := shared.JSONMarshal[workflow.Workflow](helloFlow())
		require.NoError(t, err)

		rec := doRequest(e, http.MethodPost, "/workflow-to-str", body)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "flow hello_world_flow")
	})

	t.Run("garbage errors", func(t *testing.T) {
		rec := doRequest(e, http.MethodPost, "/workflow-to-str", []byte(`garbage`))
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestUpdatingEndpointsRejectGarbage(t *testing.T) {
	_, e := newTestServer(t, nil)

	rec := doRequest(e, http.MethodPost, "/flows-updating", []byte(`{"nope`))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	rec = doRequest(e, http.MethodPost, "/state-updating", []byte(`{"nope`))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// fakeOpenAI serves canned chat-completion responses so /message can be
// tested without the real API.
func fakeOpenAI(t *testing.T, responses ...openai.ChatCompletionResponse) (*openai.Client, *int) {
	t.Helper()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls >= len(responses) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": {"message": "no more canned responses"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(responses[calls])
		calls++
	}))
	t.Cleanup(srv.Close)

	config := openai.DefaultConfig("test-key")
	config.BaseURL = srv.URL + "/v1"
	return openai.NewClientWithConfig(config), &calls
}

func chatMessage(t *testing.T, e *echo.Echo, message string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := shared.JSONMarshal[ChatCMD](&UserMessage{Message: message})
	require.NoError(t, err)
	return doRequest(e, http.MethodPost, "/message", body)
}

func TestMessageEndpoint(t *testing.T) {
	t.Run("plain answer produces one response", func(t *testing.T) {
		client, calls := fakeOpenAI(t, openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "hello there",
				},
			}},
		})
		_, e := newTestServer(t, client)

		rec := chatMessage(t, e, "hi")
		assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Body.String(), "hello there")
		assert.Equal(t, 1, *calls, "no tool calls means a single completion request")
	})

	t.Run("tool call triggers a second completion", func(t *testing.T) {
		client, calls := fakeOpenAI(t,
			openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{{
					Message: openai.ChatCompletionMessage{
						Role: openai.ChatMessageRoleAssistant,
						ToolCalls: []openai.ToolCall{
							{
								ID:   "call-1",
								Type: openai.ToolTypeFunction,
								Function: openai.FunctionCall{
									Name:      "refresh_states",
									Arguments: "{}",
								},
							},
							{
								ID:   "call-2",
								Type: openai.ToolTypeFunction,
								Function: openai.FunctionCall{
									Name:      "generate_image",
									Arguments: "{}",
								},
							},
						},
					},
				}},
			},
			openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{{
					Message: openai.ChatCompletionMessage{
						Role:    openai.ChatMessageRoleAssistant,
						Content: "states refreshed",
					},
				}},
			},
		)
		_, e := newTestServer(t, client)

		rec := chatMessage(t, e, "refresh my states")
		assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Body.String(), "states refreshed")
		assert.Equal(t, 2, *calls, "a tool call forces a follow-up completion")
	})

	t.Run("upstream error propagates", func(t *testing.T) {
		client, _ := fakeOpenAI(t) // no canned responses: every call fails
		_, e := newTestServer(t, client)

		rec := chatMessage(t, e, "hi")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("second completion error propagates", func(t *testing.T) {
		client, _ := fakeOpenAI(t, openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{{
				Message: openai.ChatCompletionMessage{
					Role: openai.ChatMessageRoleAssistant,
					ToolCalls: []openai.ToolCall{{
						ID:   "call-1",
						Type: openai.ToolTypeFunction,
						Function: openai.FunctionCall{
							Name:      "refresh_states",
							Arguments: "{}",
						},
					}},
				},
			}},
		}) // second call has no canned response and fails
		_, e := newTestServer(t, client)

		rec := chatMessage(t, e, "refresh my states")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestCommandToWhere(t *testing.T) {
	t.Run("commands addressing existing runs produce a query", func(t *testing.T) {
		for _, cmd := range []workflow.Command{
			&workflow.StopSchedule{ParentRunID: "p1"},
			&workflow.ResumeSchedule{ParentRunID: "p1"},
			&workflow.TryRecover{RunID: "r1"},
		} {
			where, ok := commandToWhere(cmd)
			assert.True(t, ok, "%T", cmd)
			assert.NotNil(t, where, "%T", cmd)
		}
	})
	t.Run("run command creates a new record", func(t *testing.T) {
		where, ok := commandToWhere(&workflow.Run{})
		assert.False(t, ok)
		assert.Nil(t, where)
	})
}
