package cloudsigma

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

const reviewBaseline = false

func native26Transport(t *testing.T, handler func(*http.Request) (*http.Response, error)) *cloudsigmaapi.Client {
	t.Helper()
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = &http.Transport{DialTLSContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		a, b := net.Pipe()
		go func() {
			defer b.Close()
			r, e := http.ReadRequest(bufio.NewReader(b))
			if e != nil {
				return
			}
			r = r.WithContext(ctx)
			res, e := handler(r)
			if e != nil {
				return
			}
			defer res.Body.Close()
			_, _ = fmt.Fprintf(b, "HTTP/1.1 %d fixture\r\nConnection: close\r\n\r\n", res.StatusCode)
			_, _ = io.Copy(b, res.Body)
		}()
		return a, nil
	}}
	c, e := cloudsigmaapi.New("zrh", "u", "p", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	return c
}
