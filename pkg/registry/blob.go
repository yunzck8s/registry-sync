package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// BlobExists checks if a blob exists in the registry
func (c *Client) BlobExists(ctx context.Context, repository, digest string) (bool, int64, error) {
	path := fmt.Sprintf("/v2/%s/blobs/%s", repository, digest)

	resp, err := c.doRequest(ctx, "HEAD", path, nil, nil)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		size := int64(0)
		if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
			size, _ = strconv.ParseInt(contentLength, 10, 64)
		}
		return true, size, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, 0, nil
	}

	return false, 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// GetBlob downloads a blob from the registry
func (c *Client) GetBlob(ctx context.Context, repository, digest string) (io.ReadCloser, int64, error) {
	path := fmt.Sprintf("/v2/%s/blobs/%s", repository, digest)

	resp, err := c.doRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("failed to get blob: %d %s", resp.StatusCode, string(body))
	}

	size := int64(0)
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		size, _ = strconv.ParseInt(contentLength, 10, 64)
	}

	return resp.Body, size, nil
}

// PutBlob uploads a blob to the registry
func (c *Client) PutBlob(ctx context.Context, repository, digest string, content io.Reader, size int64) error {
	// Step 1: Initiate upload
	uploadURL, err := c.initiateUpload(ctx, repository, digest)
	if err != nil {
		return fmt.Errorf("failed to initiate upload: %w", err)
	}

	// Step 2: Upload content (returns new Location)
	newUploadURL, err := c.uploadContent(ctx, uploadURL, content, size)
	if err != nil {
		return fmt.Errorf("failed to upload content: %w", err)
	}

	// Step 3: Complete upload using the new Location
	if err := c.completeUpload(ctx, newUploadURL, digest); err != nil {
		return fmt.Errorf("failed to complete upload: %w", err)
	}

	return nil
}

// initiateUpload initiates a blob upload
func (c *Client) initiateUpload(ctx context.Context, repository, digest string) (string, error) {
	path := fmt.Sprintf("/v2/%s/blobs/uploads/", repository)

	// Try cross-repository mount first
	if digest != "" {
		path += "?mount=" + digest + "&from=" + repository
	}

	resp, err := c.doRequest(ctx, "POST", path, nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// If mount succeeded, blob already exists
	if resp.StatusCode == http.StatusCreated {
		return "", fmt.Errorf("blob already exists")
	}

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to initiate upload: %d %s", resp.StatusCode, string(body))
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("no location header in response")
	}

	// Handle relative URLs
	if !strings.HasPrefix(location, "http") {
		location = c.BaseURL + location
	}

	return location, nil
}

// uploadContent uploads blob content and returns the new upload URL
func (c *Client) uploadContent(ctx context.Context, uploadURL string, content io.Reader, size int64) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "PATCH", uploadURL, content)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	if size > 0 {
		req.Header.Set("Content-Length", strconv.FormatInt(size, 10))
		req.ContentLength = size
	}

	// Add authentication
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload content: %d %s", resp.StatusCode, string(body))
	}

	// Get the new Location for completing the upload
	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("no location header in PATCH response")
	}

	// Handle relative URLs
	if !strings.HasPrefix(location, "http") {
		location = c.BaseURL + location
	}

	return location, nil
}

// completeUpload completes a blob upload
func (c *Client) completeUpload(ctx context.Context, uploadURL, digest string) error {
	// Add digest to URL
	u, err := url.Parse(uploadURL)
	if err != nil {
		return err
	}

	q := u.Query()
	q.Set("digest", digest)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "PUT", u.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Length", "0")

	// Add authentication
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to complete upload: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

// MountBlob attempts to mount a blob from another repository
func (c *Client) MountBlob(ctx context.Context, fromRepo, toRepo, digest string) (bool, error) {
	path := fmt.Sprintf("/v2/%s/blobs/uploads/?mount=%s&from=%s", toRepo, digest, fromRepo)

	resp, err := c.doRequest(ctx, "POST", path, nil, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Mount succeeded
	if resp.StatusCode == http.StatusCreated {
		return true, nil
	}

	// Mount not supported or failed
	if resp.StatusCode == http.StatusAccepted {
		return false, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("failed to mount blob: %d %s", resp.StatusCode, string(body))
}

// CopyBlob copies a blob from source to target
func CopyBlob(ctx context.Context, source *Client, target *Client, sourceRepo, targetRepo, digest string, size int64) error {
	// Check if blob already exists in target
	exists, _, err := target.BlobExists(ctx, targetRepo, digest)
	if err != nil {
		return fmt.Errorf("failed to check blob existence: %w", err)
	}

	if exists {
		return nil // Already exists, skip
	}

	// Try to mount blob (if target supports cross-repo mount)
	mounted, err := target.MountBlob(ctx, targetRepo, targetRepo, digest)
	if err == nil && mounted {
		return nil // Successfully mounted
	}

	// Download from source
	reader, _, err := source.GetBlob(ctx, sourceRepo, digest)
	if err != nil {
		return fmt.Errorf("failed to download blob: %w", err)
	}
	defer reader.Close()

	// Upload to target
	if err := target.PutBlob(ctx, targetRepo, digest, reader, size); err != nil {
		return fmt.Errorf("failed to upload blob: %w", err)
	}

	return nil
}
