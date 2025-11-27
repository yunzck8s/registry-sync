package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Manifest represents a Docker manifest
type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType"`
	Config        Descriptor        `json:"config"`
	Layers        []Descriptor      `json:"layers"`
	Manifests     []ManifestEntry   `json:"manifests,omitempty"` // For manifest lists
	Raw           []byte            `json:"-"`
	ContentDigest string            `json:"-"`
}

// Descriptor represents a content descriptor
type Descriptor struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
	Platform  *Platform `json:"platform,omitempty"`
}

// Platform represents a platform specification
type Platform struct {
	Architecture string   `json:"architecture"`
	OS           string   `json:"os"`
	OSVersion    string   `json:"os.version,omitempty"`
	OSFeatures   []string `json:"os.features,omitempty"`
	Variant      string   `json:"variant,omitempty"`
}

// ManifestEntry represents an entry in a manifest list
type ManifestEntry struct {
	MediaType string    `json:"mediaType"`
	Size      int64     `json:"size"`
	Digest    string    `json:"digest"`
	Platform  Platform  `json:"platform"`
}

// GetManifest retrieves a manifest from the registry
func (c *Client) GetManifest(ctx context.Context, repository, reference string) (*Manifest, error) {
	path := fmt.Sprintf("/v2/%s/manifests/%s", repository, reference)

	headers := map[string]string{
		"Accept": buildAcceptHeader(),
	}

	resp, err := c.doRequest(ctx, "GET", path, nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get manifest: %d %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	manifest.Raw = data
	manifest.ContentDigest = resp.Header.Get("Docker-Content-Digest")

	return &manifest, nil
}

// PutManifest uploads a manifest to the registry
func (c *Client) PutManifest(ctx context.Context, repository, reference string, manifest *Manifest) (string, error) {
	path := fmt.Sprintf("/v2/%s/manifests/%s", repository, reference)

	headers := map[string]string{
		"Content-Type": manifest.MediaType,
	}

	resp, err := c.doRequest(ctx, "PUT", path, strings.NewReader(string(manifest.Raw)), headers)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to put manifest: %d %s", resp.StatusCode, string(body))
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		digest = resp.Header.Get("Location")
	}

	return digest, nil
}

// HeadManifest checks if a manifest exists
func (c *Client) HeadManifest(ctx context.Context, repository, reference string) (bool, string, error) {
	path := fmt.Sprintf("/v2/%s/manifests/%s", repository, reference)

	headers := map[string]string{
		"Accept": buildAcceptHeader(),
	}

	resp, err := c.doRequest(ctx, "HEAD", path, nil, headers)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		digest := resp.Header.Get("Docker-Content-Digest")
		return true, digest, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, "", nil
	}

	return false, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// IsManifestList checks if a manifest is a manifest list
func (m *Manifest) IsManifestList() bool {
	return strings.Contains(m.MediaType, "manifest.list") ||
		strings.Contains(m.MediaType, "image.index")
}

// GetAllBlobs returns all blobs referenced in the manifest
func (m *Manifest) GetAllBlobs() []Descriptor {
	var blobs []Descriptor

	// Add config blob (if not a manifest list)
	if m.Config.Digest != "" {
		blobs = append(blobs, m.Config)
	}

	// Add layer blobs
	blobs = append(blobs, m.Layers...)

	return blobs
}

// FilterManifestsByArch filters manifest list entries by architecture
func FilterManifestsByArch(manifests []ManifestEntry, architectures []string) []ManifestEntry {
	if len(architectures) == 0 {
		return manifests
	}

	archMap := make(map[string]bool)
	for _, arch := range architectures {
		archMap[arch] = true
	}

	var filtered []ManifestEntry
	for _, m := range manifests {
		if archMap[m.Platform.Architecture] {
			filtered = append(filtered, m)
		}
	}

	return filtered
}

// ListTags lists all tags for a repository
func (c *Client) ListTags(ctx context.Context, repository string) ([]string, error) {
	path := fmt.Sprintf("/v2/%s/tags/list", repository)

	resp, err := c.doRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list tags: %d %s", resp.StatusCode, string(body))
	}

	var result struct {
		Tags []string `json:"tags"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Tags, nil
}
