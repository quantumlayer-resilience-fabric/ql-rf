// Package servicenow provides integration with ServiceNow ITSM platform.
// This enables:
// - Creating change requests for AI-planned operations
// - Creating incidents for failed executions
// - Syncing asset data between QL-RF and ServiceNow CMDB
// - Linking AI tasks to ServiceNow tickets for audit trail
package servicenow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client provides methods for interacting with ServiceNow APIs.
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// Config holds ServiceNow connection settings.
type Config struct {
	InstanceURL string // e.g., "https://mycompany.service-now.com"
	Username    string // ServiceNow API user
	Password    string // ServiceNow API password/token
	Timeout     time.Duration
}

// NewClient creates a new ServiceNow client.
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL:  cfg.InstanceURL,
		username: cfg.Username,
		password: cfg.Password,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ChangeRequest represents a ServiceNow change request.
type ChangeRequest struct {
	SysID           string `json:"sys_id,omitempty"`
	Number          string `json:"number,omitempty"`
	ShortDesc       string `json:"short_description"`
	Description     string `json:"description,omitempty"`
	Type            string `json:"type,omitempty"`          // normal, standard, emergency
	Category        string `json:"category,omitempty"`
	Priority        int    `json:"priority,omitempty"`      // 1-4 (1=Critical, 4=Low)
	Risk            string `json:"risk,omitempty"`          // high, moderate, low
	Impact          string `json:"impact,omitempty"`        // 1-3 (1=High, 3=Low)
	AssignmentGroup string `json:"assignment_group,omitempty"`
	AssignedTo      string `json:"assigned_to,omitempty"`
	State           string `json:"state,omitempty"`
	ClosedCode      string `json:"close_code,omitempty"`
	ClosedNotes     string `json:"close_notes,omitempty"`
	StartDate       string `json:"start_date,omitempty"`
	EndDate         string `json:"end_date,omitempty"`
	WorkNotes       string `json:"work_notes,omitempty"`
	QLRFID          string `json:"u_qlrf_task_id,omitempty"` // Custom field for QL-RF task ID
}

// Incident represents a ServiceNow incident.
type Incident struct {
	SysID           string `json:"sys_id,omitempty"`
	Number          string `json:"number,omitempty"`
	ShortDesc       string `json:"short_description"`
	Description     string `json:"description,omitempty"`
	Category        string `json:"category,omitempty"`
	Subcategory     string `json:"subcategory,omitempty"`
	Priority        int    `json:"priority,omitempty"` // 1-5 (1=Critical)
	Impact          string `json:"impact,omitempty"`   // 1-3
	Urgency         string `json:"urgency,omitempty"`  // 1-3
	AssignmentGroup string `json:"assignment_group,omitempty"`
	AssignedTo      string `json:"assigned_to,omitempty"`
	State           string `json:"state,omitempty"`     // 1=New, 2=In Progress, 6=Resolved, 7=Closed
	CIAffected      string `json:"cmdb_ci,omitempty"`   // Configuration Item
	QLRFID          string `json:"u_qlrf_task_id,omitempty"`
}

// CMDBConfigurationItem represents a ServiceNow CMDB CI.
type CMDBConfigurationItem struct {
	SysID        string `json:"sys_id,omitempty"`
	Name         string `json:"name"`
	Class        string `json:"sys_class_name,omitempty"`
	AssetTag     string `json:"asset_tag,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
	Environment  string `json:"u_environment,omitempty"` // Custom field
	Platform     string `json:"u_platform,omitempty"`    // Custom field (aws, azure, gcp, vsphere)
	Region       string `json:"u_region,omitempty"`      // Custom field
	ImageVersion string `json:"u_image_version,omitempty"`
	DriftStatus  string `json:"u_drift_status,omitempty"` // Custom field (compliant, drifted)
	QLRFID       string `json:"u_qlrf_asset_id,omitempty"`
}

// CreateChangeRequest creates a new change request in ServiceNow.
func (c *Client) CreateChangeRequest(ctx context.Context, cr ChangeRequest) (*ChangeRequest, error) {
	endpoint := "/api/now/table/change_request"

	body, err := json.Marshal(cr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal change request: %w", err)
	}

	respBody, err := c.doRequest(ctx, "POST", endpoint, body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result ChangeRequest `json:"result"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Result, nil
}

