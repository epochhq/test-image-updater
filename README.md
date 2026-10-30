# test-image-updater

A Go demonstration for ArgoCD Image Updater and the platform's image/signature workflow. The application displays version, commit, build time, hostname and environment so image changes are observable.

| Path     | Behavior                                                  |
| -------- | --------------------------------------------------------- |
| /        | HTML build metadata and optional Rybbit script            |
| /health  | Healthy JSON with a UTC timestamp                         |
| /version | Version, commit, buildTime, hostname and environment JSON |
| /metrics | Prometheus metrics                                        |

| Metric                                   | Type      | Purpose                             |
| ---------------------------------------- | --------- | ----------------------------------- |
| epochcloud_http_requests_total           | Counter   | Requests by method, path and status |
| epochcloud_http_request_duration_seconds | Histogram | Request duration by method and path |
| epochcloud_app_info                      | Gauge     | App version and commit              |

Platform PodMonitor discovery uses app.kubernetes.io/part-of:epochcloud labels. Middleware retains method/path/status labels and duration observations. Unknown paths return 404. ENVIRONMENT defaults to dev; RYBBIT_SITE_ID and RYBBIT_HOST remain process-environment settings.

## Local toolchain

Install Nix and direnv, then run `direnv allow`. The shell provides the compiler pinned by go.mod. The root go.mod is the compiler contract; the Nix check and Docker dependency stage compare it with Go's actual version using Go's native module metadata. Shell entry does not install modules.

```bash
go mod download
go -C devtools mod download
```

The isolated devtools module pins Staticcheck and gocyclo. It keeps checker dependencies out of the application's module graph. The selected supported compiler includes the [crypto/x509 fix](https://pkg.go.dev/vuln/GO-2026-6809). Staticcheck 2026.2.2 supports the compiler's [current export-data format](https://github.com/dominikh/go-tools/releases/tag/2026.2.2).

## Checks

After module setup, run the maintained native checks offline:

```bash
export GOPROXY=off GOSUMDB=off
nix flake check --offline --no-update-lock-file
go run -mod=readonly ./tools/check-format
go mod verify
go -C devtools mod verify
go test -mod=readonly -timeout=5s ./...
go vet -mod=readonly ./...
go tool -modfile=devtools/go.mod staticcheck ./...
go tool -modfile=devtools/go.mod gocyclo -over 10 .
```

The formatter bridge makes unformatted Go source a failing check. Tests use in-process handlers and local files/processes. Native analysis tools run through Go's pinned tool directives.

## Packaging

The official Go builder and runtime are pinned by index digest. Production module checksums are copied before dependency download, and a failed download stops the build. Module builds are readonly. Build metadata retains VERSION, COMMIT and BUILD_TIME arguments.

```bash
docker build --target dependencies -t test-image-updater-dependencies . &&
docker build --network=none \
  --build-context dependencies=docker-image://test-image-updater-dependencies \
  -t test-image-updater-local \
  --build-arg VERSION="$(cat VERSION)" \
  --build-arg COMMIT="$(git rev-parse HEAD)" .
docker run --rm --read-only --cap-drop=ALL --security-opt=no-new-privileges \
  -p 127.0.0.1:8080:8080 test-image-updater-local
```

The dependency target prepares an image containing the verified modules. The offline build uses that image as a [named build context](https://docs.docker.com/reference/cli/docker/buildx/build/#build-context) for the dependencies stage. The command chain stops when preparation fails, and the builder checks the compiler against the copied application module before compiling. Each container keeps its unprivileged user and serves PORT (default 8080). Startup failure exits nonzero.

## Dependency updates

Renovate requires 14-day published timestamps for normal updates and vulnerability alerts. The exact Go 1.27.3 crypto/x509 release is the sole compiler age exception. Go module versions, compiler source, compiler recipe input and builder image update as a group within the supported Go 1.27 line. Native Nix lock regeneration owns archive hashes. Package-set revisions use the maintained GitHub digest datasource's commit timestamps. The Renovate operator must support native postUpgradeTasks.installTools and allow exactly `nix --extra-experimental-features 'nix-command flakes' flake lock`.

The runtime retains official Distroless nonroot behavior on the currently maintained Debian 13 family. The pinned digest includes the [upstream CA remediation](https://github.com/GoogleContainerTools/distroless/commit/73f4b38fb74c96475f47f56eb81ad267246939f4), backed by Debian's ca-certificates security package.

Renovate's native Docker datasource currently supplies release timestamps only for Docker Hub. Ordinary GCR Distroless digest updates remain pending under timestamp-required policy. Automatic ordinary update qualification remains open until a maintained timestamp-capable path is proven. The exact security image above has retained upstream provenance.

## Deployment purpose

The existing platform workflow builds and publishes images to Harbor; ArgoCD Image Updater watches new digests and updates the Application. Image signing/promotion belongs to the platform workflow. The application preserves its metadata display and Prometheus signals for observing those updates.

Related source: [Epochcloud](https://github.com/EpochBoy/epochcloud) and [Kargo demonstration](https://github.com/EpochBoy/epochcloud-test).
