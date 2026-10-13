package relayconvert

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponsesToolStreamPreservesIdentityAndArguments(t *testing.T) {
	const arguments = `{"limit":2000,"path":"template/reusable-skill/SKILL.md"}`
	tests := []struct {
		name             string
		initialItemOnly  bool
		terminalCallOnly bool
		terminalNewItem  bool
		terminalString   bool
		pendingOnly      bool
		terminalOnly     bool
		partial          bool
		doneArguments    bool
		multiple         bool
		terminalType     string
	}{
		{name: "completed snapshot after all deltas"},
		{name: "done snapshot after all deltas", terminalType: responsesEventDone},
		{name: "incomplete snapshot after all deltas", terminalType: responsesEventIncomplete},
		{name: "item identity before output index", initialItemOnly: true},
		{name: "terminal identity only has call id", terminalCallOnly: true},
		{name: "same call id with a different terminal item id", terminalNewItem: true},
		{name: "string encoded terminal arguments", terminalString: true},
		{name: "pending deltas supplemented by terminal snapshot", pendingOnly: true},
		{name: "terminal snapshot without earlier events", terminalOnly: true},
		{name: "terminal snapshot supplies missing suffix", partial: true},
		{name: "arguments done supplies missing suffix", partial: true, doneArguments: true},
		{name: "different calls with identical names and arguments", multiple: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := NewResponsesToChatStreamState("gpt-5.6-luna", true)
			state.ID = "chatcmpl_test"
			state.Created = 123
			var events []dto.ResponsesStreamResponse
			// Reasoning occupies output index 0, but must not occupy a Chat tool index.
			output := []dto.ResponsesOutput{{Type: responsesOutputTypeReasoning}}
			wantCalls := 1
			if tt.multiple {
				wantCalls = 2
			}
			for i := 0; i < wantCalls; i++ {
				itemID := []string{"fc_1", "fc_2"}[i]
				callID := []string{"call_1", "call_2"}[i]
				index := i + 1
				item := &dto.ResponsesOutput{Type: responsesOutputTypeFunctionCall, ID: itemID, CallId: callID, Name: "read"}
				if !tt.terminalOnly && !tt.pendingOnly {
					added := dto.ResponsesStreamResponse{Type: responsesEventOutputItemAdded, OutputIndex: &index, Item: item}
					if tt.initialItemOnly {
						added.OutputIndex = nil
					}
					events = append(events, added)
				}
				if !tt.terminalOnly {
					events = append(events, dto.ResponsesStreamResponse{
						Type: responsesEventFunctionArgsDelta, OutputIndex: &index, ItemID: itemID, Delta: arguments[:18],
					})
					if !tt.partial {
						events = append(events, dto.ResponsesStreamResponse{
							Type: responsesEventFunctionArgsDelta, OutputIndex: &index, ItemID: itemID, Delta: arguments[18:],
						})
					}
				}
				if tt.doneArguments {
					events = append(events, dto.ResponsesStreamResponse{
						Type: responsesEventFunctionArgsDone, OutputIndex: &index, ItemID: itemID, Arguments: []byte(arguments),
					})
				}
				finalItem := *item
				finalItem.Arguments = []byte(arguments)
				if tt.terminalString {
					var err error
					finalItem.Arguments, err = common.Marshal(arguments)
					require.NoError(t, err)
				}
				if tt.terminalCallOnly {
					finalItem.ID = ""
				}
				if tt.terminalNewItem {
					finalItem.ID = "fc_final"
				}
				output = append(output, finalItem)
			}
			terminalType := tt.terminalType
			if terminalType == "" {
				terminalType = responsesEventCompleted
			}
			response := &dto.OpenAIResponsesResponse{
				Output: output, Usage: &dto.Usage{InputTokens: 125, OutputTokens: 30, TotalTokens: 155},
			}
			if terminalType != responsesEventIncomplete {
				response.Status = []byte(`"completed"`)
			}
			events = append(events, dto.ResponsesStreamResponse{Type: terminalType, Response: response})
			assembled := make(map[int]dto.ToolCallResponse)
			finishCount, usageCount := 0, 0
			for _, event := range events {
				chunks, err := ResponsesStreamEventToChatChunks(&event, state)
				require.NoError(t, err)
				for _, chunk := range chunks {
					assert.Equal(t, "chatcmpl_test", chunk.Id)
					if chunk.Usage != nil {
						usageCount++
						assert.Equal(t, 155, chunk.Usage.TotalTokens)
					}
					for _, choice := range chunk.Choices {
						if choice.FinishReason != nil {
							finishCount++
							wantReason := "tool_calls"
							if terminalType == responsesEventIncomplete {
								wantReason = "length"
							}
							assert.Equal(t, wantReason, *choice.FinishReason)
						}
						for _, call := range choice.Delta.ToolCalls {
							require.NotNil(t, call.Index)
							previous := assembled[*call.Index]
							if previous.ID != "" {
								assert.Equal(t, previous.ID, call.ID)
							}
							previous.ID = call.ID
							previous.Function.Name += call.Function.Name
							previous.Function.Arguments += call.Function.Arguments
							assembled[*call.Index] = previous
						}
					}
				}
			}
			require.Len(t, assembled, wantCalls)
			for i := 0; i < wantCalls; i++ {
				assert.Equal(t, []string{"call_1", "call_2"}[i], assembled[i].ID)
				assert.Equal(t, "read", assembled[i].Function.Name)
				assert.Equal(t, arguments, assembled[i].Function.Arguments)
				var parsed map[string]any
				require.NoError(t, common.UnmarshalJsonStr(assembled[i].Function.Arguments, &parsed))
				assert.Equal(t, "template/reusable-skill/SKILL.md", parsed["path"])
			}
			assert.Equal(t, 1, finishCount)
			assert.Equal(t, 1, usageCount)
			assert.Equal(t, wantCalls*(len("read")+len(arguments)), len(state.UsageText()))
			assert.Empty(t, FinalizeResponsesToChatStream(state))
		})
	}
}

