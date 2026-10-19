package cloudsigmaapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// reviewNativeNew is test-only and overlaid, never delivered or configured publicly.
func reviewNativeNew(f func(*http.Request) (*http.Response, error)) *Client {
	c, _ := New("zrh", "synthetic", "synthetic", "")
	c.transport = fakeDoer(f)
	return c
}

type reviewFailReader struct{}

func (reviewFailReader) Read([]byte) (int, error) { return 0, errors.New("PRIVATE-READ-ERROR") }
func TestReviewReadError(t *testing.T) {
	b := &trackedBody{Reader: reviewFailReader{}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	_, e := c.Profile(context.Background())
	if e == nil || e.Error() != "transport status=200" || !b.closed {
		t.Fatalf("%v close=%v", e, b.closed)
	}
}
func TestReviewEarlierDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	want, _ := ctx.Deadline()
	c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
		got, _ := r.Context().Deadline()
		if !got.Equal(want) {
			t.Fatal("deadline extended")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})
	if _, e := c.Profile(ctx); e != nil {
		t.Fatal(e)
	}
}

type reviewCancelReader struct {
	cancel context.CancelFunc
	sent   bool
}

func (r *reviewCancelReader) Read(b []byte) (int, error) {
	if r.sent {
		return 0, io.EOF
	}
	r.sent = true
	r.cancel()
	return copy(b, `{"uuid":"late"}`), io.EOF
}
func TestReviewCancellationDuringRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: cancel}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	p, e := c.Profile(ctx)
	if p != nil || e == nil || e.Error() != "cancelled status=200" || !b.closed {
		t.Fatalf("accepted profile after cancellation: %+v error=%v closed=%v", p, e, b.closed)
	}
}

func TestReviewDeadlineDuringRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: func() { <-ctx.Done() }}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	profile, err := c.Profile(ctx)
	if profile != nil || err == nil || err.Error() != "timeout status=200" || !b.closed {
		t.Fatalf("deadline: %v %v closed=%v", profile, err, b.closed)
	}
}
