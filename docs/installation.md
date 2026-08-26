# Install Payment Sandbox

This guide is for using a published Payment Sandbox release. It does not
require Git, Go or a repository checkout.

## Choose a release

Published releases are available on the [GitHub Releases page](https://github.com/proxynth/payment-sandbox/releases).
Prefer the latest stable release. Development builds and source checkouts are
intended for contributors; see the [contributor getting started guide](getting-started.md)
when working on the repository itself.

Release tags use the `vMAJOR.MINOR.PATCH` format, such as `v1.2.1`.

Each release provides these archives:

| Operating system | Intel/AMD 64-bit | ARM 64-bit |
| --- | --- | --- |
| macOS | `payment-sandbox_<version>_darwin_amd64.tar.gz` | `payment-sandbox_<version>_darwin_arm64.tar.gz` |
| Linux | `payment-sandbox_<version>_linux_amd64.tar.gz` | `payment-sandbox_<version>_linux_arm64.tar.gz` |
| Windows | `payment-sandbox_<version>_windows_amd64.zip` | `payment-sandbox_<version>_windows_arm64.zip` |

The filenames use the version without the leading `v`. For example, the
`v1.2.1` Linux ARM64 archive is
`payment-sandbox_1.2.1_linux_arm64.tar.gz`.

## Download and verify an archive

Download the archive and the `payment-sandbox_<version>_checksums.txt` file
from the same release. Verify the archive before extracting it.

On macOS or Linux, from the directory containing both files:

```bash
sha256sum --check payment-sandbox_1.2.1_checksums.txt --ignore-missing
```

If `sha256sum` is not available on macOS, use:

```bash
shasum -a 256 payment-sandbox_1.2.1_linux_arm64.tar.gz
```

Compare the printed digest with the matching line in the checksums file.
On Windows, use PowerShell:

```powershell
Get-FileHash .\payment-sandbox_1.2.1_windows_amd64.zip -Algorithm SHA256
```

Compare the result with the matching entry in
`payment-sandbox_1.2.1_checksums.txt`.

## Extract and install

Extract a macOS or Linux archive:

```bash
mkdir payment-sandbox-1.2.1-linux-arm64
tar -xzf payment-sandbox_1.2.1_linux_arm64.tar.gz -C payment-sandbox-1.2.1-linux-arm64
cd payment-sandbox-1.2.1-linux-arm64
```

The archive contains the `payment-sandbox` binary and `.env.example`.
Optionally install the binary in a user-local directory:

```bash
mkdir -p "$HOME/.local/bin"
install -m 0755 payment-sandbox "$HOME/.local/bin/payment-sandbox"
export PATH="$HOME/.local/bin:$PATH"
```

On Windows, extract the ZIP with File Explorer or PowerShell and run
`payment-sandbox.exe` from the extracted directory. Add that directory to
`PATH` if you want to invoke the command from any terminal.

## Configure the application

The application is configured through environment variables. The archive
contains `.env.example` as a reference; copy it and export its values before
starting the application:

```bash
cp .env.example .env
set -a
source .env
set +a
```

The available variables are:

| Variable | Default | Purpose |
| --- | --- | --- |
| `PAYMENT_SANDBOX_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn` or `error` |
| `PAYMENT_SANDBOX_LOG_FORMAT` | `text` | Log format: `text` or `json` |
| `PAYMENT_SANDBOX_HTTP_ADDRESS` | `:8080` | HTTP listen address |
| `PAYMENT_SANDBOX_ADMIN_TOKEN` | none | Required bearer token for `/admin/*` routes; use a long random value |
| `PAYMENT_SANDBOX_DATABASE_PATH` | `payment-sandbox.db` | SQLite database path |
| `PAYMENT_SANDBOX_DATABASE_BUSY_TIMEOUT` | `5s` | SQLite busy timeout |

The application does not load `.env` automatically. Export the variables as
shown above, or set them directly in the shell that starts the binary.

The admin token has no default. Replace the placeholder in `.env` with a
unique high-entropy secret. Send it as `Authorization: Bearer <token>` when
calling administrative or diagnostic routes.

## Start and check readiness

Start the application from the directory containing the database file:

```bash
payment-sandbox
```

The process stays running and writes startup information to the configured
output. In another terminal, check the health endpoints:

```bash
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
```

Both requests should return HTTP `200`. Stop the application with `Ctrl+C`.

## Make a first payment request

Create a payment with an identifier chosen by your test:

```bash
curl -X POST http://127.0.0.1:8080/payments \
  -H 'Content-Type: application/json' \
  -d '{"id":"demo-payment","amount":1000,"currency":"EUR"}'
```

Retrieve it and authorize it:

```bash
curl http://127.0.0.1:8080/payments/demo-payment

curl -X POST http://127.0.0.1:8080/payments/demo-payment/authorize
```

The payment identifier is application data; it is not a fixed value required
by the API. The endpoint returns JSON with the current payment state.

Register a webhook endpoint for a local simulation:

```bash
curl -X POST http://127.0.0.1:8080/webhook-endpoints \
  -H 'Content-Type: application/json' \
  -d '{"id":"demo-webhook","url":"https://example.test/payment-events"}'

curl http://127.0.0.1:8080/webhook-endpoints
```

## Data and database lifecycle

SQLite is stored at `PAYMENT_SANDBOX_DATABASE_PATH`. With the default
configuration, the file is `payment-sandbox.db` in the current working
directory. Parent directories must already exist and be writable.

The application creates the database schema on startup and keeps the database
open while it is running. Stop the process before moving, copying or deleting
the database. To start with a new simulation, stop the application and remove
the database file, then start it again.

Payment records and their event history are stored in SQLite. Webhook endpoint
registrations and other in-memory runtime state are reset when the process
stops.

For a separate test instance, use an explicit path:

```bash
PAYMENT_SANDBOX_DATABASE_PATH="$HOME/.local/share/payment-sandbox/test.db" \
payment-sandbox
```

## Troubleshooting

### The port is already in use

Choose another address and use the same address for your health and API
requests:

```bash
PAYMENT_SANDBOX_HTTP_ADDRESS=127.0.0.1:8081 payment-sandbox
curl http://127.0.0.1:8081/health/ready
```

### The binary cannot be executed

Download the artifact matching both your operating system and architecture.
On macOS or Linux, restore the executable bit if necessary:

```bash
chmod +x payment-sandbox
```

### The database cannot be opened

Check that the configured parent directory exists and is writable, and that
another Payment Sandbox process is not already using the same database.

### The artifact is unsupported

Do not install a Windows archive on macOS/Linux, or a macOS/Linux archive on
Windows. Check the operating system and architecture columns in the release
table and download a matching artifact.

### I want to modify the source

Published binaries are the supported end-user installation path. If you need
to change Go code, run tests or contribute to the project, follow the
[contributor getting started guide](getting-started.md) instead.