// UpdateChangeRequest updates an existing change request.
func (c *Client) UpdateChangeRequest(ctx context.Context, sysID string, cr ChangeRequest) (*ChangeRequest, error) {
	endpoint := fmt.Sprintf("/api/now/table/change_request/%s", sysID)

	body, err := json.Marshal(cr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal change request: %w", err)
	}

	respBody, err := c.doRequest(ctx, "PATCH", endpoint, body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result ChangeRequest `json:"result"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Result, nil
}

// GetChangeRequest retrieves a change request by sys_id.
func (c *Client) GetChangeRequest(ctx context.Context, sysID string) (*ChangeRequest, error) {
	endpoint := fmt.Sprintf("/api/now/table/change_request/%s", sysID)

	respBody, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result ChangeRequest `json:"result"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Result, nil
}

// CloseChangeRequest closes a change request with the given state and notes.
func (c *Client) CloseChangeRequest(ctx context.Context, sysID, closeCode, closeNotes string) error {
	cr := ChangeRequest{
		State:       "3", // Closed
		ClosedCode:  closeCode,
		ClosedNotes: closeNotes,
	}
	_, err := c.UpdateChangeRequest(ctx, sysID, cr)
	return err
}

// CreateIncident creates a new incident in ServiceNow.
func (c *Client) CreateIncident(ctx context.Context, inc Incident) (*Incident, error) {
	endpoint := "/api/now/table/incident"

	body, err := json.Marshal(inc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal incident: %w", err)
	}

	respBody, err := c.doRequest(ctx, "POST", endpoint, body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result Incident `json:"result"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Result, nil
}

// UpdateIncident updates an existing incident.
func (c *Client) UpdateIncident(ctx context.Context, sysID string, inc Incident) (*Incident, error) {
	endpoint := fmt.Sprintf("/api/now/table/incident/%s", sysID)

	body, err := json.Marshal(inc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal incident: %w", err)
	}

	respBody, err := c.doRequest(ctx, "PATCH", endpoint, body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result Incident `json:"result"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Result, nil
}

// GetIncident retrieves an incident by sys_id.
func (c *Client) GetIncident(ctx context.Context, sysID string) (*Incident, error) {
	endpoint := fmt.Sprintf("/api/now/table/incident/%s", sysID)

	respBody, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result Incident `json:"result"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Result, nil
}

// UpsertCMDBCI creates or updates a Configuration Item in the CMDB.
func (c *Client) UpsertCMDBCI(ctx context.Context, ci CMDBConfigurationItem) (*CMDBConfigurationItem, error) {
	// Try to find existing CI by QL-RF ID
	existing, err := c.FindCMDBCIByQLRFID(ctx, ci.QLRFID)
	if err == nil && existing != nil {
		// Update existing
		return c.UpdateCMDBCI(ctx, existing.SysID, ci)
	}

	// Create new
	return c.CreateCMDBCI(ctx, ci)
}

// CreateCMDBCI creates a new Configuration Item.
func (c *Client) CreateCMDBCI(ctx context.Context, ci CMDBConfigurationItem) (*CMDBConfigurationItem, error) {
	// Default to cmdb_ci_server if no class specified
	tableName := "cmdb_ci_server"
	if ci.Class != "" {
		tableName = ci.Class
	}

	endpoint := fmt.Sprintf("/api/now/table/%s", tableName)

	body, err := json.Marshal(ci)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CI: %w", err)
	}

	respBody, err := c.doRequest(ctx, "POST", endpoint, body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result CMDBConfigurationItem `json:"result"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Result, nil
}

// UpdateCMDBCI updates an existing Configuration Item.
func (c *Client) UpdateCMDBCI(ctx context.Context, sysID string, ci CMDBConfigurationItem) (*CMDBConfigurationItem, error) {
	tableName := "cmdb_ci_server"
	if ci.Class != "" {
		tableName = ci.Class
	}

	endpoint := fmt.Sprintf("/api/now/table/%s/%s", tableName, sysID)

	body, err := json.Marshal(ci)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CI: %w", err)
	}

	respBody, err := c.doRequest(ctx, "PATCH", endpoint, body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result CMDBConfigurationItem `json:"result"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Result, nil
}

// FindCMDBCIByQLRFID finds a CI by its QL-RF asset ID.
func (c *Client) FindCMDBCIByQLRFID(ctx context.Context, qlrfID string) (*CMDBConfigurationItem, error) {
	endpoint := fmt.Sprintf("/api/now/table/cmdb_ci_server?sysparm_query=u_qlrf_asset_id=%s&sysparm_limit=1", url.QueryEscape(qlrfID))

	respBody, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result []CMDBConfigurationItem `json:"result"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Result) == 0 {
		return nil, fmt.Errorf("CI not found")
	}

	return &response.Result[0], nil
}

// doRequest performs an HTTP request to ServiceNow API.
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body []byte) ([]byte, error) {
	fullURL := c.baseURL + endpoint

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ServiceNow API error: %d - %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
