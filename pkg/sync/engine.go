package sync

import (
	"context"
	"fmt"
	"time"

	"registry-sync/pkg/config"
	"registry-sync/pkg/filter"
	"registry-sync/pkg/registry"
)

// Engine is the main synchronization engine
type Engine struct {
	config       *config.Config
	retryConfig  RetryConfig
	dryRun       bool
	progressFunc ProgressFunc
}

// ProgressFunc is called to report progress
type ProgressFunc func(info ProgressInfo)

// ProgressInfo contains synchronization progress information
type ProgressInfo struct {
	TaskName      string
	Repository    string
	Tag           string
	Phase         string // "manifest", "blob", "complete"
	TotalBlobs    int
	SyncedBlobs   int
	TotalSize     int64
	SyncedSize    int64
	CurrentBlob   string
	CurrentSize   int64
	Error         error
}

// NewEngine creates a new synchronization engine
func NewEngine(cfg *config.Config, dryRun bool) *Engine {
	return &Engine{
		config:  cfg,
		dryRun:  dryRun,
		retryConfig: RetryConfig{
			MaxAttempts:     cfg.Global.Retry.MaxAttempts,
			InitialInterval: cfg.Global.Retry.InitialInterval,
			MaxInterval:     cfg.Global.Retry.MaxInterval,
		},
	}
}

// SetProgressFunc sets the progress callback function
func (e *Engine) SetProgressFunc(fn ProgressFunc) {
	e.progressFunc = fn
}

// reportProgress reports progress if callback is set
func (e *Engine) reportProgress(info ProgressInfo) {
	if e.progressFunc != nil {
		e.progressFunc(info)
	}
}

// SyncAll synchronizes all enabled sync rules
func (e *Engine) SyncAll(ctx context.Context) error {
	rules := e.config.GetEnabledRules()
	if len(rules) == 0 {
		return fmt.Errorf("no enabled sync rules found")
	}

	fmt.Printf("Starting sync for %d rules...\n", len(rules))

	for _, rule := range rules {
		fmt.Printf("\n=== Syncing: %s ===\n", rule.Name)

		if err := e.SyncRule(ctx, rule); err != nil {
			fmt.Printf("❌ Failed to sync %s: %v\n", rule.Name, err)
			return err
		}

		fmt.Printf("✅ Successfully synced %s\n", rule.Name)
	}

	return nil
}

// SyncRule synchronizes a single sync rule
func (e *Engine) SyncRule(ctx context.Context, rule config.SyncRule) error {
	// Get source and target registries
	sourceReg, err := e.config.GetRegistry(rule.Source.Registry)
	if err != nil {
		return err
	}

	targetReg, err := e.config.GetRegistry(rule.Target.Registry)
	if err != nil {
		return err
	}

	// Create registry clients
	sourceClient := registry.NewClient(
		config.NormalizeRegistryURL(sourceReg.URL),
		sourceReg.Username,
		sourceReg.Password,
		sourceReg.Insecure,
		sourceReg.RateLimit.QPS,
	)

	targetClient := registry.NewClient(
		config.NormalizeRegistryURL(targetReg.URL),
		targetReg.Username,
		targetReg.Password,
		targetReg.Insecure,
		targetReg.RateLimit.QPS,
	)

	// Test connectivity
	if err := sourceClient.PingCheck(ctx); err != nil {
		return fmt.Errorf("failed to connect to source registry: %w", err)
	}

	if err := targetClient.PingCheck(ctx); err != nil {
		return fmt.Errorf("failed to connect to target registry: %w", err)
	}

	// List tags from source
	fmt.Println("Fetching tags from source...")
	tags, err := sourceClient.ListTags(ctx, rule.Source.Repository)
	if err != nil {
		return fmt.Errorf("failed to list tags: %w", err)
	}

	fmt.Printf("Found %d tags in source repository\n", len(tags))

	// Apply tag filters
	tagFilter, err := filter.NewFilter(rule.Tags.Include, rule.Tags.Exclude, rule.Tags.Latest)
	if err != nil {
		return fmt.Errorf("failed to create tag filter: %w", err)
	}

	// Convert tags to TagInfo (with current time as placeholder)
	tagInfos := make([]filter.TagInfo, len(tags))
	for i, tag := range tags {
		tagInfos[i] = filter.TagInfo{
			Name:    tag,
			Updated: time.Now(), // TODO: Get actual timestamp from registry
		}
	}

	filteredTags := tagFilter.FilterTags(tagInfos)
	fmt.Printf("After filtering: %d tags to sync\n", len(filteredTags))

	if len(filteredTags) == 0 {
		fmt.Println("No tags to sync")
		return nil
	}

	if e.dryRun {
		fmt.Println("\n[DRY RUN] Would sync the following tags:")
		for _, tag := range filteredTags {
			fmt.Printf("  - %s\n", tag)
		}
		return nil
	}

	// Sync each tag
	for i, tag := range filteredTags {
		fmt.Printf("\n[%d/%d] Syncing tag: %s\n", i+1, len(filteredTags), tag)

		if err := e.SyncTag(ctx, sourceClient, targetClient, rule, tag); err != nil {
			return fmt.Errorf("failed to sync tag %s: %w", tag, err)
		}
	}

	return nil
}

// SyncTag synchronizes a single tag
func (e *Engine) SyncTag(ctx context.Context, source, target *registry.Client, rule config.SyncRule, tag string) error {
	e.reportProgress(ProgressInfo{
		TaskName:   rule.Name,
		Repository: rule.Source.Repository,
		Tag:        tag,
		Phase:      "manifest",
	})

	// Get manifest from source
	manifest, err := source.GetManifest(ctx, rule.Source.Repository, tag)
	if err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}

	// Handle manifest list (multi-arch)
	if manifest.IsManifestList() {
		return e.SyncManifestList(ctx, source, target, rule, tag, manifest)
	}

	// Sync single manifest
	return e.SyncSingleManifest(ctx, source, target, rule, tag, manifest)
}

