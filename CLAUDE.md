# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OS-Baka is a bare-metal server provisioning and asset management platform. It uses PXE + iPXE chain-loading to fully automate OS installation (Ubuntu / Debian via Preseed), full-disk encryption (LUKS2 + TPM2), DHCP management, and node lifecycle. The system is a Go monolith (Gin + GORM + Postgres) with a React 19 SPA, plus a separate `pxe-services` container that runs dnsmasq + TFTP + iPXE.

Documentation in the repo (README.md and inline) is primarily in **Chinese**. New user-facing copy should follow the existing convention; backend logs/comments are English.

## Commands

### Backend (Go 1.24)

```bash
cd backend
go mod download
go run ./cmd/server                       # run server (loads ./config.yaml or ../config.yaml)
go test -v ./...                          # run all tests
go test -v ./internal/api -run TestName   # run a single test
golangci-lint run --timeout=5m            # lint (CI pins v1.64.8)
swag init -g cmd/server/main.go           # regenerate backend/docs/ Swagger after annotation changes
```

### Frontend (React 19 + Vite, Node 20)

```bash
cd frontend
npm install
npm run dev                  # vite dev server on :3000 (host 0.0.0.0)
npm run build                # production build
npm test                     # vitest run (use `-- --run` from CI)
npm run test:watch           # watch mode
npx tsc --noEmit             # type-check only
```

### Full stack via Docker Compose

```bash
cd docker
docker-compose up -d                         # dev: db + backend + frontend
docker-compose -f docker-compose.prod.yml up -d
```

`docker/.env` is required and must define `POSTGRES_PASSWORD` and `SECRET_KEY` (compose uses `:?` to hard-fail without them). Use `docker/.env.example` as a template.

## Architecture

### Component layout

```
frontend/  React 19 SPA → talks to backend over /api/v1 (Axios + JWT in localStorage)
backend/   Go monolith (Gin + GORM + Postgres) — single binary, all features in one process
pxe-services/  Separate container: dnsmasq (DHCP+TFTP) + iPXE firmware; chain-loads to backend
docker/    Compose orchestration (dev + prod variants)
config.yaml  Single YAML for backend; env vars override (highest precedence)
```

The backend serves the JSON API, the WebSocket SSH proxy, PXE-related HTTP endpoints (iPXE script / preseed / postinstall), and statics at `/tftp` (mapped to `/tftpboot`). The frontend is served separately (Vite in dev, nginx image in prod).

### Backend layering (`backend/internal/`)

- `cmd/server/main.go` — composition root. Order: load config → init DB (which runs migrations then seeds defaults) → `api.InitHandlers(db)` → synchronous one-time dnsmasq regen for initial on-disk state → `StartDnsmasqScheduler` (coalescing worker) → register middleware → register routes → `StartStaleNodeChecker` (background `active→offline` + `installing→error` watcher) → start HTTP server with timeouts → on SIGINT/SIGTERM: HTTP graceful shutdown then drain dnsmasq scheduler with 5s ctx.
- `config/` — YAML loader with env-var override. **Validates JWT secret on boot**: in `release` mode (`GIN_MODE=release`), a secret shorter than 32 chars or equal to the default `"change-this-in-production"` aborts startup.
- `model/` — GORM models + `InitDB`. `InitDB` opens Postgres, pings it (fatal on failure), configures the pool, runs `dbmigrate.Up()` (fatal on failure or dirty state), then seeds defaults (a `default` DHCP config and an `admin` user — password from `ADMIN_PASSWORD`, defaulting to `admin` with a warning). GORM models are query-time DTOs only; they do NOT drive schema.
- `dbmigrate/` — embedded golang-migrate runner. Migrations live in `internal/dbmigrate/migrations/NNNNNN_name.{up,down}.sql` and are baked into the binary via `embed.FS`. Apply locally with the `migrate` CLI for testing; the server applies them automatically on startup.
- LUKS passphrases live in `Node.EncryptionPassphrase` as AES-256-GCM ciphertext (`EncryptField`) — read via `DecryptField`. Same encryption applies to `IPMIPassword` and `RootPassword`. The preseed path uses `resolveNodePassphrase` which refuses to serve when the field is empty so the installer can't write garbage into LUKS.
- IPMI columns on `nodes` table are `ipmi_address` / `ipmi_username` / `ipmi_password` / `ipmi_allow_untrusted`. The model uses **explicit `gorm:"column:..."` tags** because GORM's snake_case converter mangles `IPMI` mid-acronym (would produce `ip_m_ipassword`). Don't remove the tags.
- `Node.InstallingStartedAt *time.Time` (column `installing_started_at`) is set when status enters `installing` and cleared when it leaves to any terminal state. Drives the install-timeout watcher. A partial index targets non-null rows only.
- `sysutil/` — host-level helpers (network interface discovery for DHCP UI).
- `api/` — HTTP layer. Handlers are stateless structs (`NewXxxHandler()`); the shared DB is a package-level global set by `InitHandlers(db)` and accessed via `getDB()`. Do not reach into `model.DB` directly from handlers — go through `getDB()` so tests can inject.

