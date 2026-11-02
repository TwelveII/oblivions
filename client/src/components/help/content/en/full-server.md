# Full server (everything yourself)

Run the entire Oblivions platform on your own server — accounts, messages, files, and voice — fully independent of your Oblivions server.

## Requirements

- Linux server (Ubuntu 22.04+ / Debian 12+), x86_64 or arm64
- 2 vCPU, 4 GB RAM minimum
- A domain is **optional** — without one the installer uses a free `sslip.io` hostname so HTTPS works (browsers block voice/camera over plain HTTP)

## One-command install

SSH into your server and run:

```bash
curl -fsSL https://raw.githubusercontent.com/akinalpfdn/Mqvi/main/deploy/install.sh | sudo bash
```

You'll be asked how to expose Oblivions:

1. **Your own domain** (recommended) — sets up Caddy + Let's Encrypt automatically
2. **sslip.io** — no domain needed; HTTPS still works out of the box
3. **HTTP only** — testing only; browsers will block mic, camera, and screen-share

Prefer a non-interactive run? Pass flags:

```bash
sudo bash install.sh --domain demo.example.com --port 9092 -y   # your own domain
sudo bash install.sh -y                                          # sslip.io, all defaults
sudo bash install.sh --no-tls --port 8080 -y                     # HTTP only
```

## What it sets up

A dedicated `Oblivions` user and `/opt/Oblivions` directory, the prebuilt server binary (~40 MB — everything embedded, no Go/Node/Docker needed), the LiveKit voice server, random secrets, hardened systemd services, the firewall, and (in TLS modes) Caddy. **Re-running is safe** — existing secrets are preserved.

When it finishes, the installer prints your URL (`https://yourdomain` or `https://1-2-3-4.sslip.io`). **The first user to register becomes the owner.**

## Running it

```bash
journalctl -u Oblivions-server -f          # follow logs
systemctl restart Oblivions-server         # restart

# update to a newer release:
curl -fsSL https://raw.githubusercontent.com/akinalpfdn/Mqvi/main/deploy/install.sh | sudo bash
systemctl restart Oblivions-server
```

Your data lives in **`/opt/Oblivions/data/`** (the database + uploaded files) — **back it up.** To use your server from the desktop app, point it there via **Connections** (next page).
