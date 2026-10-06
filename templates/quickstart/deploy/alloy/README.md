# Alloy log shipping

One Alloy per server ships every container's stdout to that server's
Grafana Cloud Loki. Setup, credentials and Grafana linking:
[docs/OBSERVABILITY.md → 日志存储](../../../../docs/OBSERVABILITY.md#日志存储).

```bash
LOKI_URL=... LOKI_USERNAME=... LOKI_PASSWORD=... docker compose up -d
```

Validate edits before deploying: `alloy fmt config.alloy && alloy validate config.alloy`.