### Route groups (`api/routes.go`)

Four-tier routing — get the group right when adding endpoints:

1. **Health/info** (root level): `/health`, `/health/live`, `/health/ready` (DB-checking readiness), `/`, static `/tftp/`.
2. **Public** (`/api/v1/*`, no auth): `POST /auth/login` (with `LoginRateLimitMiddleware` — 5/min/IP), `GET /ws/ssh` (WebSocket SSH, auths via query token internally).
3. **PXE** (`/api/v1/pxe/*`, no JWT — booting machines have no credentials): `init`, `boot/:mac`, `preseed/:mac?t=<token>`, `postinstall/:mac?t=<token>`. **Sensitive**: preseed responses contain root passwords and LUKS passphrases in plaintext. Gated by per-install PXE token (see PXE flow section). Still wants network segmentation in production because the token-issuing `boot/:mac` is itself unauthenticated.
4. **Internal** (`/api/v1/internal/*`, `InternalAPIMiddleware`): allowed if the request either comes from a private/loopback IP (`127.`, `10.`, `192.168.`, `172.16-31.`, `::1`, `fc00:`, `fe80:`) **or** carries `X-Internal-Token` matching `cfg.Server.SecretKey`. Used by node post-install and heartbeat agents.
5. **Protected** (everything else under `/api/v1`): `AuthMiddleware` + `AuditMiddleware`. Per-route role gates via `RequireRole("admin")` or `RequireRole("admin","operator")`.

### Authentication and authorization

`AuthMiddleware` accepts **two token formats** in `Authorization: Bearer <token>`:

- Tokens starting with `osbaka_` → looked up as a service-account API key (`model.APIKey`, SHA-256 hash compared against `key_hash`, checks `is_active` and `expires_at`, updates `last_used_at`). Sets `userID` = creator, `username` = `"apikey:<name>"`, `role` = key's role.
- Anything else → JWT (HS256, signed with `cfg.Server.SecretKey`). Claims: `sub` (user id), `name`, `role`, `exp` (24h). Role defaults to `"operator"` for legacy tokens missing the claim.

Authenticated context is set via `c.Set("userID"|"username"|"role", …)`. Retrieve with `GetAuthUserID`, `GetAuthUsername`, `GetAuthRole`. The role taxonomy is **admin / operator / auditor** — `RequireRole` is the single chokepoint.

Sensitive fields (e.g., IPMI passwords) are encrypted at rest via AES-256-GCM in `api/crypto.go`. The key is read from `FIELD_ENCRYPTION_KEY` (hex-encoded, 32 bytes). If unset, fields are stored as plaintext with a warning — values prefixed with `enc:` indicate ciphertext.

### Audit logging

`AuditMiddleware` is applied to all protected routes — every successful POST/PUT/DELETE produces a coarse "what path with what status" entry. Handlers also call `WriteAuditLog(c, action, resource, resourceID, details)` explicitly for higher-fidelity events (status transitions, key rotation, IPMI power actions).

