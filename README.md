# spark

A small GitHub webhook server that runs per-project deploy scripts.

It listens for GitHub webhooks, verifies the signature, figures out which
repository the event is for, and runs the scripts configured for that project.

## Requirements

- Go 1.26+
- `bash` available on PATH (scripts are run with `bash <script>`)

## Configuration

All configuration lives under `$HOME/.spark`.

### `$HOME/.spark/config.json`

```json
{
  "port": 8010,
  "secret": "your-github-webhook-secret"
}
```

- `port` — TCP port to listen on (default `8010`).
- `secret` — GitHub webhook secret. Used to verify the
  `X-Hub-Signature-256` HMAC-SHA256 header. If empty, verification is skipped.

### Per-project folders

Each project gets its own folder under `$HOME/.spark`, named after the GitHub
repository (`repository.name` in the webhook payload). For example, repo
`tianyl1984/spark` maps to `$HOME/.spark/spark/`.

Fixed file names inside a project folder:

| File         | Purpose                                                                 |
| ------------ | ----------------------------------------------------------------------- |
| `execute.sh` | **Required.** The script to run when a webhook arrives.                 |
| `workdir`    | Working directory for the scripts. Falls back to the project folder.    |
| `success.sh` | Run after `execute.sh` succeeds. Skipped if missing or empty.           |
| `fail.sh`    | Run after `execute.sh` fails. Skipped if missing or empty.              |

The combined stdout+stderr of `execute.sh` is passed to `success.sh` / `fail.sh`
as their **first argument** (`$1`).

Scaffold a new project's empty files with:

```bash
spark create <repo-name>      # creates $HOME/.spark/<repo-name>/{execute.sh,workdir,success.sh,fail.sh}
```

Existing files are never overwritten. Then fill in `execute.sh` and `workdir`.

Example layout:

```
$HOME/.spark/
├── config.json
└── spark/
    ├── execute.sh      # e.g. git pull && make build && systemctl restart spark
    ├── workdir         # e.g. /srv/spark
    ├── success.sh      # e.g. notify "$1"
    └── fail.sh         # e.g. alert "deploy failed: $1"
```

## GitHub setup

In the repo's **Settings → Webhooks**, add:

- Payload URL: `http://<host>:8010/`
- Content type: `application/json`
- Secret: same value as `config.json` `secret`

## Run

```bash
make build      # build to bin/spark
make run        # run from source
./bin/spark     # run the built binary
```

## Install as a service (Linux + systemd)

`install.sh` builds the binary, installs it to `~/.local/bin/spark`, and
registers a **system-wide systemd service** (`/etc/systemd/system/spark.service`)
that **starts automatically at boot**. The service runs as your user (so the
`$HOME/.spark` config still applies) and logs to `$HOME/.spark/spark.log`.

```bash
./install.sh          # run as your normal user; it uses sudo where needed
```

Manage it with:

```bash
sudo systemctl status spark
sudo systemctl restart spark
tail -f ~/.spark/spark.log     # or: sudo journalctl -u spark -f
```

Module path: `github.com/tianyl1984/spark`
