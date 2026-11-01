package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// BackupSchedulers reads the complete GET-only scheduler collection. It does
// not infer assignment, readiness, or deletion semantics.
func (c *Client) BackupSchedulers(ctx context.Context) ([]json.RawMessage, error) {
	v, err := c.getLibraryDrives(ctx, backupSchedulersEndpoint, "backupschedulers/?limit=0")
	if err != nil {
		return nil, err
	}
	return v.([]json.RawMessage), nil
}

// BackupScheduler reads one scheduler detail document by UUID.
func (c *Client) BackupScheduler(ctx context.Context, uuid string) (json.RawMessage, error) {
	if !driveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	v, err := c.getLibraryDrives(ctx, backupSchedulerEndpoint, "backupschedulers/"+uuid+"/")
	if err != nil {
		return nil, err
	}
	return v.(json.RawMessage), nil
}

func decodeRawCollection(body []byte) ([]json.RawMessage, error) {
	var root struct {
		Meta *struct {
			Offset     *int `json:"offset"`
			TotalCount *int `json:"total_count"`
		} `json:"meta"`
		Objects []json.RawMessage `json:"objects"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&root); err != nil || root.Objects == nil {
		return nil, &Error{Category: "decode_error"}
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, &Error{Category: "decode_error"}
	}
	if root.Meta == nil || root.Meta.Offset == nil || root.Meta.TotalCount == nil || *root.Meta.Offset != 0 || *root.Meta.TotalCount != len(root.Objects) {
		return nil, &Error{Category: "incomplete_response"}
	}
	for _, object := range root.Objects {
		var identity struct {
			UUID string `json:"uuid"`
		}
		if json.Unmarshal(object, &identity) != nil || !driveSegment.MatchString(identity.UUID) {
			return nil, &Error{Category: "decode_error"}
		}
	}
	return root.Objects, nil
}

func decodeRawObject(body []byte) (json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	var object json.RawMessage
	if err := decoder.Decode(&object); err != nil {
		return nil, &Error{Category: "decode_error"}
	}
	var identity struct {
		UUID string `json:"uuid"`
	}
	if json.Unmarshal(object, &identity) != nil || !driveSegment.MatchString(identity.UUID) {
		return nil, &Error{Category: "decode_error"}
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, &Error{Category: "decode_error"}
	}
	return object, nil
}

// SchedulerClock is the documented hour/minute time window used by scheduler
// initial and incremental backup definitions.
type SchedulerClock struct {
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

// SchedulerRepeat describes the documented repeat interval. Empty strings are
// meaningful and are preserved when supplied by the caller.
type SchedulerRepeat struct {
	Hour   string `json:"hour"`
	Minute string `json:"minute"`
}

// SchedulerInitialBackup contains the documented optional initial backup
// window. Response-only fields are intentionally absent.
type SchedulerInitialBackup struct {
	DayOfWeek any `json:"day_of_week,omitempty"`
	StartTime any `json:"start_time,omitempty"`
	EndTime   any `json:"end_time,omitempty"`
}

// SchedulerIncrementalBackup contains the documented recurring schedule
// fields. Month accepts either the documented string form (for example "*")
// or a []string form used by the public examples.
type SchedulerIncrementalBackup struct {
	DayOfMonth string           `json:"day_of_month,omitempty"`
	DayOfWeek  []string         `json:"day_of_week,omitempty"`
	Month      any              `json:"month,omitempty"`
	Hour       string           `json:"hour,omitempty"`
	Minute     string           `json:"minute,omitempty"`
	Repeat     *SchedulerRepeat `json:"repeat,omitempty"`
	StartTime  *SchedulerClock  `json:"start_time,omitempty"`
	EndTime    *SchedulerClock  `json:"end_time,omitempty"`
}

// SchedulerRetentionRule is one documented retention period.
type SchedulerRetentionRule struct {
	Period     string `json:"period,omitempty"`
	Quantity   int    `json:"quantity,omitempty"`
	TimeOption *int   `json:"time_option,omitempty"`
}

// SchedulerRetentionPolicy defines the documented retention rules.
type SchedulerRetentionPolicy struct {
	Name  string                   `json:"name,omitempty"`
	Rules []SchedulerRetentionRule `json:"rules"`
}

// SchedulerTagRef is the minimal writable tag reference accepted by the
// scheduler API. Response-only tag fields are intentionally not represented.
type SchedulerTagRef struct {
	UUID string `json:"uuid"`
}

// SchedulerCreateRequest is the canonical provider-owned writable projection
// derived from the public field reference and scheduler PUT example. It does
// not include response-only owner, permissions, resource_uri, grantees, or
// uuid fields.
type SchedulerCreateRequest struct {
	Type                               string                     `json:"type"`
	Name                               string                     `json:"name"`
	EnableBackupPolicyNotifications    bool                       `json:"enable_backup_policy_notifications"`
	EnableRetentionPolicyNotifications bool                       `json:"enable_retention_policy_notifications"`
	IsEnabled                          bool                       `json:"is_enabled"`
	IsDefault                          bool                       `json:"is_default"`
	ManualIncrementalBackup            string                     `json:"manual_incremental_backup,omitempty"`
	RetentionPolicy                    SchedulerRetentionPolicy   `json:"retention_policy"`
	InitialBackup                      *SchedulerInitialBackup    `json:"initial_backup,omitempty"`
	IncrementalBackup                  SchedulerIncrementalBackup `json:"incremental_backup"`
	UserTimezone                       string                     `json:"user_timezone"`
	OrchestratorTimezone               string                     `json:"orchestrator_timezone,omitempty"`
	RemoteLocation                     string                     `json:"remote_location,omitempty"`
	Tags                               []SchedulerTagRef          `json:"tags,omitempty"`
	Meta                               map[string]any             `json:"meta,omitempty"`
}

// Scheduler is the acknowledged identity and state returned by create. The
// full response is deliberately not used as a write request model.
type Scheduler struct {
	UUID        string `json:"uuid"`
	ResourceURI string `json:"resource_uri,omitempty"`
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	IsEnabled   bool   `json:"is_enabled,omitempty"`
	IsDefault   bool   `json:"is_default,omitempty"`
}

// CreateBackupScheduler creates one scheduler using the documented POST
// envelope. It requires HTTP 201 and one acknowledged UUID; it does not bind
// a drive, poll asynchronous actions, or infer deletion/readiness semantics.
func (c *Client) CreateBackupScheduler(ctx context.Context, input SchedulerCreateRequest) (*Scheduler, error) {
	if err := validateSchedulerCreate(input); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	payload, err := json.Marshal(struct {
		Objects []SchedulerCreateRequest `json:"objects"`
	}{Objects: []SchedulerCreateRequest{input}})
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	transport := c.transport
	if h, ok := transport.(*http.Client); ok {
		clone := *h
		clone.Timeout = 0
		transport = &clone
	}
	if err := ctx.Err(); err != nil {
		return nil, remoteContextError(ctx, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"backupschedulers/", bytes.NewReader(payload))
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
		return nil, remoteContextError(ctx, err)
	}
	if resp == nil {
		return nil, &Error{Category: "transport"}
	}
	fail := func(category string) (*Scheduler, error) {
		return nil, &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	if resp.StatusCode != http.StatusCreated {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return fail(category)
	}
	if enc := resp.Header.Get("Content-Encoding"); enc != "" && !strings.EqualFold(enc, "identity") {
		return fail("unsupported_encoding")
	}
	if resp.Body == nil {
		return fail("decode_error")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, remoteContextError(ctx, ctxErr)
		}
		return fail("transport")
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, remoteContextError(ctx, ctxErr)
	}
	if len(body) > responseLimit {
		return fail("response_too_large")
	}
	var document struct {
		Objects []Scheduler `json:"objects"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&document); err != nil {
		return fail("decode_error")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fail("decode_error")
	}
	if len(document.Objects) != 1 || !driveSegment.MatchString(document.Objects[0].UUID) {
		return fail("decode_error")
	}
	return &document.Objects[0], nil
}