**Writes are synchronous** with a 2-second `context.WithTimeout`. Errors land in `slog.Error` (visible to operators) but never propagate to the caller — the user-facing action already succeeded by the time audit fires. The earlier fire-and-forget goroutine model is gone: it spawned N goroutines on bulk paths and silently lost failures.

Bulk paths use `WriteAuditLogs([]model.AuditLog)` which calls `CreateInBatches(logs, 100)` — one round-trip per 100 entries. Use this for any path that produces > ~5 entries in a single request (e.g. `BulkDelete`).

System-initiated entries (background checkers, no `gin.Context`) go through `writeSystemAuditLog(db, ...)` — same sync semantics, fills `username = "system"`.

**Destructive operations must snapshot before destroying.** `DeleteNode` and `BulkDelete` call `makeNodeDeleteSnapshot(&node)` which JSON-encodes `{id, hostname, mac_address, ip_address, asset_tag, group_id, status}` into the audit `details` column **before** the row goes away. Without this, audit logs would say "deleted node #123" with no way to know which physical machine that was. The snapshot shape is pinned by a test (`audit_snapshot_test.go`) — adding/removing fields is a contract change.

### Target disk selection (`api/pxe_disk.go`)

`Node.TargetDisk` is the device path partman installs onto. Two modes:

- **Operator-pinned** (e.g. `/dev/nvme0n1`): preseed emits `d-i partman-auto/disk string <path>`. Use when auto-detect can't disambiguate (multiple same-size disks, controllers with weird naming, data-disk-must-not-be-touched).
- **Empty (default)**: preseed emits a `partman/early_command` shell script that runs in the d-i initramfs BEFORE partman scans devices. Picks the largest non-removable non-read-only disk, preferring `nvme*` > `vd*` (virtio) > `sd*`. Result is fed back via `debconf-set partman-auto/disk`.

Script uses only busybox-available tools (`lsblk -dn -o NAME,TYPE,SIZE,RM,RO`, awk, sort, head, debconf-set). If no eligible disk is found, the script leaves debconf alone and partman will prompt — safer than picking wrong and erasing the wrong drive.

**Breaking change vs. earlier behavior**: previous releases hardcoded `/dev/sda` in both preseed blocks. Existing nodes created before this change with `target_disk = NULL` will get the new auto-detect logic on next rebuild. If the auto-detect picks a different device than `/dev/sda` (e.g. NVMe present), the install lands on a different disk than the previous run. Set `TargetDisk = "/dev/sda"` explicitly on legacy nodes if you need the old behavior.

API validation: `isValidDiskPath` regex `^/dev/[a-zA-Z0-9/_.-]+$` rejects shell metacharacters and missing `/dev/` prefix. Loose enough for `/dev/disk/by-id/...` paths; d-i validates the actual device existence at install time.

### PXE auto-discovery (`api/pxe_discover.go`)

When an unknown MAC hits `GET /pxe/boot/:mac`, the handler calls `discoverNode(db, mac, clientIP)` which inserts a `Node{Status: discovered, Hostname: "discovered-<suffix>", MACAddress, IPAddress: clientIP}`. The MAC's partial unique index on `nodes (mac_address) WHERE deleted_at IS NULL` makes concurrent discovers race-safe: the loser catches the insert error, re-fetches the existing row, and treats that as a no-op (no duplicate audit entry).

The `discovered` status is **distinct from `pending`** specifically so BootScript can refuse to install discovered nodes. Switch in `BootScript`:

- `status == "installing" || status == "pending"` → real PXE install (legacy + operator-registered paths)
- `status == "discovered"` → test menu with "awaiting operator approval" text; default to `local` boot so a discovered machine that's already provisioned with another OS doesn't get stuck in a re-discover loop
- `status == "active"` → boot local disk
- `status == "maintenance"` → drop to iPXE shell
- node not found AND auto-discover disabled → generic test menu

Promotion path: operator sees the new row in the UI inventory, edits OS / encryption / IPMI config, clicks Rebuild → `discovered → installing` via `SetNodeStatus` (the status machine permits this), then the IPMI auto-cycle (see below) fires the next boot.

