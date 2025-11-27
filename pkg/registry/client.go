package registry

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"registry-sync/pkg/ratelimit"
)

// Client represents a Docker Registry V2 API client
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Username   string
	Password   string
	Token      string
	Limiter    *ratelimit.Limiter
}

// NewClient creates a new registry client
func NewClient(baseURL, username, password string, insecure bool, qps int) *Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Username:   username,
		Password:   password,
		Limiter:    ratelimit.NewLimiter(qps),
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   300 * time.Second, // 增加到5分钟，处理慢速Registry
		},
	}
}

// PingCheck checks if the registry is accessible
func (c *Client) PingCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/v2/", nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to ping registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		return nil
	}

	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// doRequest performs an HTTP request with authentication and rate limiting
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	// Apply rate limiting
	if err := c.Limiter.Wait(ctx); err != nil {
		return nil, err
	}

	fullURL := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, err
	}

	// Set headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Add basic auth if available
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	// Try with auth
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	// If unauthorized, try to authenticate
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		// Parse WWW-Authenticate header
		authHeader := resp.Header.Get("WWW-Authenticate")
		if authHeader == "" {
			// Try basic auth
			req, _ = http.NewRequestWithContext(ctx, method, fullURL, body)
			for k, v := range headers {
				req.Header.Set(k, v)
			}
			req.SetBasicAuth(c.Username, c.Password)
			return c.HTTPClient.Do(req)
		}

		// Try bearer token auth
		token, err := c.getBearerToken(ctx, authHeader, path)
		if err != nil {
			return nil, fmt.Errorf("failed to get bearer token: %w", err)
		}

		// Retry with token
		req, _ = http.NewRequestWithContext(ctx, method, fullURL, body)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return c.HTTPClient.Do(req)
	}

	return resp, nil
}

// getBearerToken obtains a bearer token from the auth server
func (c *Client) getBearerToken(ctx context.Context, authHeader, requestPath string) (string, error) {
	// Parse WWW-Authenticate header
	// Format: Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/nginx:pull"
	params := parseAuthHeader(authHeader)

	realm := params["realm"]
	if realm == "" {
		return "", fmt.Errorf("no realm in WWW-Authenticate header")
	}

	// Build token request URL
	tokenURL, err := url.Parse(realm)
	if err != nil {
		return "", err
	}

	q := tokenURL.Query()
	if service := params["service"]; service != "" {
		q.Set("service", service)
	}
	if scope := params["scope"]; scope != "" {
		q.Set("scope", scope)
	}
	tokenURL.RawQuery = q.Encode()

	// Request token
	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL.String(), nil)
	if err != nil {
		return "", err
	}

	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed: %d %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	if tokenResp.Token != "" {
		return tokenResp.Token, nil
	}
	return tokenResp.AccessToken, nil
}

// parseAuthHeader parses WWW-Authenticate header
func parseAuthHeader(header string) map[string]string {
	params := make(map[string]string)

	// Remove "Bearer " prefix
	header = strings.TrimPrefix(header, "Bearer ")

	// Split by comma
	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Split by =
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.Trim(strings.TrimSpace(kv[1]), "\"")
			params[key] = value
		}
	}

	return params
}

// GetManifestMediaType returns the appropriate media type for manifest requests
func GetManifestMediaType() []string {
	return []string{
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.oci.image.index.v1+json",
	}
}

// buildAcceptHeader builds the Accept header for manifest requests
func buildAcceptHeader() string {
	return strings.Join(GetManifestMediaType(), ",")
}

// HarborProject represents a Harbor project
type HarborProject struct {
	Name      string `json:"name"`
	ProjectID int    `json:"project_id"`
	Public    bool   `json:"public"`
}

// HarborRepository represents a Harbor repository
type HarborRepository struct {
	Name         string `json:"name"`
	ProjectID    int    `json:"project_id"`
	ArtifactCount int   `json:"artifact_count"`
}

// ListProjects lists all projects from Harbor
// For Harbor: uses /api/v2.0/projects
// For Docker Hub: returns namespace
// For standard registry: extracts from catalog
func (c *Client) ListProjects(ctx context.Context) ([]string, error) {
	// Try Harbor API first
	projects, err := c.listHarborProjects(ctx)
	if err == nil {
		return projects, nil
	}

	// Fallback: use catalog and extract projects
	return c.listProjectsFromCatalog(ctx)
}

// listHarborProjects lists projects using Harbor API with pagination
func (c *Client) listHarborProjects(ctx context.Context) ([]string, error) {
	var allProjects []string
	page := 1
	pageSize := 100

	for {
		// Harbor API endpoint with pagination
		apiPath := fmt.Sprintf("/api/v2.0/projects?page=%d&page_size=%d", page, pageSize)

		req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+apiPath, nil)
		if err != nil {
			return nil, err
		}

		// Add basic auth for Harbor
		if c.Username != "" {
			req.SetBasicAuth(c.Username, c.Password)
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("harbor API returned status %d", resp.StatusCode)
		}

		var harborProjects []HarborProject
		if err := json.NewDecoder(resp.Body).Decode(&harborProjects); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		// Add projects from this page
		for _, p := range harborProjects {
			allProjects = append(allProjects, p.Name)
		}

		// If we got less than pageSize, we're on the last page
		if len(harborProjects) < pageSize {
			break
		}

		page++
	}

	return allProjects, nil
}

