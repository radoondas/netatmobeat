# Netatmobeat Operational Runbook

## Authentication Overview

Netatmobeat uses Netatmo's OAuth2 API with refresh tokens. Key behaviors:

- **Token rotation**: Netatmo invalidates the old refresh token on every use (since May 2024). The beat persists rotated tokens to a file after each refresh.
- **Automatic refresh**: A background goroutine refreshes the access token before expiry (~60s before the `expires_in` window).
- **Retry with backoff**: On transient failures (network errors, HTTP 5xx), the beat retries with exponential backoff (60s → 120s → 240s, capped at 15 minutes).
- **Terminal failure detection**: On permanent auth failures (`invalid_grant`, `invalid_client`), the beat stops retrying and logs re-bootstrap instructions.

## Validating Configuration

### Config syntax check

```bash
./netatmobeat test config -c netatmobeat.yml
```

Validates config file parsing. Prints `Config OK` on success.

### Auth validation

Start the beat normally. Authentication is validated at startup:

```bash
./netatmobeat -c netatmobeat.yml -e
```

If credentials are invalid, the beat exits immediately with a clear error. If authentication succeeds, the beat starts collecting data.

For a quick auth check without running indefinitely, start the beat, wait for the first data collection cycle, then stop with Ctrl-C.

## Common Operations

### Setting up index templates (required before first run)

Netatmobeat requires composable index templates to be loaded into Elasticsearch before indexing data. Without templates, date fields (e.g., `last_message`, `last_seen`) and geo_point fields will not be mapped correctly.

```bash
curl -X PUT -u elastic "localhost:9200/_index_template/netatmobeat-publicdata" \
  -H 'Content-Type: application/json' -d @netatmobeat.template.publicdata.json

curl -X PUT -u elastic "localhost:9200/_index_template/netatmobeat-stationdata" \
  -H 'Content-Type: application/json' -d @netatmobeat.template.stastiondata.json
```

Adjust the URL and authentication for your environment (e.g., HTTPS, API keys, Elastic Cloud).

The template files are included in the release package. You only need to load them once per cluster (or again after template changes in a new release).

### First-time bootstrap (obtaining a refresh token)

