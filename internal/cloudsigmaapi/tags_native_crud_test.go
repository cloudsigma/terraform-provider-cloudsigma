package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNativeTagCRUDUsesDocumentedRoutesAndProjection(t *testing.T) {
	var calls []string
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		user, password, ok := req.BasicAuth()
		if !ok || user != "fixture-user" || password != "fixture-password" {
			t.Fatalf("authorization=%q", req.Header.Get("Authorization"))
		}
		calls = append(calls, req.Method+" "+req.URL.Path)
		var body []byte
		if req.Body != nil {
			body, _ = io.ReadAll(req.Body)
		}
		switch req.Method {
		case http.MethodPost:
			var doc map[string]any
			if err := json.Unmarshal(body, &doc); err != nil {
				t.Fatal(err)
			}
			objects := doc["objects"].([]any)
			if len(objects) != 1 || len(objects[0].(map[string]any)) != 1 || objects[0].(map[string]any)["name"] != "blue" {
				t.Fatalf("create body=%s", body)
			}
			return remoteSnapshotResponse(http.StatusCreated, `{"objects":[{"uuid":"tag-1","name":"blue","resource_uri":"/api/2.0/tags/tag-1/","owner":{"uuid":"private"},"resources":[{"uuid":"private"}]}]}`), nil
		case http.MethodGet:
			return remoteSnapshotResponse(http.StatusOK, `{"uuid":"tag-1","name":"blue","resource_uri":"/api/2.0/tags/tag-1/","owner":{"uuid":"private"},"resources":[{"uuid":"private"}]}`), nil
		case http.MethodPut:
			var doc map[string]any
			if err := json.Unmarshal(body, &doc); err != nil || len(doc) != 1 || doc["name"] != "green" {
				t.Fatalf("update body=%s", body)
			}
			return remoteSnapshotResponse(http.StatusOK, `{"uuid":"tag-1","name":"green","resource_uri":"/api/2.0/tags/tag-1/"}`), nil
		case http.MethodDelete:
			return remoteSnapshotResponse(http.StatusNoContent, ""), nil
		default:
			return nil, &Error{Category: "transport"}
		}
	})

	created, err := c.CreateTag(context.Background(), TagCreateRequest{Name: "blue"})
	if err != nil || created.UUID != "tag-1" || created.Name != "blue" || created.ResourceURI != "/api/2.0/tags/tag-1/" {
		t.Fatalf("create=%+v err=%v", created, err)
	}
	read, err := c.Tag(context.Background(), "tag-1")
	if err != nil || read.UUID != "tag-1" || read.Name != "blue" {
		t.Fatalf("read=%+v err=%v", read, err)
	}
	updated, err := c.UpdateTag(context.Background(), "tag-1", TagUpdateRequest{Name: "green"})
	if err != nil || updated.Name != "green" {
		t.Fatalf("update=%+v err=%v", updated, err)
	}
	if err := c.DeleteTag(context.Background(), "tag-1"); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(calls, ","), "POST /api/2.0/tags/,GET /api/2.0/tags/tag-1/,PUT /api/2.0/tags/tag-1/,DELETE /api/2.0/tags/tag-1/"; got != want {
		t.Fatalf("calls=%s want=%s", got, want)
	}
}

func TestNativeTagCRUDRejectsMalformedAcknowledgements(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		path   string
		status int
		body   string
	}{
		{"create status", http.MethodPost, "tags/", http.StatusOK, `{"objects":[{"uuid":"tag"}]}`},
		{"redirect refused", http.MethodPost, "tags/", http.StatusFound, `{"objects":[{"uuid":"tag"}]}`},
		{"create malformed", http.MethodPost, "tags/", http.StatusCreated, `{"objects":[`},
		{"get malformed", http.MethodGet, "tags/tag/", http.StatusOK, `{"uuid":""}`},
		{"update malformed", http.MethodPut, "tags/tag/", http.StatusOK, `{"uuid":""}`},
		{"delete status", http.MethodDelete, "tags/tag/", http.StatusOK, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
				return remoteSnapshotResponse(tc.status, tc.body), nil
			})
			var err error
			switch tc.method {
			case http.MethodPost:
				_, err = c.CreateTag(context.Background(), TagCreateRequest{Name: "blue"})
			case http.MethodGet:
				_, err = c.Tag(context.Background(), "tag")
			case http.MethodPut:
				_, err = c.UpdateTag(context.Background(), "tag", TagUpdateRequest{Name: "green"})
			default:
				err = c.DeleteTag(context.Background(), "tag")
			}
			if err == nil {
				t.Fatal("malformed acknowledgement accepted")
			}
		})
	}
}

func TestNativeTagWriteContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
		t.Fatal("cancelled request reached transport")
		return nil, nil
	})
	if err := c.DeleteTag(ctx, "tag"); err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("err=%v", err)
	}
}

func TestNativeTagCreateUsesBearerAuthentication(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer fixture-token" {
			t.Fatalf("authorization=%q", got)
		}
		return remoteSnapshotResponse(http.StatusCreated, `{"objects":[{"uuid":"tag-token","name":"blue"}]}`), nil
	})
	c.username, c.password, c.token = "", "", "fixture-token"
	if _, err := c.CreateTag(context.Background(), TagCreateRequest{Name: "blue"}); err != nil {
		t.Fatal(err)
	}
}
