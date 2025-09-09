// github-forgejo-mirror - Automated GitHub to Forgejo repository mirroring tool
// Author: HRA42 Team
// License: MIT

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

const (
	version   = "1.0.0"
	userAgent = "github-forgejo-mirror/" + version
)

// Config holds all configuration parameters
type Config struct {
	GitHubToken    string
	GitHubUser     string
	ForgejoURL     string
	ForgejoToken   string
	ForgejoUser    string
	Organization   string
	IncludePrivate bool
	IncludeForks   bool
	DryRun         bool
	CleanupOrphans bool
	Concurrent     int
	Verbose        bool
	OnlyRepos      []string
	ExcludeRepos   []string
}

// GitHubRepo represents a GitHub repository
type GitHubRepo struct {
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	CloneURL    string `json:"clone_url"`
	Private     bool   `json:"private"`
	Fork        bool   `json:"fork"`
	Language    string `json:"language"`
	Stars       int    `json:"stargazers_count"`
	UpdatedAt   string `json:"updated_at"`
}

// ForgejoMigrationRequest represents a Forgejo migration API request
type ForgejoMigrationRequest struct {
	CloneAddr    string `json:"clone_addr"`
	RepoName     string `json:"repo_name"`
	RepoOwner    string `json:"repo_owner,omitempty"`
	Description  string `json:"description"`
	Private      bool   `json:"private"`
	Mirror       bool   `json:"mirror"`
	AuthToken    string `json:"auth_token,omitempty"`
	AuthUsername string `json:"auth_username,omitempty"`
	Issues       bool   `json:"issues"`
	PullRequests bool   `json:"pull_requests"`
	Releases     bool   `json:"releases"`
	Wiki         bool   `json:"wiki"`
	Milestones   bool   `json:"milestones"`
	Labels       bool   `json:"labels"`
}

// ForgejoRepo represents a Forgejo repository
type ForgejoRepo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Mirror   bool   `json:"mirror"`
}

// Client wraps HTTP client with custom methods
type Client struct {
	httpClient *http.Client
	config     *Config
}

// NewClient creates a new HTTP client with custom configuration
func NewClient(config *Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// GetGitHubRepos fetches all repositories for a user
func (c *Client) GetGitHubRepos(ctx context.Context) ([]*GitHubRepo, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.config.GitHubToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	client.UserAgent = userAgent

	var allRepos []*github.Repository
	opts := &github.RepositoryListOptions{
		Type:        "owner",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := client.Repositories.List(ctx, "", opts)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch GitHub repos: %w", err)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	var result []*GitHubRepo
	for _, repo := range allRepos {
		// Apply filters
		if !c.config.IncludeForks && repo.GetFork() {
			continue
		}
		if !c.config.IncludePrivate && repo.GetPrivate() {
			continue
		}
		if c.shouldSkipRepo(repo.GetName()) {
			continue
		}

		result = append(result, &GitHubRepo{
			Name:        repo.GetName(),
			FullName:    repo.GetFullName(),
			Description: repo.GetDescription(),
			CloneURL:    repo.GetCloneURL(),
			Private:     repo.GetPrivate(),
			Fork:        repo.GetFork(),
			Language:    repo.GetLanguage(),
			Stars:       repo.GetStargazersCount(),
			UpdatedAt:   repo.GetUpdatedAt().Format(time.RFC3339),
		})
	}

	return result, nil
}

// GetForgejoRepos fetches all repositories from Forgejo
func (c *Client) GetForgejoRepos(ctx context.Context) ([]*ForgejoRepo, error) {
	url := fmt.Sprintf("%s/api/v1/user/repos?limit=100", c.config.ForgejoURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+c.config.ForgejoToken)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Forgejo repos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Forgejo API returned status %d", resp.StatusCode)
	}

	var repos []*ForgejoRepo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo repos: %w", err)
	}

	return repos, nil
}

