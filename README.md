# Expense Tracker

[![Go Report Card](https://goreportcard.com/badge/github.com/rajware/expensetracker-go)](https://goreportcard.com/report/github.com/rajware/expensetracker-go)
[![GitHub release](https://img.shields.io/github/v/release/rajware/expensetracker-go)](https://github.com/rajware/expensetracker-go/releases/latest)

Expense Tracker is a personal finance application designed to help users monitor their daily spending. It provides a simple and intuitive interface to record expenses as they occur.

## Key Features

- **Personalized Expense Tracking**: Record expenses with descriptions, amounts, and dates.
- **Expense Dashboard**: View expenses with a modern UI, featuring pagination, sorting, and date filtering.
- **Secure Authentication**: Session-based authentication with HMAC-signed cookies and password management.
- **Multi-Database Support**: Use SQLite for local use and PostgreSQL for production-grade deployments.
- **Container Ready**: Ready to deploy on Docker, Kubernetes, or OpenShift.

## Installation and Setup

There are several ways to get the Expense Tracker application running depending on your preference and environment.

### Docker Image

Container images are published to:

* `quay.io/rajware/expensetracker-go`
* `ghcr.io/rajware/expensetracker-go`

You can pull the image from your preferred registry and run it using Docker.

1. Pull the image from the registry:
   ```bash
   docker pull quay.io/rajware/expensetracker-go:latest
   ```
2. Run the container, ensuring you provide a signing key for cookie authentication and mount a volume at `/app/data`:
   ```bash
   docker run -d \
     -v etvol:/app/data \
     -p 8080:8080 \
     -e ET_HMAC_KEY=your_secret_key_here \
     quay.io/rajware/expensetracker-go:latest
   ```

### Docker Compose

Docker Compose manifests are provided in each release. You can download them by navigating to the [GitHub Releases Page](https://github.com/rajware/expensetracker-go/releases).

* `tracker-sqlite.compose.yaml`
* `tracker-postgres.compose.yaml`

To start the application with a local SQLite database:
```bash
docker compose -f tracker-sqlite.compose.yaml up -d
```

To start the application with a PostgreSQL database:
```bash
docker compose -f tracker-postgres.compose.yaml up -d
```

### GitHub Release

If you do not want to use containers, you can download a pre-built binary for your platform. Binaries are provided for Linux, macOS and Windows on amd64 and arm64 architectures.

1. Navigate to the latest release on the [GitHub Releases Page](https://github.com/rajware/expensetracker-go/releases).
2. Download the binary corresponding to your operating system and architecture (e.g., `tracker-web_linux_amd64`, `tracker-web_darwin_arm64`, or `tracker-web_windows_amd64.exe`).
3. Run the binary from your terminal.

When running the binary, the `-hmac-key` flag is mandatory for session security. You can also configure the database using optional flags or environment variables:

```bash
./tracker-web_linux_amd64 -hmac-key "your_secret_key" -db-driver sqlite -db-path data/expense_tracker.db
```

### Configuration Options

The following options can be set via CLI flags or environment variables.

| CLI Flag | Environment Variable | Description | Default |
|---|---|---|---|
| `-hmac-key` | `ET_HMAC_KEY` | **Mandatory** HMAC signing key for authentication. | |
| `-db-driver` | `ET_DB_DRIVER` | Database driver (sqlite or postgres). | `sqlite` |
| `-db-path` | `ET_DB_PATH` | Path to the SQLite database file. | `data/expense_tracker.db` |
| `-db-url` | `ET_DB_URL` | PostgreSQL connection URL (required if driver is postgres). | |
| `-addr` | `ET_ADDR` | Address the server listens on. | `:8080` |

### Kubernetes and OpenShift

Deployment manifests for Kubernetes or OpenShift environments are provided in each release.

* `tracker-sqlite.k8s.yaml`
* `tracker-postgres.k8s.yaml`

In each manifest, there is a `Secret` containing configuration data. These are called `tracker-sqlite-secret` and `tracker-postgres-secret` respectively. Both secrets have a key called `hmacKey`, which should be set to a base64-encoded HMAC signing key. The Postgres secret additionally has the following keys:

|Key|Value|
|---|---|
|`dbPassword`|A base64-encoded password. Set to `something` in the release manifest.|
|`dbUrl`|A base64 encoded Postgres URL, in the form "postgres://trackeruser:_PASSWORD_@tracker-postgres-db-svc:5432/expensetrackerdb". Set using the default password above in the release manifest.|

By default, the Kubernetes manifests expose the Expense Tracker on port 8080 via a `NodePort`-type service. If your cluster has an Ingress controller set up, you can change the service to use `ClusterIP` and configure the Ingress to route traffic to the service. If you do this, you should also change the `host` in the `Ingress` defined in both manifests to your desired `host` (e.g. `tracker.yourdomain.com`) which points to the Ingress controller.

To deploy the version using SQLite:
```bash
kubectl apply -f tracker-sqlite.k8s.yaml
```

To deploy the version using PostgreSQL:
```bash
kubectl apply -f tracker-postgres.k8s.yaml
```
