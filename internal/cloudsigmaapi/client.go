// Package cloudsigmaapi implements the provider-owned HTTP transport.
package cloudsigmaapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const responseLimit = 1048576
const requestTimeout = 60 * time.Second

type doer interface {
	Do(*http.Request) (*http.Response, error)
}
type Client struct {
	transport                                  doer
	base, username, password, token, userAgent string
}

var locationLabel = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// New constructs a fixed-policy client. No arbitrary endpoint is accepted.
func New(location, username, password, userAgent string) (*Client, error) {
	return newClient(location, username, password, "", userAgent)
}

// NewToken constructs a fixed-policy bearer-token client for native read
// paths. It shares the same endpoint and redirect policy as Basic clients.
func NewToken(location, token, userAgent string) (*Client, error) {
	if token == "" {
		return nil, &Error{Category: "transport"}
	}
	return newClient(location, "", "", token, userAgent)
}

func newClient(location, username, password, token, userAgent string) (*Client, error) {
	location = strings.ToLower(location)
	if !locationLabel.MatchString(location) {
		return nil, &Error{Category: "transport"}
	}
	host := location + ".cloudsigma.com"
	switch location {
	case "dub":
		host = "ec.servecentric.com"
	case "ruh":
		host = "ruh.cld.v2.sa"
	case "adb":
		host = "siaflex.cloud"
	case "wdc":
		host = "wdc.alpha3cloud.com"
	}
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, &Error{Category: "transport"}
	}
	transport := t.Clone()
	transport.DisableCompression = true
	h := &http.Client{Transport: transport, Timeout: requestTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &Client{transport: h, base: "https://" + host + "/api/2.0/", username: username, password: password, token: token, userAgent: userAgent}, nil
}

func (c *Client) authenticate(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
		return
	}
	req.SetBasicAuth(c.username, c.password)
}

type readEndpoint uint8

const (
	profileEndpoint readEndpoint = iota
	locationsEndpoint
	licensesEndpoint
	subscriptionsEndpoint
	libraryDrivesEndpoint
	libraryDriveEndpoint
	drivesEndpoint
	driveEndpoint
	ipsEndpoint
	ipEndpoint
	vlansEndpoint
	vlanEndpoint
	tagsEndpoint
	snapshotEndpoint
	pubkeyEndpoint
	keypairEndpoint
	serverEndpoint
)

func (c *Client) get(ctx context.Context, endpoint readEndpoint) (any, error) {
	return c.getLibraryDrives(ctx, endpoint, "")
}

// getLibraryDrives accepts routes only from the closed endpoint methods.
func (c *Client) getLibraryDrives(ctx context.Context, endpoint readEndpoint, libraryRoute string) (any, error) {
	return c.getLibraryDrivesContext(ctx, endpoint, libraryRoute, true)
}

func (c *Client) getLibraryDrivesContext(ctx context.Context, endpoint readEndpoint, libraryRoute string, bounded bool) (any, error) {
	route := ""
	switch endpoint {
	case libraryDrivesEndpoint, libraryDriveEndpoint, drivesEndpoint, driveEndpoint, ipsEndpoint, ipEndpoint, vlansEndpoint, vlanEndpoint, snapshotEndpoint, pubkeyEndpoint, keypairEndpoint, serverEndpoint:
		route = libraryRoute
	case tagsEndpoint:
		route = "tags/"
	case profileEndpoint:
		route = "profile/"
	case locationsEndpoint:
		route = "locations/"
	case subscriptionsEndpoint:
		route = "subscriptions/"
	case licensesEndpoint:
		route = "licenses/"
	default:
		return nil, &Error{Category: "transport"}
	}
	transport := c.transport
	if bounded {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, requestTimeout)
		defer cancel()
	} else if h, ok := transport.(*http.Client); ok {
		clone := *h
		clone.Timeout = 0
		transport = &clone
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+route, nil)
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", c.userAgent)
	c.authenticate(req)
	resp, err := transport.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		category := "transport"
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
			category = "cancelled"
		} else if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			category = "timeout"
		}
		return nil, &Error{Category: category}
	}
	if resp == nil {
		return nil, &Error{Category: "transport"}
	}
	failure := func(category string) error {
		return &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	if resp.StatusCode != http.StatusOK {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return nil, failure(category)
	}
	encoding := resp.Header.Get("Content-Encoding")
	if encoding != "" && !strings.EqualFold(encoding, "identity") {
		return nil, failure("unsupported_encoding")
	}
	if resp.Body == nil {
		return nil, failure("decode_error")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		category := "transport"
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
			category = "cancelled"
		} else if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			category = "timeout"
		}
		return nil, failure(category)
	}
	if err := ctx.Err(); err != nil {
		category := "cancelled"
		if errors.Is(err, context.DeadlineExceeded) {
			category = "timeout"
		}
		return nil, failure(category)
	}
	if len(body) > responseLimit {
		return nil, failure("response_too_large")
	}
	var result any
	switch endpoint {
	case serverEndpoint:
		result, err = decodeServer(body)
	case keypairEndpoint:
		result, err = decodeKeypair(body)
	case pubkeyEndpoint:
		result, err = decodePubkey(body)
	case snapshotEndpoint:
		result, err = decodeSnapshot(body)
	case tagsEndpoint:
		result, err = decodeTags(body)
	case vlansEndpoint:
		result, err = decodeVLANs(body)
	case vlanEndpoint:
		result, err = decodeVLAN(body)
	case ipsEndpoint:
		result, err = decodeIPs(body)
	case ipEndpoint:
		result, err = decodeIP(body)
	case drivesEndpoint:
		result, err = decodeDrives(body)
	case driveEndpoint:
		result, err = decodeDrive(body)
	case libraryDrivesEndpoint:
		result, err = decodeLibraryDrives(body)
	case libraryDriveEndpoint:
		result, err = decodeLibraryDrive(body)
	case profileEndpoint:
		result, err = decodeProfile(body)
	case locationsEndpoint:
		result, err = decodeLocations(body)
	case subscriptionsEndpoint:
		result, err = decodeSubscriptions(body)
	case licensesEndpoint:
		result, err = decodeLicenses(body)
	}
	if err != nil {
		return nil, failure("decode_error")
	}
	if err := ctx.Err(); err != nil {
		category := "cancelled"
		if errors.Is(err, context.DeadlineExceeded) {
			category = "timeout"
		}
		return nil, failure(category)
	}
	return result, nil
}