Disable via env `PXE_AUTO_DISCOVER=false` for shared L2 networks where stray PXE devices would pollute inventory. Typo-safe (only literal `false/0/no/False/FALSE/No` disable; anything else stays on).

### IPMI auto power-cycle on rebuild (`api/ipmi.go`)

`POST /nodes/:id/rebuild` and `POST /nodes/bulk/rebuild` mark the node(s) `installing` and then asynchronously fire `ipmitool power cycle` against the BMC. Without this, the operator would set status → installing → walk to the rack → power cycle → wait. With it, real zero-touch rebuilds.

Building blocks:
- `buildIpmitoolArgs(node, subcommand...)` is the single source of truth for the argv (lanplus, `-N 5 -R 1` for bounded network behavior, `-H/-U/-P`, optional `-C 0` for untrusted ciphers, then the subcommand). Used by `PowerAction`, `TestIPMI`, and the auto-cycle. Don't reimplement.
- `runIPMICommand(ctx, args)` is a package-level `var` so tests can stub. Real impl: `exec.CommandContext("ipmitool", args...).CombinedOutput()` + strip whitespace.
- `TriggerPowerCycle(node)` returns immediately with one of: `scheduled`, `skipped:no_bmc`, `disabled`. The actual cycle runs in a goroutine bounded by `powerCycleSem` (capacity 8 by default, env `IPMI_POWER_CYCLE_CONCURRENCY`). Each invocation has a 30s `context.WithTimeout`.
- `TriggerPowerCycleBatch(nodes)` is the bulk counterpart. Returns a tally `{outcome: count}` for the HTTP response.
- Audit trail: `ipmi.power_cycle_scheduled` is implicit (the rebuild audit covers it); `ipmi.power_cycle_skipped` / `ipmi.power_cycle_ok` / `ipmi.power_cycle_error` record each goroutine's outcome via `writeSystemAuditLog`. Caller never blocks on the audit row.
- Opt-out: env `IPMI_AUTO_POWER_CYCLE=false` disables globally; query `?no_power_cycle=1` on rebuild endpoints disables per-call. Unknown env values keep the feature ON (typo-safe convention shared with `PXE_REQUIRE_TOKEN`).
- **Runtime requirement**: `ipmitool` must be on PATH. The backend Dockerfile installs it; local dev without ipmitool will see `executable file not found` in the audit log (the rebuild itself still succeeds — only the cycle fails).

### Node status machine (`api/node_status.go`)

Seven legal statuses: `discovered / pending / installing / active / maintenance / error / offline`. Use the `NodeStatus*` constants — not raw strings. `discovered` is set ONLY by the PXE auto-discover path; everything else can be reached via operator action or background watchers.

