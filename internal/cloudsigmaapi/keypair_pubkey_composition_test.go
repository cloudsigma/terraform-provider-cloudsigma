package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestKeypairPubkeyCoexistence(t *testing.T) {
	calls := 0
	c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
		calls++
		expected := "/api/2.0/keypairs/fixture/"
		if calls == 2 {
			expected = "/api/2.0/pubkeys/fixture/"
		}
		if r.Method != "GET" || r.URL.Path != expected || r.URL.RawQuery != "" {
			t.Fatal("route crossover")
		}
		u, p, ok := r.BasicAuth()
		if !ok || u != "synthetic" || p != "synthetic" {
			t.Fatal("auth")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"uuid":"response","private_key":"SYNTHETIC-PRIVATE-SENTINEL","public_key":"SYNTHETIC-PUBLIC-SENTINEL","has_private_key":true}`))}, nil
	})
	key, err := c.Keypair(context.Background(), "fixture")
	if err != nil || key.PrivateKey != "SYNTHETIC-PRIVATE-SENTINEL" || key.PublicKey != "SYNTHETIC-PUBLIC-SENTINEL" || key.UUID != "response" {
		t.Fatal("keypair projection")
	}
	public, err := c.Pubkey(context.Background(), "fixture")
	if err != nil || public.PublicKey != key.PublicKey || public.UUID != key.UUID || !public.HasPrivateKey {
		t.Fatal("public projection")
	}
	b, err := json.Marshal(public)
	if err != nil || strings.Contains(string(b), `"private_key"`) || strings.Contains(string(b), "SYNTHETIC-PRIVATE-SENTINEL") {
		t.Fatal("private field crossed into public DTO")
	}
	if calls != 2 {
		t.Fatal("unexpected retry")
	}
}
