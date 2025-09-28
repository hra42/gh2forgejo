# GitHub to Forgejo Mirror Tool

A robust Go command-line tool for automatically mirroring GitHub repositories to Forgejo with advanced features.

## 🚀 Features

- **Batch Migration**: Migrate all repositories at once
- **Smart Filtering**: Include/exclude repos, handle forks and private repos
- **Concurrent Processing**: Configurable parallel migrations
- **Dry Run Mode**: Test migrations without making changes
- **Mirror Sync**: Keep existing mirrors updated
- **Recreate Mode**: Delete and recreate existing repositories for fresh migration
- **Cleanup**: Remove orphaned mirrors
- **Progress Tracking**: Real-time status updates
- **Flexible Config**: Environment variables or command-line flags

## 🔧 Configuration

### Required Environment Variables
```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
export GITHUB_USER="your-github-username"
export FORGEJO_URL="https://git.hra42.com"
export FORGEJO_TOKEN="your-forgejo-access-token"
export FORGEJO_USER="your-forgejo-username"
```

### Optional Environment Variables
```bash
export FORGEJO_ORG="your-organization"           # Target organization instead of user
export MIRROR_INTERVAL="10m"                     # Mirror sync interval (e.g., '10m', '1h', '24h')
export INCLUDE_PRIVATE="true"                    # Include private repositories
export INCLUDE_FORKS="true"                      # Include forked repositories
export RECREATE_REPOS="true"                     # Delete and recreate existing repositories
export ONLY_REPOS="repo1,repo2,repo3"           # Only migrate specific repos
export EXCLUDE_REPOS="test-repo,old-repo"       # Exclude specific repos
```

## 🎯 Usage Examples

### Basic Migration
```bash
# Migrate all public repositories
./github-forgejo-mirror

# Include private repositories (requires GitHub token with 'repo' scope)
./github-forgejo-mirror --include-private

# Include forks as well
./github-forgejo-mirror --include-private --include-forks
```

**Note:** When migrating repositories, the tool automatically configures pull mirrors with authentication:
- Username: Your GitHub username (from `GITHUB_USER` or `--github-user`)
- Password/Token: Your GitHub personal access token (from `GITHUB_TOKEN` or `--github-token`)
This ensures that Forgejo can automatically pull updates from GitHub, even for private repositories.

### Advanced Usage
```bash
# Dry run to see what would be migrated
./github-forgejo-mirror --dry-run --verbose

# Migrate only specific repositories
./github-forgejo-mirror --only="important-repo,another-repo"

# Exclude specific repositories
./github-forgejo-mirror --exclude="test-repo,old-stuff" --include-private

# Migrate to organization instead of user
./github-forgejo-mirror --organization="my-org" --include-private

# Fast migration with more concurrent workers
./github-forgejo-mirror --concurrent=10 --include-private

# Migration with cleanup of orphaned mirrors
./github-forgejo-mirror --cleanup --include-private

# Recreate existing repositories (delete and re-migrate)
./github-forgejo-mirror --recreate --include-private

# Set custom mirror sync interval (default is Forgejo's default)
./github-forgejo-mirror --mirror-interval="30m" --include-private

# Use hourly sync for less critical repos
./github-forgejo-mirror --mirror-interval="1h" --include-private

# Daily sync for archived or stable repos
./github-forgejo-mirror --mirror-interval="24h" --include-private
```

### Command-line Flags
```bash
Usage of ./github-forgejo-mirror:
  -github-token string       GitHub personal access token
  -github-user string        GitHub username
  -forgejo-url string        Forgejo instance URL
  -forgejo-token string      Forgejo access token
  -forgejo-user string       Forgejo username
  -organization string       Forgejo organization (optional)
  -mirror-interval string    Mirror sync interval (e.g., '10m', '1h', '24h')
  -include-private           Include private repositories
  -include-forks             Include forked repositories
  -dry-run                   Show what would be done without making changes
  -cleanup                   Remove mirrors that no longer exist on GitHub
  -recreate                  Delete and recreate existing repositories
  -concurrent int            Number of concurrent migrations (default 3)
  -verbose                   Enable verbose logging
  -only string               Comma-separated list of repos to migrate
  -exclude string            Comma-separated list of repos to exclude
  -version                   Show version and exit
```

## 🛠️ Development Setup

```bash
# Clone and setup
git clone <your-repo>
cd github-forgejo-mirror

# Install dependencies
go mod tidy

# Run tests (if you add them)
go test ./...

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o github-forgejo-mirror-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o github-forgejo-mirror-windows-amd64.exe .
GOOS=darwin GOARCH=amd64 go build -o github-forgejo-mirror-darwin-amd64 .
```

## 🔐 Token Setup

### GitHub Personal Access Token
1. Go to GitHub Settings → Developer settings → Personal access tokens
2. Generate new token (classic)
3. Select scopes:
   - `repo` (for private repositories)
   - `public_repo` (for public repositories)
   - `read:org` (if migrating organization repos)

**Important:** This token is used for two purposes:
- Fetching repository information from GitHub API
- **Authentication for pull mirrors** - Forgejo will use this token to authenticate with GitHub and automatically pull changes

### Forgejo Access Token
1. Login to your Forgejo instance
2. Go to Settings → Applications
3. Generate new token
4. Select permissions: `repository` (read/write), `organization` (if using orgs)

## 📋 Migration Features

### What gets migrated:
- ✅ Repository code and history
- ✅ Branches and tags
- ✅ Issues (if supported by Forgejo)
- ✅ Pull requests (if supported)
- ✅ Releases
- ✅ Wiki (if present)
- ✅ Milestones and labels
- ✅ Repository settings (private/public)

### Mirror Features:
- 🔄 Automatic periodic sync from GitHub
- 🔄 Manual sync trigger via API
- 🔄 Keeps repositories in sync with upstream
- 🔐 Authenticated pulling using GitHub token and username
- 🔐 Supports both public and private repository mirroring
- ⏰ Configurable sync intervals (10m, 30m, 1h, 24h, etc.)

## 🚨 Error Handling

The tool includes comprehensive error handling for:
- Network timeouts and retries
- API rate limiting
- Authentication failures
- Repository conflicts
- Invalid configurations

## 📊 Output Example

```
🚀 GitHub to Forgejo Mirror Tool v1.0.0
   Source: your-user@github.com
   Target: https://git.hra42.com

📡 Fetching GitHub repositories...
   Found 42 repositories on GitHub

🔄 Starting migration...
✅ Successfully migrated: awesome-project
✅ Successfully migrated: cool-library
⚠️  Repository already exists: old-mirror
✅ Successfully migrated: new-tool

📊 Migration Summary:
   Total repos: 42
   Migrated: 38
   Skipped: 3
   Failed: 1
   Duration: 2m34s

🎉 Migration completed successfully!
```

## 🤝 Contributing

1. Fork the repository
2. Create feature branch
3. Add tests for new features
4. Submit pull request