// UpdateBackupScheduler sends the full writable scheduler projection. The API
// does not provide patch semantics; callers must preserve required nested
// blocks such as initial_backup when the deployment requires them.
func (c *Client) UpdateBackupScheduler(ctx context.Context, uuid string, input SchedulerCreateRequest) (*Scheduler, error) {
	if !driveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	if err := validateSchedulerCreate(input); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.base+"backupschedulers/"+uuid+"/", bytes.NewReader(payload))
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", c.userAgent)
	c.authenticate(req)
	transport := c.transport
	if h, ok := transport.(*http.Client); ok {
		clone := *h
		clone.Timeout = 0
		transport = &clone
	}
	resp, err := transport.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, remoteContextError(ctx, err)
	}
	if resp == nil {
		return nil, &Error{Category: "transport"}
	}
	fail := func(category string) (*Scheduler, error) {
		return nil, &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	if resp.StatusCode != http.StatusOK {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return fail(category)
	}
	if enc := resp.Header.Get("Content-Encoding"); enc != "" && !strings.EqualFold(enc, "identity") {
		return fail("unsupported_encoding")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		return nil, remoteContextError(ctx, err)
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, remoteContextError(ctx, ctxErr)
	}
	if len(body) > responseLimit {
		return fail("response_too_large")
	}
	var scheduler Scheduler
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&scheduler); err != nil || scheduler.UUID != uuid {
		return fail("decode_error")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fail("decode_error")
	}
	return &scheduler, nil
}

