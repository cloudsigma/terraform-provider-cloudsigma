package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestDrives(t *testing.T) {
	for _, single := range []bool{false, true} {
		for _, object := range []string{
			`{"uuid":"selected","name":"image","size":9007199254740993,"storage_type":null,"licenses":[{"amount":1,"license":{"free":9007199254740993,"initial_purchase_requirement":1.2300,"subsequent_purchase_requirement":1e99}}],"meta":{"raw":123},"owner":{"uuid":"owner"},"runtime":{"is_snapshotable":true,"snapshots_allocated_size":9007199254740993,"storage_type":"ssd"},"snapshots":[{"uuid":"snap"}],"remote_snapshots":[{"uuid":"remote"}],"mounted_on":[{"uuid":"server"}],"tags":[{"uuid":"tag","meta":{"n":123},"resources":[{"uuid":"res"}]}],"unknown":{}}`,
			`null`, `{}`, `{"size":null,"name":null,"category":null,"licenses":null}`, `{"licenses":[null,{"license":null}]}`, `{"name":"first","name":null}`, `{"licenses":[{"license":{"free":1,"free":null}}]}`,
			`{"size":"123"}`, `{"size":1.5}`, `{"size":9223372036854775808}`, `{"licenses":[{"license":{"free":true,"free":"ok"}}]}`, `{"runtime":[]}`, `[]`,
		} {
			t.Run(object, func(t *testing.T) {
				body := object
				if !single {
					body = `{"objects":[` + object + `],"meta":{"limit":0,"total_count":9,"next":"ignored"}}`
				}
				calls := 0
				c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
					calls++
					want := "/api/2.0/drives/selected/"
					query := ""
					if !single {
						want = "/api/2.0/drives/detail/"
						query = "limit=0&name=A%2CB+%26%2F%3F"
					}
					if r.URL.Path != want || r.URL.RawQuery != query || r.Method != "GET" || r.Body != nil || r.Header.Get("Accept-Encoding") != "identity" || r.Header.Get("Authorization") == "" {
						t.Fatal(r.URL)
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
				})
				oracle := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(func(r *http.Request) (*http.Response, error) {
					want := "/api/2.0/drives/selected/"
					query := ""
					if !single {
						want = "/api/2.0/drives/detail/"
						query = "limit=0&name=A%2CB+%26%2F%3F"
					}
					if r.URL.Path != want || r.URL.RawQuery != query {
						t.Fatal(r.URL)
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
				})}))
				var got, old any
				var e, oe error
				if single {
					got, e = c.Drive(context.Background(), "selected")
					old, _, oe = oracle.Drives.Get(context.Background(), "selected")
				} else {
					got, e = c.Drives(context.Background(), "A,B &/?")
					old, _, oe = oracle.Drives.List(context.Background(), &cloudsigma.DriveListOptions{ListOptions: cloudsigma.ListOptions{Limit: 0}, Names: []string{"A,B &/?"}})
				}
				if (e == nil) != (oe == nil) || calls != 1 {
					t.Fatalf("native %v oracle %v", e, oe)
				}
				if e == nil {
					a, _ := json.Marshal(got)
					b, _ := json.Marshal(old)
					if !reflect.DeepEqual(a, b) {
						t.Fatalf("%s != %s", a, b)
					}
				}
			})
		}
	}
	for _, id := range []string{"", "../profile", "a/b", "a?x", "a#x", "a%2fb", "a\n"} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("unsafe dispatch"); return nil, nil })
		if _, e := c.Drive(context.Background(), id); e == nil {
			t.Fatal(id)
		}
	}
	for _, body := range []string{`{}`, `null`, `{"objects":null}`, `{"objects":[]}`, `{"objects":{}}`, `{"meta":{"limit":"bad"}}`} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		_, e := c.Drives(context.Background(), "")
		if (e != nil) != (strings.Contains(body, `"objects":{}`) || strings.Contains(body, `"bad"`)) {
			t.Fatal(body, e)
		}
	}
}
func driveListReadError(t *testing.T) {
	b := &trackedBody{Reader: reviewFailReader{}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	_, e := c.Drives(context.Background(), "")
	if e == nil || e.Error() != "transport status=200" || !b.closed {
		t.Fatalf("%v close=%v", e, b.closed)
	}
}
func driveListEarlierDeadline(t *testing.T) {
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
	if _, e := c.Drives(ctx, ""); e != nil {
		t.Fatal(e)
	}
}

func driveListLateCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: cancel}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	p, e := c.Drives(ctx, "")
	if p != nil || e == nil || e.Error() != "cancelled status=200" || !b.closed {
		t.Fatalf("accepted profile after cancellation: %+v error=%v closed=%v", p, e, b.closed)
	}
}

func driveListDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: func() { <-ctx.Done() }}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	profile, err := c.Drives(ctx, "")
	if profile != nil || err == nil || err.Error() != "timeout status=200" || !b.closed {
		t.Fatalf("deadline: %v %v closed=%v", profile, err, b.closed)
	}
}

func TestDriveListBoundaries(t *testing.T) {
	t.Run("body-read", driveListReadError)
	t.Run("earlier-deadline", driveListEarlierDeadline)
	t.Run("late-cancel", driveListLateCancel)
	t.Run("deadline", driveListDeadline)
	for _, body := range []string{strings.Repeat(" ", responseLimit+1), `{"objects":[]}`} {
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > requestTimeout {
				t.Fatal("deadline")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Encoding": []string{"gzip"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, err := c.Drives(context.Background(), ""); err == nil {
			t.Fatal("encoding accepted")
		}
	}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", responseLimit+1)))}, nil
	})
	if _, err := c.Drives(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "response_too_large") {
		t.Fatal(err)
	}
}
func driveSingleReadError(t *testing.T) {
	b := &trackedBody{Reader: reviewFailReader{}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	_, e := c.Drive(context.Background(), "selected")
	if e == nil || e.Error() != "transport status=200" || !b.closed {
		t.Fatalf("%v close=%v", e, b.closed)
	}
}
func driveSingleEarlierDeadline(t *testing.T) {
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
	if _, e := c.Drive(ctx, "selected"); e != nil {
		t.Fatal(e)
	}
}

func driveSingleLateCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: cancel}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	p, e := c.Drive(ctx, "selected")
	if p != nil || e == nil || e.Error() != "cancelled status=200" || !b.closed {
		t.Fatalf("accepted profile after cancellation: %+v error=%v closed=%v", p, e, b.closed)
	}
}

func driveSingleDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: func() { <-ctx.Done() }}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	profile, err := c.Drive(ctx, "selected")
	if profile != nil || err == nil || err.Error() != "timeout status=200" || !b.closed {
		t.Fatalf("deadline: %v %v closed=%v", profile, err, b.closed)
	}
}

func TestDriveSingleBoundaries(t *testing.T) {
	t.Run("body-read", driveSingleReadError)
	t.Run("earlier-deadline", driveSingleEarlierDeadline)
	t.Run("late-cancel", driveSingleLateCancel)
	t.Run("deadline", driveSingleDeadline)
	for _, body := range []string{strings.Repeat(" ", responseLimit+1), `{"objects":[]}`} {
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > requestTimeout {
				t.Fatal("deadline")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Encoding": []string{"gzip"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, err := c.Drive(context.Background(), "selected"); err == nil {
			t.Fatal("encoding accepted")
		}
	}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", responseLimit+1)))}, nil
	})
	if _, err := c.Drive(context.Background(), "selected"); err == nil || !strings.Contains(err.Error(), "response_too_large") {
		t.Fatal(err)
	}
}

func TestDriveHTTP(t *testing.T) {
	for _, single := range []bool{false, true} {
		for _, code := range []int{201, 204, 301, 404, 500} {
			b := &trackedBody{Reader: strings.NewReader("PRIVATE")}
			c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: code, Body: b}, nil })
			var e error
			if single {
				_, e = c.Drive(context.Background(), "selected")
			} else {
				_, e = c.Drives(context.Background(), "")
			}
			if e == nil || strings.Contains(e.Error(), "PRIVATE") || !b.closed {
				t.Fatal(e)
			}
		}
	}
}
