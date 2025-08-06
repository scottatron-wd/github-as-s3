# GitHub-as-S3 Copilot Instructions

## Architecture Overview

This is an S3-compatible proxy that uses Git repositories as storage. The system implements the S3 API by mapping S3 operations to Git/GitHub operations:
- **Buckets** → GitHub repositories under a configured owner
- **Objects** → Files in Git repositories  
- **Versions** → Git commit SHAs
- **Object operations** → Git commits with file changes

## Core Components

### Application Bootstrap (`internal/application/`)
- Uses functional options pattern: `NewApplicationWithOpts(WithDefaultGit(), WithDefaultGithub())`
- Environment-driven config via `getEnv()` with defaults in `application.go`
- Required env vars: `GITHUB_TOKEN`, `GITHUB_OWNER`
- Optional: `GHS3_PORT` (default: 8080), `GHS3_ADDRESS` (default: 0.0.0.0)

### Dual Operation Modes (`internal/server/`)
**Synchronous mode**: Direct GitHub API calls via `internal/github/`
**Async mode**: Local Git operations via `internal/git/` with background pushes

Handler routing switches between sync/async object operations based on `Handler.async` flag.

### S3 Protocol Implementation (`internal/s3/`)
- XML models match S3 API spec exactly
- Error responses use S3-compatible error codes (NoSuchKey, InternalError, etc.)
- ETags use Git commit/blob SHAs 
- `x-amz-version-id` header maps to Git commit SHAs

## Key Patterns

### Structured Logging
Uses zerolog with context propagation:
```go
logger := log.Ctx(c.Request().Context()).With().
    Str("bucket", bucketName).
    Str("object", objectKey).
    Logger()
```

### Error Handling Strategy
1. Catch GitHub API errors and convert to S3 error responses
2. Use typed errors in `internal/git/errors.go` and `internal/github/errors.go` 
3. Always include request context in error logging

### Async Operations Pattern
- `GitAsync` wraps synchronous `Git` operations
- Background jobs use constants from `internal/consts/async.go` (PUT, DELETE)
- Async handlers in `object_async.go` mirror sync handlers in `object.go`

## Development Workflow

### Local Development
```bash
# Setup environment
cp .env.example .env
# Edit .env with GITHUB_TOKEN and GITHUB_OWNER

# Hot reload development
air

# Direct Go run
go run ./cmd/cli/
```

### GitHub Token Requirements
Token needs `repo` and `delete_repo` permissions - validates at startup via `CheckPermissions()`

### Docker Development
- Multi-stage build in `Dockerfile`
- `docker-compose.yaml` uses `.env` file
- Service exposes port 8080 by default

## File Organization Logic

- **Entry point**: `cmd/cli/main.go` - minimal bootstrap
- **HTTP layer**: `internal/server/` - S3 API implementation  
- **Git operations**: `internal/git/` - local Git repo management
- **GitHub operations**: `internal/github/` - GitHub API client
- **Protocol models**: `internal/s3/` - S3 XML structures
- **Utilities**: `internal/util/` - logging helpers, naming conventions

## Integration Points

### S3 Client Compatibility
Tested with rclone, AWS CLI, PocketBase. Configure clients with:
- Endpoint: `http://localhost:8080`
- Provider: "Other" 
- List version: 2

### Git Repository Mapping
- Bucket names become GitHub repo names under configured owner
- Object keys become file paths in repositories
- Git commits preserve S3 metadata via commit messages and file content