// DeleteBackupScheduler acknowledges scheduler deletion with HTTP 204. It
// does not infer durable absence from the acknowledgement.
func (c *Client) DeleteBackupScheduler(ctx context.Context, uuid string) error {
	if !driveSegment.MatchString(uuid) {
		return &Error{Category: "transport"}
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.base+"backupschedulers/"+uuid+"/", nil)
	if err != nil {
		return &Error{Category: "transport"}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", c.userAgent)
	c.authenticate(req)
	resp, err := c.transport.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return remoteContextError(ctx, err)
	}
	if resp == nil {
		return &Error{Category: "transport"}
	}
	if resp.StatusCode != http.StatusNoContent {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	return nil
}

func validateSchedulerCreate(input SchedulerCreateRequest) error {
	if input.Type != "backup" && input.Type != "snapshot" && input.Type != "migration" && input.Type != "dr" {
		return &Error{Category: "transport"}
	}
	if input.Name == "" || input.UserTimezone == "" || input.IncrementalBackup.Repeat == nil {
		return &Error{Category: "transport"}
	}
	for _, tag := range input.Tags {
		if !driveSegment.MatchString(tag.UUID) {
			return &Error{Category: "transport"}
		}
	}
	for key, value := range input.Meta {
		if len(key) == 0 || len(key) > 32 {
			return &Error{Category: "transport"}
		}
		switch value.(type) {
		case string, bool:
		default:
			return &Error{Category: "transport"}
		}
	}
	for _, rule := range input.RetentionPolicy.Rules {
		if rule.Period != "items" && rule.Period != "minutes" && rule.Period != "hours" && rule.Period != "days" && rule.Period != "weeks" && rule.Period != "months" && rule.Period != "years" {
			return &Error{Category: "transport"}
		}
		if rule.Quantity <= 0 {
			return &Error{Category: "transport"}
		}
	}
	if month := input.IncrementalBackup.Month; month != nil {
		switch month.(type) {
		case string, []string:
		default:
			return &Error{Category: "transport"}
		}
	}
	// manual_incremental_backup has server-defined wildcard and comma-separated
	// grammar. Keep it opaque here so the provider does not reject forms that
	// the deployment accepts; the API remains authoritative for validation.
	return nil
}
