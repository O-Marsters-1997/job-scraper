package openrouter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const errorBodyLimit = 2 << 10

type StatusError struct {
	Code   int
	Header http.Header
	Body   string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("status %d: %s", e.Code, e.Body)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type JSONSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type ResponseFormat struct {
	Type       string     `json:"type"`
	JSONSchema JSONSchema `json:"json_schema"`
}

type UsageOptions struct {
	Include bool `json:"include"`
}

type ReasoningOptions struct {
	Effort string `json:"effort"`
}

type Request struct {
	Model          string            `json:"model"`
	Messages       []Message         `json:"messages"`
	ResponseFormat *ResponseFormat   `json:"response_format,omitempty"`
	Stream         bool              `json:"stream,omitempty"`
	Usage          *UsageOptions     `json:"usage,omitempty"`
	Reasoning      *ReasoningOptions `json:"reasoning,omitempty"`
}

type Reply struct {
	Content string
	Cost    float64
}

func NewRequest(model, schemaName string, schema map[string]any, messages ...Message) Request {
	return Request{
		Model:          model,
		Messages:       messages,
		ResponseFormat: &ResponseFormat{Type: "json_schema", JSONSchema: JSONSchema{Name: schemaName, Strict: true, Schema: schema}},
	}
}

func Chat(ctx context.Context, hc *http.Client, url, apiKey string, req Request) (Reply, error) {
	var decoded struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
		Usage struct {
			Cost float64 `json:"cost"`
		} `json:"usage"`
	}
	if err := Post(ctx, hc, url, apiKey, req, &decoded); err != nil {
		return Reply{}, err
	}
	if len(decoded.Choices) == 0 {
		return Reply{}, errors.New("no choices in response")
	}
	return Reply{Content: decoded.Choices[0].Message.Content, Cost: decoded.Usage.Cost}, nil
}

// ChatStream sends req as a streamed completion and calls onDelta with each
// content fragment as it arrives. The Reply carries the full content and the
// cost from the final usage chunk.
func ChatStream(ctx context.Context, hc *http.Client, url, apiKey string, req Request, onDelta func(string)) (Reply, error) {
	req.Stream = true
	resp, err := send(ctx, hc, url, apiKey, req)
	if err != nil {
		return Reply{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	var content strings.Builder
	var cost float64
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		payload, ok := strings.CutPrefix(scanner.Text(), "data: ")
		if !ok {
			continue
		}
		if payload == "[DONE]" {
			return Reply{Content: content.String(), Cost: cost}, nil
		}
		var chunk struct {
			Error   *struct{ Message string } `json:"error"`
			Choices []struct {
				Delta Message `json:"delta"`
			} `json:"choices"`
			Usage struct {
				Cost float64 `json:"cost"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return Reply{}, fmt.Errorf("decode stream chunk: %w", err)
		}
		if chunk.Error != nil {
			return Reply{Content: content.String(), Cost: cost}, fmt.Errorf("stream error: %s", chunk.Error.Message)
		}
		cost = max(cost, chunk.Usage.Cost)
		for _, c := range chunk.Choices {
			if c.Delta.Content == "" {
				continue
			}
			content.WriteString(c.Delta.Content)
			onDelta(c.Delta.Content)
		}
	}
	if err := scanner.Err(); err != nil {
		return Reply{Content: content.String(), Cost: cost}, fmt.Errorf("read stream: %w", err)
	}
	return Reply{Content: content.String(), Cost: cost}, errors.New("stream ended before [DONE]")
}

func Post(ctx context.Context, hc *http.Client, url, apiKey string, body, out any) error {
	resp, err := send(ctx, hc, url, apiKey, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func send(ctx context.Context, hc *http.Client, url, apiKey string, body any) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := hc.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
		return nil, &StatusError{Code: resp.StatusCode, Header: resp.Header, Body: string(respBody)}
	}
	return resp, nil
}