// SyncManifestList synchronizes a manifest list (multi-arch)
func (e *Engine) SyncManifestList(ctx context.Context, source, target *registry.Client, rule config.SyncRule, tag string, manifestList *registry.Manifest) error {
	fmt.Println("  Detected manifest list (multi-arch)")

	// Filter by architecture
	entries := registry.FilterManifestsByArch(manifestList.Manifests, rule.Architectures)
	fmt.Printf("  Architectures to sync: %d\n", len(entries))

	// Sync each architecture
	for _, entry := range entries {
		fmt.Printf("  Syncing architecture: %s/%s\n", entry.Platform.OS, entry.Platform.Architecture)

		// Get the actual manifest for this architecture
		archManifest, err := source.GetManifest(ctx, rule.Source.Repository, entry.Digest)
		if err != nil {
			return fmt.Errorf("failed to get manifest for %s: %w", entry.Digest, err)
		}

		// Sync this manifest
		if err := e.SyncSingleManifest(ctx, source, target, rule, entry.Digest, archManifest); err != nil {
			return err
		}
	}

	// Upload the manifest list to target
	fmt.Println("  Uploading manifest list...")
	if _, err := target.PutManifest(ctx, rule.Target.Repository, tag, manifestList); err != nil {
		return fmt.Errorf("failed to upload manifest list: %w", err)
	}

	return nil
}

// SyncSingleManifest synchronizes a single manifest
func (e *Engine) SyncSingleManifest(ctx context.Context, source, target *registry.Client, rule config.SyncRule, reference string, manifest *registry.Manifest) error {
	// Get all blobs from manifest
	blobs := manifest.GetAllBlobs()
	fmt.Printf("  Found %d blobs to sync\n", len(blobs))

	e.reportProgress(ProgressInfo{
		TaskName:    rule.Name,
		Repository:  rule.Source.Repository,
		Tag:         reference,
		Phase:       "blob",
		TotalBlobs:  len(blobs),
		SyncedBlobs: 0,
	})

	// Create worker pool for concurrent blob sync
	pool := NewWorkerPool(ctx, e.config.Global.Concurrency)
	pool.Start()

	// Submit blob sync tasks
	for _, blob := range blobs {
		task := &BlobSyncTask{
			Source:      source,
			Target:      target,
			SourceRepo:  rule.Source.Repository,
			TargetRepo:  rule.Target.Repository,
			Digest:      blob.Digest,
			Size:        blob.Size,
			RetryConfig: e.retryConfig,
			OnProgress: func(digest string, size int64) {
				e.reportProgress(ProgressInfo{
					TaskName:    rule.Name,
					Repository:  rule.Source.Repository,
					Tag:         reference,
					Phase:       "blob",
					CurrentBlob: digest,
					CurrentSize: size,
				})
			},
		}

		if err := pool.Submit(task); err != nil {
			pool.Stop()
			return err
		}
	}

	// Wait for all blobs to sync
	if err := pool.Wait(); err != nil {
		return fmt.Errorf("blob sync failed: %w", err)
	}

	fmt.Println("  All blobs synced successfully")

	// Upload manifest to target
	fmt.Println("  Uploading manifest...")
	if _, err := target.PutManifest(ctx, rule.Target.Repository, reference, manifest); err != nil {
		return fmt.Errorf("failed to upload manifest: %w", err)
	}

	e.reportProgress(ProgressInfo{
		TaskName:   rule.Name,
		Repository: rule.Source.Repository,
		Tag:        reference,
		Phase:      "complete",
	})

	return nil
}

// BlobSyncTask represents a blob synchronization task
type BlobSyncTask struct {
	Source      *registry.Client
	Target      *registry.Client
	SourceRepo  string
	TargetRepo  string
	Digest      string
	Size        int64
	RetryConfig RetryConfig
	OnProgress  func(digest string, size int64)
}

// Execute executes the blob sync task
func (t *BlobSyncTask) Execute(ctx context.Context) error {
	// Check if blob already exists in target
	exists, _, err := t.Target.BlobExists(ctx, t.TargetRepo, t.Digest)
	if err != nil {
		return fmt.Errorf("failed to check blob existence: %w", err)
	}

	if exists {
		fmt.Printf("  ⏩ Blob already exists: %s\n", t.Digest[:12])
		if t.OnProgress != nil {
			t.OnProgress(t.Digest, t.Size)
		}
		return nil
	}

	fmt.Printf("  ⬇️  Syncing blob: %s (%.2f MB)\n", t.Digest[:12], float64(t.Size)/(1024*1024))

	// Copy blob with retry
	err = RetryWithBackoff(ctx, t.RetryConfig, func() error {
		return registry.CopyBlob(ctx, t.Source, t.Target, t.SourceRepo, t.TargetRepo, t.Digest, t.Size)
	})

	if err != nil {
		return fmt.Errorf("failed to copy blob %s: %w", t.Digest[:12], err)
	}

	fmt.Printf("  ✅ Blob synced: %s\n", t.Digest[:12])

	if t.OnProgress != nil {
		t.OnProgress(t.Digest, t.Size)
	}

	return nil
}

// Description returns a description of the task
func (t *BlobSyncTask) Description() string {
	return fmt.Sprintf("sync blob %s", t.Digest[:12])
}
