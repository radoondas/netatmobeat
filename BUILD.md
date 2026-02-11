# Netatmobeat — Build & Release

## Requirements

* [Go](https://golang.org/dl/) 1.24+
* [nfpm](https://nfpm.goreleaser.com/) (for deb/rpm packages)
* [Docker](https://www.docker.com/) with buildx (for container images)

## Local Build

```bash
# Single-platform build (outputs ./netatmobeat)
go build ./...

# Or use the goenv.sh wrapper if system Go is too old
./goenv.sh go build ./...
```

## Cross-Compilation

Build for all supported platforms (linux/darwin amd64+arm64, windows amd64):

```bash
./scripts/build.sh
```

Binaries are output to `dist/netatmobeat-{VERSION}-{os}-{arch}/`.

Override the version:
```bash
BEAT_VERSION=9.3.0-rc1 ./scripts/build.sh
# or
./scripts/build.sh 9.3.0-rc1
```

Build for specific platforms only:
```bash
PLATFORMS="linux/amd64 darwin/arm64" ./scripts/build.sh
```

## Packaging

Create tar.gz, zip, deb, rpm archives with SHA512 checksums:

```bash
# Build first, then package
./scripts/build.sh
./scripts/package.sh
```

Skip deb/rpm if nfpm is not installed:
```bash
SKIP_DEB_RPM=true ./scripts/package.sh
```

Verify all expected artifacts exist:
```bash
./scripts/verify-assets.sh
```

### Install nfpm

```bash
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.41.1
```

## Docker Image

Build a local Docker image:
```bash
docker buildx build --platform linux/amd64 -t netatmobeat:local .
```

Multi-arch build:
```bash
docker buildx build --platform linux/amd64,linux/arm64 -t netatmobeat:local .
```

## Tests

```bash
go test ./beater/...
go test ./config/...
go vet ./...
```

## Field Generation

Regenerate `fields.yml` and `include/fields.go` from `_meta/fields.yml`:
```bash
make update
```

## Release Process

Releases are automated via GitHub Actions:

1. **CI** runs on every push/PR: build, vet, test (`.github/workflows/ci.yml`)
2. **Dry-run** release via `workflow_dispatch` in `.github/workflows/release.yml` — builds and packages all artifacts without publishing
3. **Tag** a release: `git tag v9.3.0 && git push origin v9.3.0`
4. GitHub Actions automatically:
   - Cross-compiles for all platforms
   - Creates tar.gz/zip/deb/rpm packages with SHA512 checksums
   - Pushes multi-arch Docker image to Docker Hub
   - Creates GitHub Release with all artifacts

### Artifact Matrix

| Format | Platforms |
|--------|-----------|
| tar.gz | linux-x86_64, linux-aarch64, darwin-x86_64, darwin-aarch64 |
| zip | windows-x86_64 |
| deb | amd64, arm64 |
| rpm | x86_64, aarch64 |
| Docker | linux/amd64, linux/arm64 |

### Required Secrets

For Docker Hub publishing, set these as GitHub repository secrets:
- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN`