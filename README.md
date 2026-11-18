
# webhook-relay

Small Go service that forwards git webhook payloads (GitHub, Gitea, GitLab) to configurable targets: HTTP endpoints, Slack channels or email digests. Ships as a single binary with a YAML config.

## Features

- Receives push, tag and release events from git hosts
- Filters events by repository and branch
- Retries failed deliveries with a backlog
- Exposes a /health endpoint for monitoring

## Installation

Download the binary from the releases page, or build from source:

```bash
go build ./cmd/webhook-relay
```

## Usage

```bash
webhook-relay --config relay.yaml
```

## License

MIT, based on github.com/hayat-se/webhook-relay.
