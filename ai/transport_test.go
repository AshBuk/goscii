// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/v2/option"

	"github.com/AshBuk/goscii/engine"
)

var testSchema = map[string]any{"type": "object", "properties": map[string]any{"title": map[string]any{"type": "string"}}}

func chatServer(t *testing.T, handle func(w http.ResponseWriter, body map[string]any)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			_, _ = io.WriteString(w, `{"data":[{"id":"m1"},{"id":"m2"}]}`)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		handle(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func ok(text string) string {
	b, _ := json.Marshal(map[string]any{
		"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": text}, "finish_reason": "stop"}},
		"usage":   map[string]any{"prompt_tokens": 10, "completion_tokens": 5},
	})
	return string(b)
}

func TestChatSendsStrictSchemaAndNoTemperature(t *testing.T) {
	var got map[string]any
	srv := chatServer(t, func(w http.ResponseWriter, body map[string]any) {
		got = body
		_, _ = io.WriteString(w, ok(`{"title":"x"}`))
	})
	c := NewChat("openai", srv.URL, "k", "gpt-6.1-sol", true)
	r, err := c.Complete(context.Background(), Prompt{System: "sys", Turns: []Turn{User("hi")}, Schema: testSchema, Effort: "low"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Text() != `{"title":"x"}` || r.Usage.Input != 10 {
		t.Fatalf("bad reply %+v", r)
	}
	if _, has := got["temperature"]; has {
		t.Fatal("temperature must not be sent")
	}
	rf := got["response_format"].(map[string]any)
	if rf["type"] != "json_schema" || rf["json_schema"].(map[string]any)["strict"] != true {
		t.Fatalf("want strict json_schema, got %v", rf)
	}
	if got["reasoning_effort"] != "low" {
		t.Fatalf("want reasoning_effort low, got %v", got["reasoning_effort"])
	}
}

func TestChatFallsBackToJSONObject(t *testing.T) {
	var formats []string
	srv := chatServer(t, func(w http.ResponseWriter, body map[string]any) {
		rf := body["response_format"].(map[string]any)
		formats = append(formats, rf["type"].(string))
		if rf["type"] == "json_schema" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":{"message":"json_schema not supported"}}`)
			return
		}
		sys := body["messages"].([]any)[0].(map[string]any)["content"].(string)
		if !strings.Contains(sys, "JSON schema") {
			t.Error("fallback must spell the schema out in the system prompt")
		}
		_, _ = io.WriteString(w, ok(`{"title":"x"}`))
	})
	c := NewChat("custom", srv.URL, "", "local", false)
	for range 2 {
		if _, err := c.Complete(context.Background(), Prompt{Turns: []Turn{User("hi")}, Schema: testSchema}); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Join(formats, ",") != "json_schema,json_object,json_object" {
		t.Fatalf("want one schema attempt then json_object, got %v", formats)
	}
}

func TestChatMapsFailures(t *testing.T) {
	status := http.StatusUnauthorized
	finish := "stop"
	srv := chatServer(t, func(w http.ResponseWriter, _ map[string]any) {
		if status != http.StatusOK {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"error":{"message":"nope"}}`)
			return
		}
		_, _ = io.WriteString(w, strings.Replace(ok("partial"), `"stop"`, `"`+finish+`"`, 1))
	})
	c := NewChat("groq", srv.URL, "bad", "m", true)
	if _, err := c.Complete(context.Background(), Prompt{Turns: []Turn{User("x")}}); !errors.Is(err, ErrKeyRejected) {
		t.Fatalf("401 must map to ErrKeyRejected, got %v", err)
	}
	status, finish = http.StatusOK, "length"
	if _, err := c.Complete(context.Background(), Prompt{Turns: []Turn{User("x")}}); !errors.Is(err, ErrTruncated) {
		t.Fatalf("finish_reason length must map to ErrTruncated, got %v", err)
	}
	ids, err := c.Models(context.Background())
	if err != nil || strings.Join(ids, ",") != "m1,m2" {
		t.Fatalf("models: %v %v", ids, err)
	}
}

func anthropicServer(t *testing.T, got *map[string]any, header *http.Header, reply string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(got)
		*header = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv
}

const anthropicOK = `{"id":"msg_1","type":"message","role":"assistant","model":"m","content":[{"type":"text","text":"{\"title\":\"x\"}"}],"stop_reason":"end_turn","usage":{"input_tokens":7,"output_tokens":3}}`

func TestAnthropicRequestShape(t *testing.T) {
	var got map[string]any
	var header http.Header
	srv := anthropicServer(t, &got, &header, anthropicOK)
	a := NewAnthropic("k", "claude-sonnet-5-5", option.WithBaseURL(srv.URL), option.WithMaxRetries(0))
	r, err := a.Complete(context.Background(), Prompt{System: "sys", Turns: []Turn{User("hi")}, Schema: testSchema, Effort: "medium"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Text() != `{"title":"x"}` || r.Usage.Output != 3 {
		t.Fatalf("bad reply %+v", r)
	}
	oc := got["output_config"].(map[string]any)
	if oc["effort"] != "medium" || oc["format"].(map[string]any)["type"] != "json_schema" {
		t.Fatalf("bad output_config %v", oc)
	}
	if got["max_tokens"].(float64) < 8000 {
		t.Fatalf("max_tokens must leave room for thinking, got %v", got["max_tokens"])
	}
	if _, has := got["temperature"]; has {
		t.Fatal("temperature must not be sent")
	}
	if got["fallbacks"] != "default" || !strings.Contains(header.Get("anthropic-beta"), "server-side-fallback") {
		t.Fatalf("sonnet 5.5 must opt into refusal fallbacks: %v / %q", got["fallbacks"], header.Get("anthropic-beta"))
	}

	got = nil
	srv2 := anthropicServer(t, &got, &header, anthropicOK)
	h := NewAnthropic("k", "claude-haiku-5-5", option.WithBaseURL(srv2.URL), option.WithMaxRetries(0))
	if _, err := h.Complete(context.Background(), Prompt{Turns: []Turn{User("hi")}}); err != nil {
		t.Fatal(err)
	}
	if _, has := got["fallbacks"]; has {
		t.Fatal("haiku has no server-side fallback")
	}
}

func TestAnthropicStopReasons(t *testing.T) {
	var got map[string]any
	var header http.Header
	for reason, want := range map[string]error{"max_tokens": ErrTruncated, "refusal": ErrRefused} {
		reply := strings.Replace(anthropicOK, `"end_turn"`, `"`+reason+`"`, 1)
		srv := anthropicServer(t, &got, &header, reply)
		a := NewAnthropic("k", "claude-opus-5-5", option.WithBaseURL(srv.URL), option.WithMaxRetries(0))
		if _, err := a.Complete(context.Background(), Prompt{Turns: []Turn{User("hi")}}); !errors.Is(err, want) {
			t.Errorf("%s: want %v, got %v", reason, want, err)
		}
	}
}

func TestDialResolvesRetiredModels(t *testing.T) {
	cases := map[engine.Provider]string{
		engine.ProviderGroq:      "llama-3.3-70b-versatile",
		engine.ProviderOpenAI:    "gpt-5",
		engine.ProviderAnthropic: "claude-sonnet-4-6",
	}
	for p, old := range cases {
		sig, err := Dial(engine.Config{Provider: p, APIKey: "k", Model: old})
		if err != nil {
			t.Fatal(err)
		}
		if sig.Model() == old || !Bands[p].IsVerified(sig.Model()) {
			t.Errorf("%s: %s should resolve to a verified model, got %s", p, old, sig.Model())
		}
	}
	if sig, _ := Dial(engine.Config{Provider: engine.ProviderOpenAI, APIKey: "k", Model: "my-finetune"}); sig.Model() != "my-finetune" {
		t.Error("a hand-typed model ID must pass through")
	}
	if _, err := Dial(engine.Config{Provider: engine.ProviderCustom, Model: "x"}); err == nil {
		t.Error("custom without an endpoint must fail")
	}
}
