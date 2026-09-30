package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

type Request struct {
	Model          string         `json:"model"`
	Messages       []Message      `json:"messages"`
	ResponseFormat ResponseFormat `json:"response_format"`
	Usage          *UsageOptions  `json:"usage,omitempty"`
}

type Reply struct {
	Content string
	Cost    float64
}

func NewRequest(model, schemaName string, schema map[string]any, messages ...Message) Request {
	return Request{
		Model:          model,
		Messages:       messages,
		ResponseFormat: ResponseFormat{Type: "json_schema", JSONSchema: JSONSchema{Name: schemaName, Strict: true, Schema: schema}},
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

func Post(ctx context.Context, hc *http.Client, url, apiKey string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := hc.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
		return &StatusError{Code: resp.StatusCode, Header: resp.Header, Body: string(respBody)}
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
