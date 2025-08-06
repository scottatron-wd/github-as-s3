# github-as-s3

> This post is an educational exploration into the S3 protocol and a practical guide on how one might implement it using Git as the underlying storage mechanism.
> Please be mindful of the terms of service and code of conduct for any Git hosting provider you choose to use with these concepts; I am not responsible for any actions taken against your accounts.

![a picture of git pretending to be s3](https://kristun.dev/_astro/s3-git-scooby-doo.D9vfHZvy_ZyPgXk.webp)

This project contains the source code for the Github+S3 proxy.

The intended use is to proxy object storage onto Git itself. You can read more about this [here](https://kristun.dev/posts/git-as-s3/)

The following routes are implemented

- [PutObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObject.html)
- [GetObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObject.html)
- [HeadObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_HeadObject.html)
- [DeleteObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObject.html)
- [CopyObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CopyObject.html)
- [ListObjectsV2](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html)
- [CreateBucket](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateBucket.html)
- [DeleteBucket](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucket.html)
- [ListBuckets](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListBuckets.html)

# Getting started

## Run it with Go

```bash
cp .env.example .env
go run ./cmd/cli/
```

### Local Repository Mode

You can run the server against a local git repository without any remote operations:

```bash
# Using command line flag
go run ./cmd/cli/ --local-repo /path/to/your/git/repo

# Or using environment variable
export GHS3_LOCAL_REPO_PATH=/path/to/your/git/repo
go run ./cmd/cli/
```

In local mode:

- The server will open your existing git repository instead of cloning from GitHub
- No remote git operations (push/pull) will be performed
- GitHub token is not required

### Branch Support

github-as-s3 supports configurable Git branches for repository operations:

#### Global Branch Configuration

```bash
# Set default branch for all repository operations
export GHS3_DEFAULT_BRANCH=main
go run ./cmd/cli/

# Or use CLI flag
go run ./cmd/cli/ --default-branch=main
```

#### Per-Bucket Branch Configuration

When creating a bucket (repository), specify a custom branch using the `X-Ghs3-Branch` header:

```bash
# Using curl
curl -X PUT http://localhost:8080/my-bucket \
    -H "X-Ghs3-Branch: develop"
```

All operations on that bucket will use the specified branch. If no branch is specified, the configured default branch is used.

- All S3 operations work with local git commits only

The environment values require the following:

```bash
# required for remote mode - replace with your own username and token
GITHUB_TOKEN=
GITHUB_OWNER=ktunprasert

# optional
# hosting related
GHS3_PORT=
GHS3_ADDRESS=
# committer's information
# these are the defaults defined in application.go
GIT_USERNAME=GHS3
GIT_EMAIL=bot@ghs3.com

# git branch configuration (optional)
# default branch to use for all repository operations
GHS3_DEFAULT_BRANCH=master

# local repository mode (optional)
# when set, operates on the local git repository without remote operations
GHS3_LOCAL_REPO_PATH=/path/to/your/git/repo
```

The token requires 2 permissions: `repo`, `delete_repo`

## Run it with Docker

```bash
docker build . -t ghs3
docker run --rm -t ghs3 --env "GITHUB_OWNER=ktunprasert" --env "GITHUB_TOKEN=$TOKEN"
```

## Run it with docker-compose

```bash
# configure your compose file with proper environment values
docker-compose up -d
```

## Development

```bash
air
```

Hot reloading with [`air`](https://github.com/air-verse/air) is configured

## Testing

The project includes comprehensive S3 API compatibility testing:

### Custom S3 Test Suite (Recommended)

Focused testing of only the operations that github-as-s3 supports:

```bash
# Quick setup and test
make test-s3-custom
```

The custom test suite:

- Tests only supported S3 operations (no early failures)
- Runs in 2-5 seconds vs 2-30 minutes for comprehensive suites
- Provides detailed pass/fail status for each operation
- Includes advanced tests like prefix filtering and multi-object operations

### MinIO Mint (Comprehensive)

Industry-standard S3 compatibility testing with [MinIO Mint](https://github.com/minio/mint):

```bash
# Full compatibility test suite
make test-full

# Test with Docker Compose
make test-docker
```

MinIO Mint tests the S3 API against multiple clients including:

- AWS CLI and SDKs (Go, Java, JavaScript, Python, PHP, Ruby)
- MinIO clients and tools
- Third-party S3 tools (s3cmd, rclone)

For detailed testing documentation, see [`test/README.md`](./test/README.md) and [`test/S3_TESTS.md`](./test/S3_TESTS.md).

# Compatible & Tested applications

| Tool                                  | Tested                                                            |     |
| ------------------------------------- | ----------------------------------------------------------------- | --- |
| [rclone](https://rclone.org/)         | cp, sync, delete, mkdir, purge, ls, deletefile, touch, lsd, rmdir | ✅  |
| [aws s3](https://aws.amazon.com/cli/) | mb, cp, ls, rm                                                    | ✅  |
| [pocketbase](https://pocketbase.io/)  | creating and restoring back up + deleting files                   | ✅  |
| [pocketbase](https://pocketbase.io/)  | using as file storage                                             | ❓  |

# FAQ

## How to set up my rclone to read to it?

```conf
[ghs3]
type = s3
provider = Other
endpoint = http://localhost:8080
list_version = 2
```

This was all I needed to start moving files

```bash
rclone mkdir ghs3:/test-repo/
rclone copy hello-world.txt ghs3:/test-repo/
```