func TestResponsesToolStreamSupplementsUnicodeArgumentsByBytes(t *testing.T) {
	const arguments = `{"path":"模板/技能.md"}`
	state := NewResponsesToChatStreamState("gpt-test", false)
	index := 0
	_, err := ResponsesStreamEventToChatChunks(&dto.ResponsesStreamResponse{
		Type: responsesEventOutputItemAdded, OutputIndex: &index, Item: &dto.ResponsesOutput{
			Type: responsesOutputTypeFunctionCall, ID: "fc_1", CallId: "call_1", Name: "read",
		},
	}, state)
	require.NoError(t, err)
	_, err = ResponsesStreamEventToChatChunks(&dto.ResponsesStreamResponse{
		Type: responsesEventFunctionArgsDelta, OutputIndex: &index, Delta: `{"path":"模板`,
	}, state)
	require.NoError(t, err)
	chunks, err := ResponsesStreamEventToChatChunks(&dto.ResponsesStreamResponse{
		Type: responsesEventFunctionArgsDone, OutputIndex: &index, Arguments: []byte(arguments),
	}, state)
	require.NoError(t, err)
	require.Len(t, chunks, 1)
	assert.Equal(t, `/技能.md"}`, chunks[0].Choices[0].Delta.ToolCalls[0].Function.Arguments)
}