1. Go to [https://dev.netatmo.com/apps/](https://dev.netatmo.com/apps/) and select your application
2. In the **Token Generator** section, select scope `read_station`
3. Click **Generate Token** and authorize
4. Copy the **refresh token** into `netatmobeat.yml`:

```yaml
netatmobeat:
  client_id: "your_client_id"
  client_secret: "your_client_secret"
  refresh_token: "the_refresh_token_from_step_3"
  token_file: "netatmobeat-tokens.json"
```

5. Start the beat — it will exchange the refresh token for an access token and persist the rotated tokens to the token file.

After the first successful refresh, the `refresh_token` in config is no longer needed (the beat reads from the token file on restart). You may leave it in config as a fallback.

### Re-bootstrap after token loss or expiry

**When to do this**: The beat logs a terminal error like:
```
refresh token abcd*** is invalid or expired (invalid_grant).
Re-authorization required: obtain a new token from https://dev.netatmo.com/apps/
```

**Steps**:
1. Repeat the [first-time bootstrap](#first-time-bootstrap-obtaining-a-refresh-token) steps to get a new refresh token
2. Update `refresh_token` in `netatmobeat.yml`
3. Delete the old token file (if it exists): `rm netatmobeat-tokens.json`
4. Restart the beat

### Rotating the Netatmo app client secret

1. Go to [https://dev.netatmo.com/apps/](https://dev.netatmo.com/apps/)
2. Regenerate the client secret for your app
3. Update `client_secret` in `netatmobeat.yml`
4. **Important**: Regenerating the client secret does NOT invalidate existing refresh tokens. No re-bootstrap is needed — just update the config and restart.

### Recovering from a corrupted or missing token file

**Symptoms**: The beat fails on startup with:
```
token file loaded but refresh failed: ...
```

Or the token file is empty/corrupted JSON.

**Steps**:
1. Delete the token file: `rm netatmobeat-tokens.json`
2. Ensure `refresh_token` is set in `netatmobeat.yml` (from your last bootstrap)
3. Restart the beat — it will fall back to the config refresh_token
4. If the config refresh_token is also expired, do a [full re-bootstrap](#re-bootstrap-after-token-loss-or-expiry)

### Checking token file health

The token file is JSON with this structure:

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "expires_in": 10800,
  "obtained_at_unix": 1700000000,
  "scope": ["read_station"]
}
```

Check the `obtained_at_unix` field to see when the last successful refresh occurred:

```bash
python3 -c "import json,datetime; d=json.load(open('netatmobeat-tokens.json')); print(datetime.datetime.utcfromtimestamp(d['obtained_at_unix']))"
```

If the timestamp is older than 6 months, the refresh token may have expired (Netatmo's exact expiry policy is not publicly documented, but tokens do expire after extended inactivity).

## Docker / Kubernetes

### Volume requirements

The token file **must** be on a persistent volume. Without persistence, every container restart requires re-bootstrap.

```yaml
# docker-compose example
volumes:
  - netatmobeat-data:/data

# netatmobeat.yml
netatmobeat:
  token_file: "/data/netatmobeat-tokens.json"
```

### Pod restart behavior

On restart, the beat:
1. Reads the token file from the persistent volume
2. Attempts a refresh to get a fresh access token
3. If the refresh fails (token expired during downtime), logs a terminal error and exits

If pods restart frequently (CrashLoopBackOff), check:
- Is the persistent volume mounted correctly?
- Are the tokens still valid? (Check logs for `invalid_grant`)
- Is there network connectivity to `api.netatmo.com`?

### Startup validation

The beat validates the token file path is writable at startup. If the volume is not mounted or is read-only, you'll see:

```
token file path validation failed: token file directory /data is not writable
```

Fix: ensure the volume is mounted with write permissions.

## Key Log Messages

### Informational (normal operation)

| Message | Meaning |
|---------|---------|
| `Loaded tokens from file: ...` | Successfully loaded tokens from disk on startup |
| `Token refreshed successfully. Expires in: 10800s` | Normal token refresh |
| `Using refresh_token from config for initial authentication.` | First run — bootstrapping from config |

### Warnings (investigate if recurring)

| Message | Meaning | Action |
|---------|---------|--------|
| `username/password config fields are deprecated and ignored` | Old config fields present | Remove `username`/`password` from config |
| `Station data request got auth error (401)...` | Access token was rejected, forcing refresh | Normal if occasional — Netatmo may invalidate tokens early. Investigate if constant. |
| `Failed to persist rotated tokens to ... (attempt N)` | Token file write failed | Check disk space and file permissions |
| `Terminal auth error, but token was rotated by another goroutine` | Concurrent refresh race resolved | Normal — another goroutine succeeded |

### Errors (action required)

| Message | Meaning | Action |
|---------|---------|--------|
| `refresh token abcd*** is invalid or expired (invalid_grant)` | Refresh token permanently invalidated | [Re-bootstrap](#re-bootstrap-after-token-loss-or-expiry) |
| `Terminal authentication failure: ... Stopping refresh loop` | Auth cannot recover | [Re-bootstrap](#re-bootstrap-after-token-loss-or-expiry) |
| `Token persistence has failed N consecutive times` | Token file not being written | Check file permissions and disk space. If the process crashes, re-bootstrap will be needed. |
| `token file path validation failed` | Token file directory not writable at startup | Fix directory permissions or volume mount |
| `client_id is required` / `client_secret is required` | Missing credentials in config | Add the required fields to `netatmobeat.yml` |
| `no authentication tokens available` | No token source configured | Follow [first-time bootstrap](#first-time-bootstrap-obtaining-a-refresh-token) |

## Monitoring

### Auth health indicators

The beat tracks these internally (visible in debug logs):
- **Last successful refresh timestamp** — stale values indicate refresh problems
- **Consecutive refresh failures** — increasing count indicates connectivity or auth issues
- **Consecutive persist failures** — increasing count indicates disk/permission issues

Enable debug logging to see detailed auth state:

```bash
./netatmobeat -c netatmobeat.yml -e -d "netatmobeat"
```

### External monitoring

Monitor the beat's output to Elasticsearch. If events stop arriving:
1. Check if the beat process is still running
2. Check logs for terminal auth errors
3. Check if the token file's `obtained_at_unix` is recent
