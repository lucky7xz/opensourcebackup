package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

const defaultTimeout = 30 * time.Second

// Client communicates with the OpensourceBackup control plane.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New creates a Client pointing at baseURL.
// apiKey is sent as X-API-Key header (placeholder until mTLS in B9).
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// ListPendingJobs returns all pending jobs for the given system.
func (c *Client) ListPendingJobs(ctx context.Context, systemID uuid.UUID) ([]catalog.BackupJob, error) {
	url := fmt.Sprintf("%s/v1/jobs?system_id=%s&status=pending", c.baseURL, systemID)
	var jobs []catalog.BackupJob
	if err := c.get(ctx, url, &jobs); err != nil {
		return nil, fmt.Errorf("list pending jobs: %w", err)
	}
	return jobs, nil
}

// GetPolicy returns the policy with the given ID.
func (c *Client) GetPolicy(ctx context.Context, id uuid.UUID) (*catalog.BackupPolicy, error) {
	url := fmt.Sprintf("%s/v1/policies/%s", c.baseURL, id)
	var p catalog.BackupPolicy
	if err := c.get(ctx, url, &p); err != nil {
		return nil, fmt.Errorf("get policy %s: %w", id, err)
	}
	return &p, nil
}

// UpdateJobStatus updates the job's status and optional metrics.
func (c *Client) UpdateJobStatus(ctx context.Context, job *catalog.BackupJob) error {
	url := fmt.Sprintf("%s/v1/jobs/%s", c.baseURL, job.ID)
	if err := c.put(ctx, url, job); err != nil {
		return fmt.Errorf("update job %s: %w", job.ID, err)
	}
	return nil
}

// CreateSnapshot registers a completed snapshot with the control plane.
func (c *Client) CreateSnapshot(ctx context.Context, s *catalog.Snapshot) error {
	if err := c.post(ctx, c.baseURL+"/v1/snapshots", s, s); err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}
	return nil
}

func (c *Client) get(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // HTTP response body close errors are not actionable
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) put(ctx context.Context, url string, body any) error {
	return c.sendJSON(ctx, http.MethodPut, url, body, nil)
}

func (c *Client) post(ctx context.Context, url string, body any, out any) error {
	return c.sendJSON(ctx, http.MethodPost, url, body, out)
}

func (c *Client) sendJSON(ctx context.Context, method, url string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // HTTP response body close errors are not actionable
	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
}
