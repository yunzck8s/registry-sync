package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"registry-sync/pkg/config"
	"registry-sync/pkg/sync"
)

const version = "1.0.0"

func main() {
	// Define CLI flags
	var (
		configFile = flag.String("config", "configs/sync.yaml", "Path to configuration file")
		taskName   = flag.String("task", "", "Sync only specific task (by name)")
		dryRun     = flag.Bool("dry-run", false, "Dry run mode (don't actually sync)")
		showVer    = flag.Bool("version", false, "Show version and exit")
		validate   = flag.Bool("validate", false, "Validate configuration and exit")
	)

	flag.Parse()

	// Show version
	if *showVer {
		fmt.Printf("registry-sync version %s\n", version)
		os.Exit(0)
	}

	// Load configuration
	fmt.Printf("Loading configuration from: %s\n", *configFile)
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Validate only
	if *validate {
		fmt.Println("✅ Configuration is valid")
		os.Exit(0)
	}

	// Print configuration summary
	printConfigSummary(cfg)

	// Filter rules if task name is specified
	if *taskName != "" {
		found := false
		var filtered []config.SyncRule
		for _, rule := range cfg.SyncRules {
			if rule.Name == *taskName {
				filtered = append(filtered, rule)
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "Error: Task '%s' not found in configuration\n", *taskName)
			os.Exit(1)
		}
		cfg.SyncRules = filtered
		fmt.Printf("\n🎯 Running single task: %s\n", *taskName)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupts
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n\n⚠️  Received interrupt signal, cancelling...")
		cancel()
	}()

	// Create sync engine
	engine := sync.NewEngine(cfg, *dryRun)

	// Set progress callback
	engine.SetProgressFunc(func(info sync.ProgressInfo) {
		switch info.Phase {
		case "manifest":
			fmt.Printf("  📦 Fetching manifest for %s:%s\n", info.Repository, info.Tag)
		case "blob":
			if info.CurrentBlob != "" {
				fmt.Printf("  📥 Syncing blob: %s\n", info.CurrentBlob[:12])
			}
		case "complete":
			fmt.Printf("  ✅ Completed: %s:%s\n", info.Repository, info.Tag)
		}
	})

	// Start sync
	startTime := time.Now()
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🚀 Starting synchronization...")
	fmt.Println(strings.Repeat("=", 60) + "\n")

	if *dryRun {
		fmt.Println("⚠️  DRY RUN MODE - No actual changes will be made\n")
	}

	// Run sync
	if err := engine.SyncAll(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Sync failed: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	duration := time.Since(startTime)
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("✅ Synchronization completed successfully in %s\n", duration.Round(time.Second))
	fmt.Println(strings.Repeat("=", 60))
}

func printConfigSummary(cfg *config.Config) {
	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Println("Configuration Summary")
	fmt.Println(strings.Repeat("-", 60))

	fmt.Printf("Version: %s\n", cfg.Version)
	fmt.Printf("Concurrency: %d workers\n", cfg.Global.Concurrency)
	fmt.Printf("Retry: max %d attempts, interval %v-%v\n",
		cfg.Global.Retry.MaxAttempts,
		cfg.Global.Retry.InitialInterval,
		cfg.Global.Retry.MaxInterval)
	fmt.Printf("Timeout: %v\n", cfg.Global.Timeout)

	fmt.Printf("\nRegistries: %d\n", len(cfg.Registries))
	for name, reg := range cfg.Registries {
		qps := "unlimited"
		if reg.RateLimit.QPS > 0 {
			qps = fmt.Sprintf("%d QPS", reg.RateLimit.QPS)
		}
		fmt.Printf("  - %s: %s (%s)\n", name, reg.URL, qps)
	}

	enabledRules := cfg.GetEnabledRules()
	fmt.Printf("\nSync Rules: %d total, %d enabled\n", len(cfg.SyncRules), len(enabledRules))
	for _, rule := range enabledRules {
		fmt.Printf("  - %s: %s/%s → %s/%s\n",
			rule.Name,
			rule.Source.Registry,
			rule.Source.Repository,
			rule.Target.Registry,
			rule.Target.Repository)

		if len(rule.Tags.Include) > 0 {
			fmt.Printf("    Include: %v\n", rule.Tags.Include)
		}
		if len(rule.Tags.Exclude) > 0 {
			fmt.Printf("    Exclude: %v\n", rule.Tags.Exclude)
		}
		if rule.Tags.Latest > 0 {
			fmt.Printf("    Latest: %d\n", rule.Tags.Latest)
		}
		if len(rule.Architectures) > 0 {
			fmt.Printf("    Architectures: %v\n", rule.Architectures)
		}
	}

	fmt.Println(strings.Repeat("-", 60))
}