// MigrateRepo creates a mirrored repository in Forgejo
func (c *Client) MigrateRepo(ctx context.Context, repo *GitHubRepo) error {
	if c.config.DryRun {
		fmt.Printf("[DRY RUN] Would migrate: %s\n", repo.Name)
		return nil
	}

	migration := &ForgejoMigrationRequest{
		CloneAddr:    repo.CloneURL,
		RepoName:     repo.Name,
		RepoOwner:    c.config.ForgejoUser,
		Description:  repo.Description,
		Private:      repo.Private,
		Mirror:       true,
		AuthToken:    c.config.GitHubToken,
		AuthUsername: c.config.GitHubUser,
		Issues:       true,
		PullRequests: true,
		Releases:     true,
		Wiki:         true,
		Milestones:   true,
		Labels:       true,
	}

	// Override owner if organization is specified
	if c.config.Organization != "" {
		migration.RepoOwner = c.config.Organization
	}

	body, err := json.Marshal(migration)
	if err != nil {
		return fmt.Errorf("failed to marshal migration request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/repos/migrate", c.config.ForgejoURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+c.config.ForgejoToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to migrate repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("✅ Successfully migrated: %s\n", repo.Name)
		return nil
	} else if resp.StatusCode == http.StatusConflict {
		fmt.Printf("⚠️  Repository already exists: %s\n", repo.Name)
		return nil
	}

	return fmt.Errorf("migration failed with status %d for repo %s", resp.StatusCode, repo.Name)
}

// SyncMirror triggers a sync for an existing mirror
func (c *Client) SyncMirror(ctx context.Context, repoName string) error {
	if c.config.DryRun {
		fmt.Printf("[DRY RUN] Would sync mirror: %s\n", repoName)
		return nil
	}

	owner := c.config.ForgejoUser
	if c.config.Organization != "" {
		owner = c.config.Organization
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/mirror-sync", c.config.ForgejoURL, owner, repoName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+c.config.ForgejoToken)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to sync mirror: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("🔄 Sync triggered for: %s\n", repoName)
		return nil
	}

	return fmt.Errorf("sync failed with status %d for repo %s", resp.StatusCode, repoName)
}

// shouldSkipRepo checks if a repository should be skipped based on filters
func (c *Client) shouldSkipRepo(repoName string) bool {
	// If only specific repos are requested
	if len(c.config.OnlyRepos) > 0 {
		for _, name := range c.config.OnlyRepos {
			if name == repoName {
				return false
			}
		}
		return true
	}

	// If repo is in exclude list
	for _, name := range c.config.ExcludeRepos {
		if name == repoName {
			return true
		}
	}

	return false
}

// parseStringSlice parses a comma-separated string into a slice
func parseStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// loadConfig loads configuration from environment variables and flags
func loadConfig() *Config {
	config := &Config{}

	// Command line flags
	flag.StringVar(&config.GitHubToken, "github-token", os.Getenv("GITHUB_TOKEN"), "GitHub personal access token")
	flag.StringVar(&config.GitHubUser, "github-user", os.Getenv("GITHUB_USER"), "GitHub username")
	flag.StringVar(&config.ForgejoURL, "forgejo-url", os.Getenv("FORGEJO_URL"), "Forgejo instance URL")
	flag.StringVar(&config.ForgejoToken, "forgejo-token", os.Getenv("FORGEJO_TOKEN"), "Forgejo access token")
	flag.StringVar(&config.ForgejoUser, "forgejo-user", os.Getenv("FORGEJO_USER"), "Forgejo username")
	flag.StringVar(&config.Organization, "organization", os.Getenv("FORGEJO_ORG"), "Forgejo organization (optional)")
	flag.BoolVar(&config.IncludePrivate, "include-private", os.Getenv("INCLUDE_PRIVATE") == "true", "Include private repositories")
	flag.BoolVar(&config.IncludeForks, "include-forks", os.Getenv("INCLUDE_FORKS") == "true", "Include forked repositories")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Show what would be done without making changes")
	flag.BoolVar(&config.CleanupOrphans, "cleanup", false, "Remove mirrors that no longer exist on GitHub")
	flag.IntVar(&config.Concurrent, "concurrent", 3, "Number of concurrent migrations")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")

	var onlyRepos, excludeRepos string
	flag.StringVar(&onlyRepos, "only", os.Getenv("ONLY_REPOS"), "Comma-separated list of repos to migrate (migrate only these)")
	flag.StringVar(&excludeRepos, "exclude", os.Getenv("EXCLUDE_REPOS"), "Comma-separated list of repos to exclude")

	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version and exit")

	flag.Parse()

	if showVersion {
		fmt.Printf("github-forgejo-mirror version %s\n", version)
		os.Exit(0)
	}

	config.OnlyRepos = parseStringSlice(onlyRepos)
	config.ExcludeRepos = parseStringSlice(excludeRepos)

	// Validation
	if config.GitHubToken == "" {
		log.Fatal("GitHub token is required (--github-token or GITHUB_TOKEN)")
	}
	if config.GitHubUser == "" {
		log.Fatal("GitHub username is required (--github-user or GITHUB_USER)")
	}
	if config.ForgejoURL == "" {
		log.Fatal("Forgejo URL is required (--forgejo-url or FORGEJO_URL)")
	}
	if config.ForgejoToken == "" {
		log.Fatal("Forgejo token is required (--forgejo-token or FORGEJO_TOKEN)")
	}
	if config.ForgejoUser == "" && config.Organization == "" {
		log.Fatal("Either Forgejo user or organization is required")
	}

	// Clean up Forgejo URL
	config.ForgejoURL = strings.TrimSuffix(config.ForgejoURL, "/")

	return config
}

// printStats prints migration statistics
func printStats(total, migrated, skipped, failed int, duration time.Duration) {
	fmt.Printf("\n📊 Migration Summary:\n")
	fmt.Printf("   Total repos: %d\n", total)
	fmt.Printf("   Migrated: %d\n", migrated)
	fmt.Printf("   Skipped: %d\n", skipped)
	fmt.Printf("   Failed: %d\n", failed)
	fmt.Printf("   Duration: %v\n", duration.Round(time.Second))
}

func main() {
	config := loadConfig()
	client := NewClient(config)

	ctx := context.Background()
	startTime := time.Now()

	fmt.Printf("🚀 GitHub to Forgejo Mirror Tool v%s\n", version)
	fmt.Printf("   Source: %s@github.com\n", config.GitHubUser)
	fmt.Printf("   Target: %s\n", config.ForgejoURL)
	if config.DryRun {
		fmt.Printf("   Mode: DRY RUN\n")
	}
	fmt.Println()

	// Fetch GitHub repositories
	fmt.Println("📡 Fetching GitHub repositories...")
	githubRepos, err := client.GetGitHubRepos(ctx)
	if err != nil {
		log.Fatalf("Failed to fetch GitHub repositories: %v", err)
	}
	fmt.Printf("   Found %d repositories on GitHub\n", len(githubRepos))

	// Optionally fetch existing Forgejo repos for cleanup
	var forgejoRepos []*ForgejoRepo
	if config.CleanupOrphans {
		fmt.Println("📡 Fetching Forgejo repositories for cleanup...")
		forgejoRepos, err = client.GetForgejoRepos(ctx)
		if err != nil {
			log.Printf("Warning: Failed to fetch Forgejo repos for cleanup: %v", err)
		} else {
			fmt.Printf("   Found %d repositories on Forgejo\n", len(forgejoRepos))
		}
	}

	// Create a semaphore for concurrent operations
	semaphore := make(chan struct{}, config.Concurrent)
	results := make(chan string, len(githubRepos))

	var migrated, skipped, failed int

	// Process each repository
	fmt.Println("\n🔄 Starting migration...")
	for _, repo := range githubRepos {
		go func(r *GitHubRepo) {
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			if config.Verbose {
				fmt.Printf("🔍 Processing: %s (⭐%d, %s)\n", r.Name, r.Stars, r.Language)
			}

			if err := client.MigrateRepo(ctx, r); err != nil {
				results <- fmt.Sprintf("❌ Failed to migrate %s: %v", r.Name, err)
				return
			}
			results <- "success"
		}(repo)
	}

	// Collect results
	for i := 0; i < len(githubRepos); i++ {
		result := <-results
		if result == "success" {
			migrated++
		} else if strings.Contains(result, "already exists") {
			skipped++
		} else {
			failed++
			if config.Verbose {
				fmt.Println(result)
			}
		}
	}

	// Cleanup orphaned mirrors
	if config.CleanupOrphans && len(forgejoRepos) > 0 {
		fmt.Println("\n🧹 Cleaning up orphaned mirrors...")
		githubNames := make(map[string]bool)
		for _, repo := range githubRepos {
			githubNames[repo.Name] = true
		}

		for _, forgejoRepo := range forgejoRepos {
			if forgejoRepo.Mirror && !githubNames[forgejoRepo.Name] {
				fmt.Printf("🗑️  Found orphaned mirror: %s\n", forgejoRepo.Name)
				// Note: Deletion would require additional API call
			}
		}
	}

	duration := time.Since(startTime)
	printStats(len(githubRepos), migrated, skipped, failed, duration)

	if failed > 0 {
		fmt.Printf("\n⚠️  %d repositories failed to migrate. Check logs for details.\n", failed)
		os.Exit(1)
	}

	fmt.Println("\n🎉 Migration completed successfully!")
}
