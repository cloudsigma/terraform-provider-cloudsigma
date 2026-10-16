package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type denyRT struct{}

func (denyRT) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("offline network denied")
}
func TestOfflineRegisteredSchemas(t *testing.T) {
	old := http.DefaultTransport
	http.DefaultTransport = denyRT{}
	defer func() { http.DefaultTransport = old }()
	p := New("offline")()
	ds := p.DataSources(context.Background())
	rs := p.Resources(context.Background())
	if len(ds) != 9 || len(rs) != 3 {
		t.Fatal("registration drift")
	}
	for _, newDS := range ds {
		d := newDS()
		var out datasource.SchemaResponse
		d.Schema(context.Background(), datasource.SchemaRequest{}, &out)
		if out.Diagnostics.HasError() || len(out.Schema.Attributes) == 0 {
			t.Fatal("datasource schema")
		}
	}
	for _, newR := range rs {
		r := newR()
		var out resource.SchemaResponse
		r.Schema(context.Background(), resource.SchemaRequest{}, &out)
		if out.Diagnostics.HasError() || len(out.Schema.Attributes) == 0 {
			t.Fatal("resource schema")
		}
	}
}
