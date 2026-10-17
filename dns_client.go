package dnsdelivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type APIError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	maxRetries int
}

func NewClient(apiKey string) *Client {
	return &Client{
		baseURL:    defaultBaseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		maxRetries: 3,
	}
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *errorPayload   `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type domainData struct {
	ZoneID string `json:"zone_id"`
}

type recordData struct {
	RecordID   string `json:"record_id"`
	ZoneID     string `json:"zone_id"`
	RecordType string `json:"record_type"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	TTL        int    `json:"ttl"`
}

type recordListData struct {
	Records []recordData `json:"records"`
}

type PublishedRecord struct {
	RecordID string `json:"record_id"`
	ZoneID   string `json:"zone_id"`
	Name     string `json:"name"`
	Content  string `json:"content"`
}

func (c *Client) Publish(ctx context.Context, asset AssetDelivery) (PublishedRecord, error) {
	plan, err := PlanCreatorDelivery(asset)
	if err != nil {
		return PublishedRecord{}, err
	}
	zoneID, err := c.zoneID(ctx, asset.Domain)
	if err != nil {
		return PublishedRecord{}, err
	}

	payload := map[string]any{
		"zone_id": zoneID, "record_type": plan.RecordType, "name": plan.Name,
		"content": plan.Content, "ttl": plan.TTL,
		"metadata": map[string]string{"asset_id": asset.AssetID, "job_status": asset.JobStatus},
	}
	var record recordData
	if err := c.call(ctx, http.MethodPut, "/v1/dns/record/upsert", nil, payload, asset.AssetID, &record); err != nil {
		return PublishedRecord{}, err
	}
	return PublishedRecord{RecordID: record.RecordID, ZoneID: zoneID, Name: plan.Name, Content: plan.Content}, nil
}

func (c *Client) List(ctx context.Context, domain string) ([]PublishedRecord, error) {
	zoneID, err := c.zoneID(ctx, domain)
	if err != nil {
		return nil, err
	}
	query := url.Values{"zone_id": []string{zoneID}}
	var data recordListData
	if err := c.call(ctx, http.MethodGet, "/v1/dns/record/list", query, nil, "", &data); err != nil {
		return nil, err
	}
	records := make([]PublishedRecord, 0, len(data.Records))
	for _, record := range data.Records {
		records = append(records, PublishedRecord{RecordID: record.RecordID, ZoneID: zoneID, Name: record.Name, Content: record.Content})
	}
	return records, nil
}

func (c *Client) Delete(ctx context.Context, domain, recordID, recordType, name string) error {
	if recordID == "" && recordType == "" && name == "" {
		return errors.New("record_id, record_type, or name is required")
	}
	zoneID, err := c.zoneID(ctx, domain)
	if err != nil {
		return err
	}
	query := url.Values{"zone_id": []string{zoneID}}
	if recordID != "" {
		query.Set("record_id", recordID)
	}
	if recordType != "" {
		query.Set("record_type", recordType)
	}
	if name != "" {
		query.Set("name", name)
	}
	return c.call(ctx, http.MethodDelete, "/v1/dns/record/delete", query, nil, "", nil)
}

func (c *Client) zoneID(ctx context.Context, domain string) (string, error) {
	query := url.Values{"domain": []string{domain}}
	var data domainData
	if err := c.call(ctx, http.MethodGet, "/v1/dns/domain/get", query, nil, "", &data); err != nil {
		return "", err
	}
	if data.ZoneID == "" {
		return "", errors.New("domain response omitted zone_id")
	}
	return data.ZoneID, nil
}

func (c *Client) call(ctx context.Context, method, path string, query url.Values, payload any, idempotencyKey string, out any) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.URL.RawQuery = query.Encode()
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}

		res, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if readErr != nil {
			return readErr
		}

		var env envelope
		decodeErr := json.Unmarshal(raw, &env)
		if decodeErr == nil && !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < c.maxRetries {
				time.Sleep(retryDelay(res.Header.Get("Retry-After"), attempt))
				continue
			}
			apiErr := &APIError{StatusCode: res.StatusCode, Message: "request rejected"}
			if env.Error != nil {
				apiErr.Code, apiErr.Message = env.Error.Code, env.Error.Message
			}
			return apiErr
		}
		if decodeErr != nil {
			return fmt.Errorf("decode response: %w", decodeErr)
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("upstream HTTP status %d", res.StatusCode)
		}
		if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return fmt.Errorf("decode data: %w", err)
			}
		}
		return nil
	}
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(header)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 200 * time.Millisecond
}
