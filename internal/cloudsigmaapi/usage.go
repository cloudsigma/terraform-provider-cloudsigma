package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/url"
	"strconv"
)

// DynamicDocument preserves deployment-specific read-only API documents whose
// fields vary by account, location, or billing configuration.
type DynamicDocument map[string]json.RawMessage

type UsageQuery struct {
	PollTimeGT string
	PollTimeLT string
	Limit      int64
}

type BurstUsageQuery struct {
	DateGT  string
	DateGTE string
	DateLT  string
	DateLTE string
}

func (c *Client) Usage(ctx context.Context, query UsageQuery) (DynamicDocument, error) {
	values := url.Values{}
	if query.PollTimeGT != "" {
		values.Set("poll_time_gt", query.PollTimeGT)
	}
	if query.PollTimeLT != "" {
		values.Set("poll_time_lt", query.PollTimeLT)
	}
	if query.Limit > 0 {
		values.Set("limit", formatInt(query.Limit))
	}
	return c.dynamicDocument(ctx, usageEndpoint, "usage/", values)
}

func (c *Client) Balance(ctx context.Context) (DynamicDocument, error) {
	return c.dynamicDocument(ctx, balanceEndpoint, "balance/", nil)
}

func (c *Client) BurstUsage(ctx context.Context, query BurstUsageQuery) (DynamicDocument, error) {
	values := url.Values{}
	if query.DateGT != "" {
		values.Set("date__gt", query.DateGT)
	}
	if query.DateGTE != "" {
		values.Set("date__gte", query.DateGTE)
	}
	if query.DateLT != "" {
		values.Set("date__lt", query.DateLT)
	}
	if query.DateLTE != "" {
		values.Set("date__lte", query.DateLTE)
	}
	return c.dynamicDocument(ctx, burstUsageEndpoint, "burstusage/", values)
}

func (c *Client) DailyBurstUsage(ctx context.Context, query BurstUsageQuery) (DynamicDocument, error) {
	values := url.Values{}
	if query.DateGT != "" {
		values.Set("date__gt", query.DateGT)
	}
	if query.DateGTE != "" {
		values.Set("date__gte", query.DateGTE)
	}
	if query.DateLT != "" {
		values.Set("date__lt", query.DateLT)
	}
	if query.DateLTE != "" {
		values.Set("date__lte", query.DateLTE)
	}
	return c.dynamicDocument(ctx, dailyBurstUsageEndpoint, "dailyburstusage/", values)
}

// Pricing reads the deployment pricing document. The optional limit is passed
// through to the API; no completeness is inferred when the server paginates.
func (c *Client) Pricing(ctx context.Context, limit int64) (DynamicDocument, error) {
	values := url.Values{}
	if limit > 0 {
		values.Set("limit", formatInt(limit))
	}
	return c.dynamicDocument(ctx, pricingEndpoint, "pricing/", values)
}

// Discount reads the account discount collection as dynamic JSON.
func (c *Client) Discount(ctx context.Context) (DynamicDocument, error) {
	return c.dynamicDocument(ctx, discountEndpoint, "discount/", nil)
}

// Ledger reads the account transaction collection as dynamic JSON. Transaction
// fields are deployment-specific and the response is paginated by the API.
func (c *Client) Ledger(ctx context.Context) (DynamicDocument, error) {
	return c.dynamicDocument(ctx, ledgerEndpoint, "ledger/", nil)
}

// AuditLogs reads the account audit-log collection as dynamic JSON. The API
// documents only the collection read and optional limit; log fields are kept
// dynamic because deployments may add fields without changing the transport.
func (c *Client) AuditLogs(ctx context.Context, limit int64) (DynamicDocument, error) {
	values := url.Values{}
	if limit > 0 {
		values.Set("limit", formatInt(limit))
	}
	return c.dynamicDocument(ctx, logsEndpoint, "logs/", values)
}

func (c *Client) dynamicDocument(ctx context.Context, endpoint readEndpoint, route string, values url.Values) (DynamicDocument, error) {
	if len(values) > 0 {
		route += "?" + values.Encode()
	}
	v, err := c.getLibraryDrives(ctx, endpoint, route)
	if err != nil {
		return nil, err
	}
	return v.(DynamicDocument), nil
}

func decodeDynamicDocument(body []byte) (DynamicDocument, error) {
	var object map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&object); err != nil || object == nil {
		return nil, &Error{Category: "decode_error"}
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, &Error{Category: "decode_error"}
	}
	return DynamicDocument(object), nil
}

func formatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}
