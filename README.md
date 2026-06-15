# Villa73 - Home Dashboard

Being rebuilt - again. Now as a monorepo including the Go backend

- frontend/ Contains the React web solution
- backend/ Contains the Go backend

## Raspberry Pi hostname setup (mDNS / `<hostname>.local`)

By default, a Raspberry Pi running Ubuntu Server will **not** be discoverable by hostname on the local network — even if `/etc/hostname` is set correctly. Without mDNS, other devices on the network have no way to resolve the hostname to an IP address unless a local DNS server or router provides that mapping.

### What's needed

Two packages must be installed on the Pi:

| Package | Role |
|---|---|
| `avahi-daemon` | Runs a background service that broadcasts the hostname as `<hostname>.local` over mDNS (Multicast DNS / Bonjour) on the local network |
| `libnss-mdns` | Extends the Linux name resolution stack (NSS) to resolve `.local` names via mDNS, and automatically updates `/etc/nsswitch.conf` |

### Installation

```bash
sudo apt-get install -y avahi-daemon libnss-mdns
```

This will:
1. Install and start `avahi-daemon` (enabled on boot automatically)
2. Install `libnss-mdns` and patch `/etc/nsswitch.conf` to include `mdns4_minimal [NOTFOUND=return]` in the `hosts:` line

### Verify it's working

Check the daemon is running and broadcasting the right name:

```bash
systemctl status avahi-daemon
```

You should see a line like:
```
"avahi-daemon: running [<hostname>.local]"
```

And `/etc/nsswitch.conf` should have:
```
hosts: files mdns4_minimal [NOTFOUND=return] dns
```

### Connecting from other devices

| Client OS | `<hostname>.local` | Notes |
|---|---|---|
| macOS / iOS | Works out of the box | Bonjour is built in |
| Linux | Works after installing `avahi-daemon` + `libnss-mdns` on the client too | Same setup as the Pi |
| Windows | Requires [Bonjour for Windows](https://support.apple.com/kb/DL999) (installed with iTunes) | Or use the IP address |

Once set up, you can use:

```bash
# SSH by name
ssh <user>@<hostname>.local

# Browser
http://<hostname>.local      # port 80
https://<hostname>.local     # port 443
```

### Why bare hostname (without `.local`) doesn't reliably work

The `.local` suffix is what triggers mDNS resolution. Without it, name resolution falls through to DNS, which your home router likely doesn't answer for local hostnames. Use `<hostname>.local` consistently for reliable results across all clients. Alternatively, configure your router's local DNS (if it supports it) or add an entry to `/etc/hosts` on each client machine.

### SSH config tip

Add this to `~/.ssh/config` on your client machine to avoid typing `.local` every time:

```
Host <hostname>
    HostName <hostname>.local
    User <user>
```

Then `ssh <hostname>` will work from that machine.

---

## Node.js version policy

- Frontend local and Docker builds are pinned to `22.22.0`.
- `22.22.0` is used to keep Raspberry Pi `armv7` builds working.
- Node 24 images do not support `linux/arm/v7` (`armv7l`) in this setup, which causes Docker build failures on the Pi.

## Raspberry Pi MCP server

This repo includes a standalone Go MCP server for Raspberry Pi access at `tools/mcp-pi/main.go`.

- Run it from the repo root with `make run-pi-mcp`
- Or run `./scripts/run-pi-mcp.sh`
- It uses `ssh` on your machine and speaks MCP over stdio
- It runs on your laptop, not on the Raspberry Pi
- Default SSH target is `192.168.10.217`

For agent workflows where you want one pre-approved command prefix instead of raw JSON-RPC payloads, use `./scripts/mcp-pi-tool.sh`. It is a thin local client that starts the MCP server on demand, initializes it, performs a standard tool call, and prints the result.

Supported tools:

- `docker_compose_logs`
  - Reads `docker compose logs` for one of two configured solution targets on the laptop side
  - Supports `target` as `dir1` or `dir2`, plus optional `composeFile`, `services`, `tail`, `since`, `timestamps`
- `compose_service_api_request`
  - Runs `curl` from the Pi host only against a Docker Compose service declared in one of the configured targets
  - Resolves the published host port with `docker compose port <service> <containerPort>`
  - Supports `target` as `dir1` or `dir2`, plus `service`, `containerPort`, optional `path`, `method`, `headers`, `body`, `scheme`, `timeoutSeconds`

This design prevents arbitrary remote URL calls and arbitrary Compose project selection. The laptop-side launcher reads the Pi host, user, and the two allowed Compose directories from `.env`, and the server rejects any target other than `dir1` or `dir2`.

Supported SSH environment variables:

- `MCP_PI_SSH_TARGET`
  - Example: `pi@192.168.10.217`
  - You can also point this at an SSH config host alias if your certificate auth already lives there
- `MCP_PI_SSH_PORT`
- `MCP_PI_SSH_IDENTITY_FILE`
- `MCP_PI_SSH_CERTIFICATE_FILE`
- `MCP_PI_SSH_KNOWN_HOSTS_FILE`

Example MCP client command:

```json
{
  "command": "./scripts/run-pi-mcp.sh",
  "cwd": "/path/to/villa73",
  "env": {
    "MCP_PI_SSH_CERTIFICATE_FILE": "/path/to/pi-cert.pub",
    "MCP_PI_SSH_IDENTITY_FILE": "/path/to/pi-key"
  }
}
```

Most MCP clients start the server process on demand. In practice that means if your client is configured to use `scripts/run-pi-mcp.sh`, asking something like "how are my services doing?" will cause the client to launch the MCP server locally and then call its tools. No backend services are started on your laptop.

If your environment does not invoke MCP servers natively and instead asks to run shell commands, prefer pre-approving `scripts/mcp-pi-tool.sh` as the single command prefix. Example standardized requests:

```sh
./scripts/mcp-pi-tool.sh logs \
  --target dir1 \
  --service scheduler \
  --tail 120 \
  --since 36h \
  --timestamps
```

```sh
./scripts/mcp-pi-tool.sh api \
  --target dir2 \
  --service scheduler \
  --container-port 6002 \
  --path /health
```
