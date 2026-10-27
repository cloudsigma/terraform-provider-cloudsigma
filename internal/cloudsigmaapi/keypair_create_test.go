package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestKeypairCreate(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		var got map[string]any
		body := &trackedBody{Reader: strings.NewReader(`{"objects":[{"uuid":"created","name":"name","public_key":"PUB","private_key":"PRIV","resource_uri":"/keypairs/created/"}]}`)}
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/2.0/keypairs/" || r.URL.RawQuery != "" {
				t.Fatal(r)
			}
			u, p, ok := r.BasicAuth()
			if !ok || u != "synthetic" || p != "synthetic" {
				t.Fatal("auth")
			}
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			return &http.Response{StatusCode: status, Header: make(http.Header), Body: body, Request: r}, nil
		})
		gotKey, err := c.CreateKeypair(context.Background(), KeypairCreateRequest{Name: "name", PrivateKey: "PRIV", PublicKey: "PUB"})
		if err != nil || gotKey.UUID != "created" || gotKey.ResourceURI != "/keypairs/created/" || !body.closed {
			t.Fatalf("status=%d key=%+v err=%v closed=%v", status, gotKey, err, body.closed)
		}
		want := map[string]any{"objects": []any{map[string]any{"name": "name", "private_key": "PRIV", "public_key": "PUB"}}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("request projection: %#v", got)
		}
	}
}

func TestKeypairCreateBoundaries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("pre-cancel dispatch"); return nil, nil })
	if _, err := c.CreateKeypair(ctx, KeypairCreateRequest{}); err == nil || err.Error() != "cancelled" {
		t.Fatal(err)
	}

	for _, status := range []int{http.StatusNoContent, http.StatusMovedPermanently, http.StatusBadRequest, http.StatusInternalServerError} {
		body := &trackedBody{Reader: strings.NewReader("PRIVATE")}
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Header: make(http.Header), Body: body}, nil
		})
		if _, err := c.CreateKeypair(context.Background(), KeypairCreateRequest{}); err == nil || strings.Contains(err.Error(), "PRIVATE") || !body.closed {
			t.Fatalf("status=%d err=%v closed=%v", status, err, body.closed)
		}
	}

	for _, payload := range []string{"", `{}`, `{"objects":[]}`, `{"objects":[{}]}`, `{"objects":[{"uuid":"a/b"}]}`, `{"objects":[{"uuid":"ok"}]} trailing`} {
		body := &trackedBody{Reader: strings.NewReader(payload)}
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusCreated, Header: make(http.Header), Body: body}, nil
		})
		key, err := c.CreateKeypair(context.Background(), KeypairCreateRequest{})
		if payload == `{"objects":[{"uuid":"ok"}]}` {
			if err != nil || key.UUID != "ok" {
				t.Fatal(key, err)
			}
		} else if err == nil || key != nil || !body.closed {
			t.Fatalf("payload=%q key=%+v err=%v", payload, key, err)
		}
	}

	c = reviewNativeNew(func(*http.Request) (*http.Response, error) { return nil, errors.New("PRIVATE transport") })
	if _, err := c.CreateKeypair(context.Background(), KeypairCreateRequest{}); err == nil || err.Error() != "transport" {
		t.Fatal(err)
	}
}