func TestResponsesToolStreamRegistryDoesNotReplayTerminalSnapshot(t *testing.T) {
	state, err := NewResponseStreamState(types.RelayFormatOpenAIResponses, types.RelayFormatOpenAI, ResponseStreamOptions{
		ID: "chatcmpl_test", Model: "gpt-test", IncludeUsage: true,
	})
	require.NoError(t, err)
	index := 0
	item := dto.ResponsesOutput{Type: responsesOutputTypeFunctionCall, ID: "fc_1", CallId: "call_1", Name: "read"}
	_, err = ConvertStreamResponseChunk(nil, nil, state, &dto.ResponsesStreamResponse{
		Type: responsesEventOutputItemAdded, OutputIndex: &index, Item: &item,
	})
	require.NoError(t, err)
	_, err = ConvertStreamResponseChunk(nil, nil, state, &dto.ResponsesStreamResponse{
		Type: responsesEventFunctionArgsDelta, OutputIndex: &index, Delta: `{"path":"x"}`,
	})
	require.NoError(t, err)
	item.Arguments = []byte(`{"path":"x"}`)
	results, err := ConvertStreamResponseChunk(nil, nil, state, &dto.ResponsesStreamResponse{
		Type: responsesEventCompleted, Response: &dto.OpenAIResponsesResponse{
			Status: []byte(`"completed"`), Output: []dto.ResponsesOutput{item},
			Usage: &dto.Usage{InputTokens: 125, OutputTokens: 30, TotalTokens: 155},
		},
	})
	require.NoError(t, err)
	require.Len(t, results, 2, "only finish_reason and usage remain after all arguments have been sent")
	for _, result := range results {
		chunk, ok := result.Value.(dto.ChatCompletionsStreamResponse)
		require.True(t, ok)
		for _, choice := range chunk.Choices {
			assert.Empty(t, choice.Delta.ToolCalls)
		}
	}
	assert.Equal(t, 155, state.Usage().TotalTokens)
	results, err = FinalizeStreamResponse(nil, nil, state)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestResponsesToolStreamArgumentsDoneEmitsMissingSuffixImmediately(t *testing.T) {
	state := NewResponsesToChatStreamState("gpt-test", false)
	index := 0
	_, err := ResponsesStreamEventToChatChunks(&dto.ResponsesStreamResponse{
		Type: responsesEventOutputItemAdded, OutputIndex: &index,
		Item: &dto.ResponsesOutput{Type: responsesOutputTypeFunctionCall, ID: "fc_1", CallId: "call_1", Name: "read"},
	}, state)
	require.NoError(t, err)
	_, err = ResponsesStreamEventToChatChunks(&dto.ResponsesStreamResponse{
		Type: responsesEventFunctionArgsDelta, OutputIndex: &index, Delta: `{"path":`,
	}, state)
	require.NoError(t, err)
	chunks, err := ResponsesStreamEventToChatChunks(&dto.ResponsesStreamResponse{
		Type: responsesEventFunctionArgsDone, ItemID: "fc_1", Arguments: []byte(`{"path":"x"}`),
	}, state)
	require.NoError(t, err)
	require.Len(t, chunks, 1)
	require.Len(t, chunks[0].Choices[0].Delta.ToolCalls, 1)
	assert.Equal(t, `"x"}`, chunks[0].Choices[0].Delta.ToolCalls[0].Function.Arguments)
}

func TestResponsesToolStreamDoesNotEmitAfterTerminal(t *testing.T) {
	state := NewResponsesToChatStreamState("gpt-test", false)
	_, err := ResponsesStreamEventToChatChunks(&dto.ResponsesStreamResponse{
		Type: responsesEventCompleted, Response: &dto.OpenAIResponsesResponse{Status: []byte(`"completed"`)},
	}, state)
	require.NoError(t, err)
	for _, event := range []dto.ResponsesStreamResponse{
		{Type: responsesEventFunctionArgsDelta, OutputIndex: common.GetPointer(0), Delta: `{}`},
		{Type: responsesEventDone, Response: &dto.OpenAIResponsesResponse{Output: []dto.ResponsesOutput{
			{Type: responsesOutputTypeFunctionCall, ID: "fc_late", CallId: "call_late", Name: "read", Arguments: []byte(`{}`)},
		}}},
	} {
		chunks, err := ResponsesStreamEventToChatChunks(&event, state)
		require.NoError(t, err)
		assert.Empty(t, chunks)
	}
	assert.Empty(t, FinalizeResponsesToChatStream(state))
}

func TestResponsesToolStreamBindsNewAliasesOnArgumentEvents(t *testing.T) {
	state := NewResponsesToChatStreamState("gpt-test", false)
	index := 1
	events := []dto.ResponsesStreamResponse{
		{Type: responsesEventOutputItemAdded, Item: &dto.ResponsesOutput{
			Type: responsesOutputTypeFunctionCall, ID: "fc_1", CallId: "call_1", Name: "read",
		}},
		{Type: responsesEventFunctionArgsDelta, OutputIndex: &index, ItemID: "fc_1", Delta: `{"path":`},
		{Type: responsesEventFunctionArgsDelta, OutputIndex: &index, Delta: `"x"}`},
		{Type: responsesEventOutputItemDone, OutputIndex: &index, Item: &dto.ResponsesOutput{
			Type: responsesOutputTypeFunctionCall, ID: "fc_1", CallId: "call_1", Name: "read", Arguments: []byte(`{"path":"x"}`),
		}},
	}
	arguments := ""
	for _, event := range events {
		chunks, err := ResponsesStreamEventToChatChunks(&event, state)
		require.NoError(t, err)
		for _, chunk := range chunks {
			for _, choice := range chunk.Choices {
				for _, call := range choice.Delta.ToolCalls {
					require.NotNil(t, call.Index)
					assert.Equal(t, 0, *call.Index)
					assert.Equal(t, "call_1", call.ID)
					arguments += call.Function.Arguments
				}
			}
		}
	}
	assert.Equal(t, `{"path":"x"}`, arguments)
}

func TestResponsesToolStreamDefersUnknownCallIdentityUntilItemArrives(t *testing.T) {
	state := NewResponsesToChatStreamState("gpt-test", false)
	index := 0
	for _, event := range []dto.ResponsesStreamResponse{
		{Type: responsesEventFunctionArgsDelta, OutputIndex: &index, ItemID: "fc_1", Delta: `{"path":"x"}`},
		{Type: responsesEventFunctionArgsDone, OutputIndex: &index, ItemID: "fc_1", Arguments: []byte(`{"path":"x"}`)},
	} {
		chunks, err := ResponsesStreamEventToChatChunks(&event, state)
		require.NoError(t, err)
		assert.Empty(t, chunks, "wait for the actual call ID instead of emitting a temporary ID")
	}
	chunks, err := ResponsesStreamEventToChatChunks(&dto.ResponsesStreamResponse{
		Type: responsesEventOutputItemDone, OutputIndex: &index, Item: &dto.ResponsesOutput{
			Type: responsesOutputTypeFunctionCall, ID: "fc_1", CallId: "call_1", Name: "read", Arguments: []byte(`{"path":"x"}`),
		},
	}, state)
	require.NoError(t, err)
	require.Len(t, chunks, 2)
	call := chunks[1].Choices[0].Delta.ToolCalls[0]
	assert.Equal(t, "call_1", call.ID)
	assert.Equal(t, "read", call.Function.Name)
	assert.Equal(t, `{"path":"x"}`, call.Function.Arguments)
}

func TestResponsesToolStreamRejectsSnapshotThatChangesSentArguments(t *testing.T) {
	for _, tt := range []struct {
		name      string
		eventType string
		arguments string
	}{
		{name: "arguments done changes sent bytes", eventType: responsesEventFunctionArgsDone, arguments: `{"path":"changed"}`},
		{name: "item done changes sent bytes", eventType: responsesEventOutputItemDone, arguments: `{"path":"changed"}`},
		{name: "completed changes sent bytes", eventType: responsesEventCompleted, arguments: `{"path":"changed"}`},
		{name: "snapshot shortens sent bytes", eventType: responsesEventCompleted, arguments: `{}`},
		{name: "same length snapshot changes sent bytes", eventType: responsesEventCompleted, arguments: `{"path":"y"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			state := NewResponsesToChatStreamState("gpt-test", false)
			index := 0
			_, err := ResponsesStreamEventToChatChunks(&dto.ResponsesStreamResponse{
				Type: responsesEventOutputItemAdded, OutputIndex: &index, Item: &dto.ResponsesOutput{
					Type: responsesOutputTypeFunctionCall, ID: "fc_1", CallId: "call_1", Name: "read",
				},
			}, state)
			require.NoError(t, err)
			_, err = ResponsesStreamEventToChatChunks(&dto.ResponsesStreamResponse{
				Type: responsesEventFunctionArgsDelta, OutputIndex: &index, Delta: `{"path":"x"}`,
			}, state)
			require.NoError(t, err)
			item := dto.ResponsesOutput{
				Type: responsesOutputTypeFunctionCall, ID: "fc_1", CallId: "call_1", Name: "read", Arguments: []byte(tt.arguments),
			}
			event := dto.ResponsesStreamResponse{Type: tt.eventType, OutputIndex: &index, Item: &item, Arguments: item.Arguments}
			if tt.eventType == responsesEventCompleted {
				event.Response = &dto.OpenAIResponsesResponse{Output: []dto.ResponsesOutput{item}}
			}
			chunks, err := ResponsesStreamEventToChatChunks(&event, state)
			require.Error(t, err, "a conflicting snapshot cannot be repaired by appending its tail")
			assert.Empty(t, chunks)
		})
	}
}
