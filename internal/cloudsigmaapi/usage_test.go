package cloudsigmaapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestUsageAndBalanceNativeReads(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/2.0/usage/" || req.URL.Query().Get("poll_time_gt") != "2024-04-01" || req.URL.Query().Get("poll_time_lt") != "2024-04-02" || req.URL.Query().Get("limit") != "1500" {
			t.Fatalf("usage request=%s %s", req.Method, req.URL.String())
		}
		return remoteSnapshotResponse(http.StatusOK, `{"meta":{"limit":1500,"offset":0,"total_count":1},"objects":[{"amount":"6000","resource":"intel_cpu"}]}`), nil
	})
	got, err := c.Usage(context.Background(), UsageQuery{PollTimeGT: "2024-04-01", PollTimeLT: "2024-04-02", Limit: 1500})
	if err != nil || string(got["meta"]) != `{"limit":1500,"offset":0,"total_count":1}` {
		t.Fatalf("usage=%s err=%v", got["meta"], err)
	}
	c = remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/2.0/balance/" || req.URL.RawQuery != "" {
			t.Fatalf("balance request=%s", req.URL.String())
		}
		return remoteSnapshotResponse(http.StatusOK, `{"balance":"12.5","currency":"EUR"}`), nil
	})
	got, err = c.Balance(context.Background())
	if err != nil || string(got["currency"]) != `"EUR"` {
		t.Fatalf("balance=%s err=%v", got["currency"], err)
	}
}

func TestBurstUsageNativeReadsAndAuth(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/2.0/burstusage/" || req.URL.Query().Get("date__gte") != "2024-07-01" || req.URL.Query().Get("date__lt") != "2024-08-01" {
			t.Fatalf("burst request=%s", req.URL.String())
		}
		if _, _, ok := req.BasicAuth(); !ok {
			t.Fatal("missing basic auth")
		}
		return remoteSnapshotResponse(http.StatusOK, `{"meta":{"offset":0,"total_count":0},"objects":[]}`), nil
	})
	if _, err := c.BurstUsage(context.Background(), BurstUsageQuery{DateGTE: "2024-07-01", DateLT: "2024-08-01"}); err != nil {
		t.Fatal(err)
	}
	c = remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/2.0/dailyburstusage/" || req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("daily request=%s auth=%q", req.URL, req.Header.Get("Authorization"))
		}
		return remoteSnapshotResponse(http.StatusOK, `{"meta":{},"objects":[]}`), nil
	})
	c.username, c.password, c.token = "", "", "fixture-token"
	if _, err := c.DailyBurstUsage(context.Background(), BurstUsageQuery{}); err != nil {
		t.Fatal(err)
	}
}

func TestPricingAndDiscountNativeReads(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/2.0/pricing/" || req.URL.Query().Get("limit") != "5" {
			t.Fatalf("pricing request=%s", req.URL.String())
		}
		if req.Header.Get("Authorization") != "Basic Zml4dHVyZS11c2VyOmZpeHR1cmUtcGFzc3dvcmQ=" {
			t.Fatalf("pricing auth=%q", req.Header.Get("Authorization"))
		}
		return remoteSnapshotResponse(http.StatusOK, `{"current":{"cpu":1},"next":{"cpu":2},"meta":{"limit":5,"offset":0,"total_count":1},"objects":[{"resource":"cpu","price":"1"}]}`), nil
	})
	got, err := c.Pricing(context.Background(), 5)
	if err != nil || string(got["current"]) != `{"cpu":1}` {
		t.Fatalf("pricing=%s err=%v", got["current"], err)
	}
	c = remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/2.0/discount/" || req.URL.RawQuery != "" {
			t.Fatalf("discount request=%s", req.URL.String())
		}
		return remoteSnapshotResponse(http.StatusOK, `{"meta":{"total_count":1},"objects":[{"period":"3 months","value":"0.03"}]}`), nil
	})
	got, err = c.Discount(context.Background())
	if err != nil || string(got["objects"]) != `[{"period":"3 months","value":"0.03"}]` {
		t.Fatalf("discount=%s err=%v", got["objects"], err)
	}
}

func TestAuditLogsNativeRead(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/2.0/logs/" || req.URL.Query().Get("limit") != "5" {
			t.Fatalf("audit logs request=%s", req.URL.String())
		}
		if req.Header.Get("Authorization") != "Basic Zml4dHVyZS11c2VyOmZpeHR1cmUtcGFzc3dvcmQ=" {
			t.Fatalf("audit logs auth=%q", req.Header.Get("Authorization"))
		}
		return remoteSnapshotResponse(http.StatusOK, `{"meta":{"limit":5},"objects":[{"action":"create","success":true}]}`), nil
	})
	got, err := c.AuditLogs(context.Background(), 5)
	if err != nil || string(got["objects"]) != `[{"action":"create","success":true}]` {
		t.Fatalf("audit logs=%s err=%v", got["objects"], err)
	}
}

func TestLedgerNativeReadAndBearerAuth(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/2.0/ledger/" || req.URL.RawQuery != "" {
			t.Fatalf("ledger request=%s %s", req.Method, req.URL.String())
		}
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("ledger auth=%q", req.Header.Get("Authorization"))
		}
		return remoteSnapshotResponse(http.StatusOK, `{"meta":{"limit":20,"offset":0,"total_count":1},"objects":[{"id":"tx-1","amount":"0.01","reason":"fixture"}]}`), nil
	})
	c.username, c.password, c.token = "", "", "fixture-token"
	got, err := c.Ledger(context.Background())
	if err != nil || string(got["meta"]) != `{"limit":20,"offset":0,"total_count":1}` {
		t.Fatalf("ledger=%s err=%v", got["meta"], err)
	}
}

func TestDynamicUsageFailures(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{name: "redirect", status: http.StatusFound, body: "SECRET", want: "http_status"},
		{name: "unexpected", status: http.StatusAccepted, body: "SECRET", want: "unexpected_status"},
		{name: "malformed", status: http.StatusOK, body: `{`, want: "decode_error"},
		{name: "trailing", status: http.StatusOK, body: `{"objects":[]} {}`, want: "decode_error"},
		{name: "null", status: http.StatusOK, body: `null`, want: "decode_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return remoteSnapshotResponse(tc.status, tc.body), nil })
			_, err := c.Usage(context.Background(), UsageQuery{})
			if err == nil || !strings.HasPrefix(err.Error(), tc.want) || strings.Contains(err.Error(), "SECRET") {
				t.Fatalf("err=%v want=%s", err, tc.want)
			}
		})
	}
	c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return nil, errors.New("private transport") })
	if _, err := c.Balance(context.Background()); err == nil || err.Error() != "transport" {
		t.Fatalf("transport=%v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c = remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return nil, context.Canceled })
	if _, err := c.BurstUsage(ctx, BurstUsageQuery{}); err == nil || err.Error() != "cancelled" {
		t.Fatalf("cancel=%v", err)
	}
}