`SetNodeStatus(c, db, node, target, reason)` is the **single chokepoint** for status mutation:
- Validates `target` is known via `IsKnownNodeStatus`.
- Checks the source→target edge against the `nodeStatusTransitions` map. Self-transitions are a no-op (don't churn timestamps / audit log on retries).
- Updates `installing_started_at` automatically (set on entering `installing/pending`, null on exit).
- Persists via `db.Model(node).Updates(map[string]any{...})` so the timestamp can be nulled (Save with a nil `*time.Time` wouldn't).
- Writes an audit entry `<from> -> <to>: <reason>` (or `<new> -> <to>: …` for empty source).
- Returns `ErrIllegalTransition` on illegal edges; callers translate that to HTTP 400 (operator input) or 409 (rebuild on in-progress install).

Direct `node.Status = "x"` assignments scattered through the codebase are being migrated incrementally. The following call sites have been converted: `UpdateNodeStatus` (postinstall callback), `RebuildNode`, `CheckStaleNodes`, `CreateNode` (initial status default + validation). Bulk paths (`BulkRebuild`, `CheckStaleNodes`, `CheckInstallTimeout`) keep raw `UPDATE` for throughput but emit a single coarse audit entry and maintain timestamps explicitly.

Background watchers (`heartbeat.go:StartStaleNodeChecker`) run on one ticker with two checks:
- `CheckStaleNodes(staleMinutes)`: `active` nodes with `last_heartbeat < threshold` → `offline`.
- `CheckInstallTimeout(timeoutMinutes)`: `installing` nodes with `installing_started_at < threshold` → `error`, clears timestamp.

Heartbeat from a node **only** promotes status when the current status is `active`/`offline`. Nodes in `installing`/`maintenance`/`pending` keep their status — heartbeat just updates the health metrics. This stops chatty agents from racing the install-timeout watcher or unlocking a maintenance hold.

### dnsmasq config scheduler (`api/dnsmasq_scheduler.go`)

`GenerateDnsmasqConfig()` rewrites `/etc/dnsmasq.d/00-main.conf` and `/etc/dnsmasq.d/01-hosts.conf` from the DB and pokes `.reload`. The scheduler wraps it with leading-edge + trailing-edge coalescing so bursts of mutations (CSV import, bulk delete, rapid UI edits) collapse to ≤ 2 regenerations:

- `ScheduleDnsmasqRegen()` is non-blocking — a 1-slot dirty channel signals the worker.
- Worker runs immediately on first dirty, then runs **once more** if anyone scheduled during the run, then idles. 100 concurrent calls produce exactly 2 runs.
- 200 ms cooldown between starts caps thrash under pathological loops.
- Errors recorded in `LastRun / LastError / RunCount`; exposed via `DnsmasqStats()` and the dashboard summary.
- `StopDnsmasqScheduler(ctx)` from graceful shutdown waits for in-flight regen before exit.

**Call site routing**: every mutation that changes desired dnsmasq state uses `ScheduleDnsmasqRegen()` (non-blocking, errors surfaced via dashboard tile). The single synchronous exception is `POST /api/v1/dhcp/service/restart` — that's an operator-explicit action and must return the real error if the regen failed.

### PXE flow

Boot lifecycle, end-to-end:

1. Node powers on → `dnsmasq` (in `pxe-services`) hands out an IP + boot file (`undionly.kpxe` for BIOS, `ipxe.efi` for UEFI).
2. iPXE chain-loads `GET /api/v1/pxe/init`, which redirects to `GET /api/v1/pxe/boot/:mac`.
3. `boot/:mac` looks up the node by MAC. If it's in a provisioning state (`installing`/`pending`), the handler **mints a fresh PXE token** (`IssuePXEToken`), invalidates any prior unconsumed tokens for that node, and embeds `?t=<token>` into the preseed URL inside the returned iPXE script. Status is stamped with `installing_started_at = now()` via the status machine.
4. Installer fetches `GET /api/v1/pxe/preseed/:mac?t=<token>` — token is **validated but not consumed** (the installer may re-fetch on retry). IP binding is NOT enforced at this step because anaconda/debian-installer may egress on a different NIC than the iPXE phase. Preseed contains the root password (decrypted via `DecryptField`), LUKS passphrase (decrypted via `resolveNodePassphrase`), mirror URL, timezone, SSH settings.
5. PostInstall script downloaded via `GET /api/v1/pxe/postinstall/:mac?t=<token>` — token is **validated AND consumed** (`ConsumePXEToken`), AND IP binding IS enforced (this is the request from the booted node on the same network it minted on).
6. After first boot the node runs the postinstall script which hits `PUT /api/v1/internal/nodes/:id/status` with `{"status":"active"}` — this goes through `SetNodeStatus` which checks `installing → active` is legal, clears `installing_started_at`, and writes an audit entry. The callback is retried 5× × 10s before giving up; if all attempts fail the script keeps itself + the systemd unit on disk so the next boot retries automatically (`buildPostinstallScript` in `pxe_postinstall_script.go`).

Token TTL defaults to 2 hours (`PXE_TOKEN_TTL_MINUTES`). Disabling the gate (`PXE_REQUIRE_TOKEN=false`) is supported for migration but not recommended — without it, anyone who can ARP onto the PXE network and spoof a MAC can fetch preseed and steal LUKS passphrases.

Mirror URL precedence in `resolveMirror` (`pxe.go`): per-node `MirrorURL` → `PXE_MIRROR_URL` env → `PXE_UBUNTU_MIRROR_URL` / `PXE_DEBIAN_MIRROR_URL` env → active DHCPConfig mirror → distro default.

Security archive is **separate** — `resolveSecurityMirror` follows the same precedence with its own field (`SecurityMirrorURL` on both Node and DHCPConfig) and env (`PXE_DEBIAN_SECURITY_MIRROR_URL`, `PXE_UBUNTU_SECURITY_MIRROR_URL`). The earlier code shared the main mirror's host with security and just appended `-security` to the path, producing `http://deb.debian.org/debian-security` which 404s. Intranet operators set `DHCPConfig.security_mirror_url` to `http://mirror.intra/debian-security` (parallel to `MirrorURL = http://mirror.intra/debian`).

If an `installing` node never sends the postinstall callback, the background `CheckInstallTimeout` watcher flips it to `error` after `INSTALL_TIMEOUT_MINUTES` (default 120).

### DHCP integration

`api/dnsmasq.go` regenerates `/etc/dnsmasq.d/00-main.conf` + `/etc/dnsmasq.d/01-hosts.conf` from the DB (active `DHCPConfig` + `DHCPReservation` rows). Mutations to nodes / DHCP configs / reservations call `ScheduleDnsmasqRegen()` (non-blocking, coalesced via the scheduler — see above). Editing dnsmasq config files by hand will be overwritten — always change state through the API or DB.

After writing both configs atomically (tempfile + rename), the regen function pokes `/etc/dnsmasq.d/.reload` as a signal. `pxe-services/start.sh` runs a 2-second poll loop that, on seeing the trigger, removes it and `kill -HUP $dnsmasq_pid`. The trigger is written **after** both config files are renamed into place — earlier code wrote it from inside `generateMainConfig` and races the hosts-config write that followed it. Manual sync escape hatch: `POST /api/v1/dhcp/service/restart` runs regen synchronously and surfaces the error.

When `/etc/dnsmasq.d/` does not exist (dev mode, no bind mount), `GenerateDnsmasqConfig` returns nil and logs `slog.Info` once — the Dashboard scheduler tile stays green instead of flagging permanent red for an environment that isn't expected to serve PXE.

### Frontend

- `src/router.tsx` — `createBrowserRouter` with `ProtectedRoute` (checks `localStorage.access_token`, redirects to `/login` if missing).
- `src/services/apiClient.ts` — single `axios` instance. Request interceptor injects Bearer token; response interceptor on 401 clears the token and redirects to `/login`. Errors are normalized via `HttpClient.toApiError` — backend returns `{ "error": "msg" }`.
- `src/services/backendService.ts` — typed wrappers for each API group; new endpoints go here, not directly in components.
- `src/services/geminiService.ts` — optional Google Gemini integration for AI-assisted CSV import analysis (`VITE_GEMINI_API_KEY`).
- Pages map 1:1 with sidebar nav (`Dashboard`, `Nodes`, `KeyVault`, `WebShell`, `DHCPSettings`, `AuditLogs`, `Notifications`, `Documentation`, `Settings`). WebShell uses `xterm.js` + WebSocket against `/api/v1/ws/ssh`.
- Dashboard `/dashboard/summary` response includes `dnsmasq_last_run` / `dnsmasq_last_error` / `dnsmasq_run_count` — render these to surface scheduler-side failures.
- IPMI fields on `Node` update: backend uses pointer-receiver request structs so PUT can do partial updates. `ipmi_password` semantics: omit field = leave ciphertext alone; empty string = clear; non-empty = encrypt and store. Don't accidentally send `""` from the form when the user didn't touch the field.
- Build splits vendor chunks: `vendor-react`, `vendor-ui` (lucide), `vendor-xterm`.

## Configuration precedence

`config.yaml` (searched at `./config.yaml`, `../config.yaml`, `backend/config.yaml`) loads first; then env vars override. Critical envs:

- **Server / DB**: `DATABASE_URL`, `PORT`, `SECRET_KEY` (≥32 chars in release mode), `GIN_MODE`, `ADMIN_PASSWORD`.
- **At-rest encryption**: `FIELD_ENCRYPTION_KEY` (hex-encoded 32 bytes / 64 hex chars). Without it, LUKS passphrases / IPMI / root passwords store as plaintext with a startup warn.
- **PXE**: `EXTERNAL_IP`, `PXE_MIRROR_URL`, `PXE_UBUNTU_MIRROR_URL`, `PXE_DEBIAN_MIRROR_URL`, `PXE_UBUNTU_SECURITY_MIRROR_URL`, `PXE_DEBIAN_SECURITY_MIRROR_URL`, `PXE_ASSETS_DIR`.
- **PXE security**: `PXE_REQUIRE_TOKEN` (default true), `PXE_TOKEN_TTL_MINUTES` (derived from `INSTALL_TIMEOUT_MINUTES × 2` when unset; explicit value wins).
- **Lifecycle watchers**: `INSTALL_TIMEOUT_MINUTES` (default 120; was 60). Token TTL follows by default so operators only tune one knob.
- **IPMI auto-cycle**: `IPMI_AUTO_POWER_CYCLE` (default true), `IPMI_POWER_CYCLE_CONCURRENCY` (default 8).
- **PXE auto-discovery**: `PXE_AUTO_DISCOVER` (default true).
- **Frontend**: `VITE_API_URL`, `VITE_GEMINI_API_KEY`.

## CI

`.github/workflows/ci.yml` runs on push/PR to `main` and weekly cron. Job order: `security` (golangci-lint v1.64.8, gosec, govulncheck, Trivy) → `backend` (build + test) and `frontend` (build + `npm test -- --run`) in parallel → `docker` (builds all three images, no push). The security job is a gate — keep lint clean and don't introduce new vulns flagged by govulncheck/gosec.

## Notes for changes

- New protected endpoint: add the route in the appropriate `registerXxxRoutes` in `api/routes.go`, gate with `RequireRole` if needed, use `getDB()` (not `model.DB`). Add Swagger annotations and regenerate `backend/docs/` via `swag init`.
- New persisted field: add it to `model/models.go` AND write a migration pair under `internal/dbmigrate/migrations/`. The model struct alone does NOT change schema. Test the round trip:
  ```
  migrate -path backend/internal/dbmigrate/migrations -database "$DATABASE_URL" up
  migrate -path backend/internal/dbmigrate/migrations -database "$DATABASE_URL" down 1
  ```
  Migration files are embedded at build time — a new migration requires a recompile, not a release-time copy.
- New secret/credential at rest: run plaintext through `EncryptField` on write and `DecryptField` on read — never persist plaintext to model fields.
- LUKS / TPM logic: passphrase generation uses `crypto/rand`; rotation lives in `nodes.go` (`RotatePassphrase`) and writes AES-GCM ciphertext via `EncryptField` plus an audit log entry.
- Status mutation: use `SetNodeStatus(c, db, node, target, reason)` rather than `node.Status = "x"; db.Save(&node)`. The helper enforces the transition map, maintains `installing_started_at`, and writes an audit entry. Bulk paths may use raw `UPDATE` for throughput but must keep the timestamp consistent.
- Adding a new status value: update `NodeStatus*` constants, `IsKnownNodeStatus`, `nodeStatusTransitions` (both as a source key AND in target lists where appropriate), and the test in `node_status_test.go` — those tests are the policy spec.
- Destructive endpoint (delete row, drop file, revoke key): capture a snapshot of the destroyed entity in the audit `details` BEFORE the destruction. See `makeNodeDeleteSnapshot` in `audit.go` for the pattern.
- Mutating dnsmasq state: call `ScheduleDnsmasqRegen()` (non-blocking, coalesced). Synchronous `GenerateDnsmasqConfig()` is reserved for startup and explicit `POST /dhcp/service/restart`. If you find yourself reaching for sync regen elsewhere, you probably want the scheduler.
- New field on `Node` that uses an acronym (IPMI / SSH / TPM / etc.): add an explicit `gorm:"column:..."` tag — GORM's snake_case converter will mangle multi-letter acronyms otherwise.