// listProjectsFromCatalog extracts projects from catalog (fallback)
func (c *Client) listProjectsFromCatalog(ctx context.Context) ([]string, error) {
	resp, err := c.doRequest(ctx, "GET", "/v2/_catalog", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list catalog: status %d", resp.StatusCode)
	}

	var catalog struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return nil, err
	}

	// Extract unique projects (part before first /)
	projectSet := make(map[string]bool)
	for _, repo := range catalog.Repositories {
		parts := strings.SplitN(repo, "/", 2)
		if len(parts) > 0 {
			projectSet[parts[0]] = true
		}
	}

	projects := make([]string, 0, len(projectSet))
	for project := range projectSet {
		projects = append(projects, project)
	}

	return projects, nil
}

// ListRepositories lists all repositories in a project
// For Harbor: uses /api/v2.0/projects/:project/repositories
// For others: filters catalog by project prefix
func (c *Client) ListRepositories(ctx context.Context, project string) ([]string, error) {
	// Try Harbor API first
	repos, err := c.listHarborRepositories(ctx, project)
	if err == nil {
		return repos, nil
	}

	// Fallback: filter catalog
	return c.listRepositoriesFromCatalog(ctx, project)
}

// listHarborRepositories lists repositories using Harbor API with pagination
func (c *Client) listHarborRepositories(ctx context.Context, project string) ([]string, error) {
	var allRepos []string
	page := 1
	pageSize := 100

	for {
		apiPath := fmt.Sprintf("/api/v2.0/projects/%s/repositories?page=%d&page_size=%d", project, page, pageSize)

		req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+apiPath, nil)
		if err != nil {
			return nil, err
		}

		if c.Username != "" {
			req.SetBasicAuth(c.Username, c.Password)
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("harbor API returned status %d", resp.StatusCode)
		}

		var harborRepos []HarborRepository
		if err := json.NewDecoder(resp.Body).Decode(&harborRepos); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		// Add repos from this page
		for _, r := range harborRepos {
			// Harbor returns full path like "project/repo", extract just repo name
			parts := strings.SplitN(r.Name, "/", 2)
			if len(parts) == 2 {
				allRepos = append(allRepos, parts[1])
			} else {
				allRepos = append(allRepos, r.Name)
			}
		}

		// If we got less than pageSize, we're on the last page
		if len(harborRepos) < pageSize {
			break
		}

		page++
	}

	return allRepos, nil
}

// listRepositoriesFromCatalog filters repositories by project prefix (fallback)
func (c *Client) listRepositoriesFromCatalog(ctx context.Context, project string) ([]string, error) {
	resp, err := c.doRequest(ctx, "GET", "/v2/_catalog", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list catalog: status %d", resp.StatusCode)
	}

	var catalog struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return nil, err
	}

	// Filter by project prefix
	prefix := project + "/"
	repos := []string{}
	for _, repo := range catalog.Repositories {
		if strings.HasPrefix(repo, prefix) {
			// Extract repo name (part after project/)
			repoName := strings.TrimPrefix(repo, prefix)
			repos = append(repos, repoName)
		}
	}

	return repos, nil
}

// CreateProject creates a project in Harbor
// For Harbor: uses /api/v2.0/projects
// For other registries: returns nil (not supported)
func (c *Client) CreateProject(ctx context.Context, projectName string, public bool) error {
	// Try Harbor API
	return c.createHarborProject(ctx, projectName, public)
}

// createHarborProject creates a project using Harbor API
func (c *Client) createHarborProject(ctx context.Context, projectName string, public bool) error {
	apiPath := "/api/v2.0/projects"

	// Prepare request body
	projectReq := map[string]interface{}{
		"project_name": projectName,
		"metadata": map[string]string{
			"public": fmt.Sprintf("%t", public),
		},
	}

	bodyBytes, err := json.Marshal(projectReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+apiPath, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Username != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		return nil
	}

	// If project already exists, it's not an error
	if resp.StatusCode == http.StatusConflict {
		return nil
	}

	// Read error response
	bodyBytes, _ = io.ReadAll(resp.Body)
	return fmt.Errorf("failed to create project (status %d): %s", resp.StatusCode, string(bodyBytes))
}

// ProjectExists checks if a project exists in Harbor
func (c *Client) ProjectExists(ctx context.Context, projectName string) (bool, error) {
	projects, err := c.ListProjects(ctx)
	if err != nil {
		return false, err
	}

	for _, p := range projects {
		if p == projectName {
			return true, nil
		}
	}

	return false, nil
}
