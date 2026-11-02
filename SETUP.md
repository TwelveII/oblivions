# Oblivions — installation and first-run guide

This guide is for people who want to run their own Oblivions instance. You do not need to know Go, JavaScript or Docker to install the released server.

## 1. Before you begin

You need:

- A clean Ubuntu 22.04+ or Debian 12+ server (x86-64 or ARM64)
- 2 CPU cores and 4 GB RAM recommended
- Root access (`sudo`)
- A public IPv4 address and inbound firewall access
- A domain name pointing to the server (recommended). Without one, the installer can use `sslip.io`.

For a domain deployment, create an `A` record pointing to your server's IPv4 address. If using Cloudflare, initially set the record to DNS-only while obtaining the first certificate. Do not put the LiveKit UDP media ports behind an HTTP-only proxy.

## Domestic-outage deployment (important for Iran)

The default application runtime is designed to keep core chat features local, but a domestic-outage deployment requires **domestic infrastructure**, not just a setting:

- Host the web app, Go API, SQLite database, uploads, LiveKit SFU and coturn TURN relay on servers reachable inside Iran. A foreign VPS cannot be reached when international routes are cut.
- Use a domain whose authoritative DNS and recursive resolution remain reachable domestically. Do not rely on Cloudflare proxy, foreign DNS resolvers, `sslip.io`, or a foreign CDN for the app hostname.
- Install a browser-trusted TLS certificate before an outage and plan renewal without relying on a foreign ACME service. Keep the certificate and private key on the server.
- Configure `LIVEKIT_URL`, `TURN_URLS` and `STUN_URLS` to your own domestic endpoints. Do not use public Google STUN or foreign TURN relays. TURN is important for calls across carrier NATs; direct P2P without a relay is not reliable.
- Keep Firebase/FCM, APNs, Resend, Klipy, Have I Been Pwned, GeoIP downloads and link previews disabled if you require zero foreign runtime calls. The installer sets the server-side options off by default. Mobile builds also leave push registration off unless `VITE_PUSH_NOTIFICATIONS_ENABLED=true` is explicitly set at build time. Push notifications may not arrive during an outage; users can still receive messages while the app is connected over its WebSocket.
- Electron auto-updates are disabled by default because they query GitHub Releases. Distribute desktop updates through a domestic mirror or offline installer; `OBLIVIONS_ENABLE_AUTO_UPDATE=true` is an explicit opt-in for networks that can reach GitHub.
- Preinstall OS packages, server binaries, LiveKit, antivirus signatures, mobile/desktop app builds and all assets before the outage. GitHub Releases, package registries and the installer are not available if international access is cut.

The application cannot guarantee domestic availability if the hosting provider, DNS, TLS validation, power, or domestic routing is unavailable. The installer itself downloads from GitHub and public package repositories, so perform installation and updates before an outage or mirror every required artifact inside Iran. Test from multiple Iranian mobile and fixed-line networks with international access blocked before calling the deployment outage-ready.

## 2. Install

Download and inspect the installer first:

```bash
curl -fL https://raw.githubusercontent.com/TwelveII/oblivions/main/deploy/install.sh -o oblivions-install.sh
less oblivions-install.sh
sudo bash oblivions-install.sh
```

The interactive installer guides you through the public URL mode:

1. **Your own domain (recommended):** Caddy obtains and renews a TLS certificate automatically.
2. **No domain:** an IP-derived `sslip.io` hostname is used for HTTPS.
3. **HTTP only:** for local testing only. Browsers restrict microphone, camera and screen sharing on insecure origins.

For a non-interactive install with your own domain:

```bash
sudo bash oblivions-install.sh --domain chat.example.com -y
```

The installer downloads a signed-off project release asset (binary), LiveKit, and required system packages; generates unique secrets; configures systemd; and starts the services. Never share `/opt/mqvi/.env` or `/opt/mqvi/livekit.yaml`.

> Only run the installer after checking the release and reviewing the script. It runs as root because it installs system services and firewall rules.

## 3. First login and administration

1. Open the HTTPS URL printed by the installer.
2. Register the first account. On a fresh instance, the first registered account becomes the server owner/platform administrator.
3. Sign in and complete the profile/setup prompts.
4. Open the administration area and review registration/invitation settings, reporting, server settings and security options.
5. Create a test account in a private browser or on another device. Verify registration, login, logout, password recovery, file upload, messaging and voice/video before inviting others.

Do not expose an instance to the public until you have confirmed the first-admin flow and configured your policies.

## 4. Network and firewall

For TLS deployments, expose TCP 80 and 443 for Caddy. LiveKit also needs direct connectivity:

| Port | Protocol | Purpose |
|---|---|---|
| 80, 443 | TCP | HTTP challenge and HTTPS |
| 7880 | TCP | LiveKit signalling |
| 7881 | TCP | LiveKit TURN/TCP relay |
| 7882 | UDP | LiveKit media |
| 50000–50200 | UDP | LiveKit ICE media range |

The application port (default 9090) should remain bound to localhost behind Caddy. If your hosting provider has a separate network firewall, open the required ports there too. Do not expose the database or internal readiness port.

## 5. Service management

The default installation directory is `/opt/mqvi`.

```bash
# Check services
sudo systemctl status mqvi-server mqvi-livekit caddy

# Follow application logs
sudo journalctl -u mqvi-server -f

# Follow voice service logs
sudo journalctl -u mqvi-livekit -f

# Restart
sudo systemctl restart mqvi-livekit mqvi-server

# Stop the application and voice service
sudo systemctl stop mqvi-server mqvi-livekit

# Start them again
sudo systemctl start mqvi-livekit mqvi-server
```

If Caddy is managed by this installer, stop it as well to make the HTTPS website unavailable:

```bash
sudo systemctl stop caddy
sudo systemctl start caddy
```

## 6. Updates

Use the installer again only when a compatible release is published. It preserves existing configuration and data. Read the release notes first, then:

```bash
curl -fL https://raw.githubusercontent.com/TwelveII/oblivions/main/deploy/install.sh -o oblivions-install.sh
less oblivions-install.sh
sudo bash oblivions-install.sh
```

Keep a backup before updating. Do not replace the application binary with an arbitrary build from an untrusted source.

## 7. Backups and recovery

Back up the entire `/opt/mqvi/data/` directory and the secret configuration files in a secure, encrypted location. The SQLite database may use WAL mode: take a consistent SQLite backup or stop the application before copying the database files.

At minimum preserve:

- `/opt/mqvi/data/` — database and uploaded files
- `/opt/mqvi/.env` — signing and encryption secrets
- `/opt/mqvi/livekit.yaml` — LiveKit API credentials and configuration

Restrict backup permissions. Losing the encryption key can make encrypted configuration data unrecoverable. Test restoration on a separate machine before trusting the backup.

## 8. Troubleshooting

**The site does not load**
- Check `systemctl status mqvi-server caddy`.
- Read `journalctl -u mqvi-server -n 100 --no-pager`.
- Confirm DNS points to this server and TCP 80/443 are reachable.

**Login works but voice/video does not**
- Confirm `mqvi-livekit` is active.
- Check UDP 7882 and 50000–50200 at both the OS and provider firewalls.
- Confirm the public URL uses HTTPS and LiveKit is reachable from clients.

**The service fails after an update**
- Check the service logs and available disk space.
- Restore the previous binary and matching configuration from your backup.
- Do not delete `/opt/mqvi/data` when troubleshooting.

## 9. Production checklist

Before inviting the public:

- [ ] Use HTTPS and verify certificate renewal.
- [ ] Confirm first-account administrator assignment is secure.
- [ ] Set registration and invitation policy intentionally.
- [ ] Test account recovery, session revocation and account deletion.
- [ ] Test moderation, reporting and abuse-handling workflows.
- [ ] Confirm backups and perform a restore test.
- [ ] Review firewall rules and ensure internal ports are not public.
- [ ] Configure operational monitoring, disk alerts and log retention.
- [ ] Publish accurate privacy and terms documents for your deployment.
- [ ] Review the security limitations: the project has not received an independent security audit.

Self-hosting gives the operator control of the infrastructure; it does not make the application or endpoint devices invulnerable.


## 10. Deploy a prebuilt binary during an international outage

GitHub Actions and public package registries may be unreachable during an international route shutdown. For an already-provisioned domestic server, prepare a Linux server binary before the outage and transfer it directly over the domestic network.

Build the artifact while dependencies are reachable (or from your own npm/Go mirrors):

```bash
cd client
npm ci
npm run build
cd ..

rm -rf server/static/dist
mkdir -p server/static/dist
mkdir -p deploy/package
cp -a client/dist/. server/static/dist/

cd server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o ../deploy/package/mqvi-server .
cd ..
```

For an ARM64 server, use `GOARCH=arm64` instead. The resulting binary includes the frontend assets. Keep the binary and its checksum on storage reachable from your domestic deployment machine before the outage.

The deployment helper does not contact GitHub, package registries, or any external API. It only uses SSH to the host you specify. The host must already have the Oblivions systemd installation, its existing `/opt/mqvi/.env`, database, uploads, and service unit. Verify the SSH host key in `known_hosts` beforehand; strict host-key checking is enabled. This helper is manual and independent of any hosted instance.

```bash
chmod +x deploy/domestic-deploy.sh
deploy/domestic-deploy.sh \
  --host YOUR_DOMESTIC_SERVER_IP \
  --user root \
  --artifact deploy/package/mqvi-server
```

The helper backs up the previous server binary and SQLite database (including WAL/SHM sidecars), installs the new one, restarts `mqvi-server.service`, and checks the loopback readiness endpoint at `127.0.0.1:9091/health/ready`. If restart or readiness fails, it restores the previous binary and database. It does not change `.env`, the database, uploads, LiveKit, Caddy, DNS, or firewall settings.

This is a deployment path for an already-provisioned host, not an offline installer. OS packages, Caddy, LiveKit, TURN, certificates, DNS, antivirus signatures, and application dependencies must be installed or mirrored domestically in advance. The helper is not connected to an automatic deployment workflow.


## 11. Build and install a complete offline application bundle

For a new, already-provisioned domestic Linux host, the repository now includes a bundle builder and a no-download installer. This avoids GitHub Actions and public registries during the actual installation. It does **not** replace provisioning the operating system, TLS, DNS, firewall, or TURN relay.

Before the outage, prepare a local LiveKit binary matching the target architecture (the regular installer currently uses LiveKit v1.9.12), and warm the npm and Go dependency caches while internet is available. From the repository root, run `cd client && npm ci && cd ../server && go mod download`; this caches the exact SheetJS tarball dependency and Go modules. The builder deliberately runs npm and Go in offline mode and fails rather than silently reaching the internet:

```bash
LIVEKIT_BINARY=/path/to/livekit-server bash deploy/prepare-offline-bundle.sh
```

The resulting archive is written under `dist/offline/`. Transfer the archive and `deploy/offline-install.sh` to the domestic server using a domestic network or physical media. The installer verifies the archive's SHA-256 manifest and CPU architecture, creates the `mqvi` service account, installs the app and local LiveKit, generates secrets on the host, disables optional external lookups, and checks database-backed readiness:

```bash
sudo bash deploy/offline-install.sh \
  --bundle oblivions-offline-linux-amd64-VERSION.tar.gz \
  --public-ip YOUR_DOMESTIC_PUBLIC_IPV4 \
  --domain chat.example.ir \
  --livekit-url wss://livekit.example.ir
```

Use the actual public IPv4 address that clients can reach. LiveKit is configured with `use_external_ip: false` and an explicit `node_ip`, so it does not need STUN-based public-IP discovery. Its media ports must be routed to that address. Open TCP 80/443 for the TLS proxy, TCP 7881 for ICE/TCP, and UDP 50000-50200; keep TCP 7880 private behind the TLS proxy. Configure an already-installed Caddy or other TLS reverse proxy to forward `chat.example.ir` to `127.0.0.1:9090` and `livekit.example.ir` to `127.0.0.1:7880`; install browser-trusted certificates for both names before the outage. The installer pins the LiveKit hostname to loopback in the server's `/etc/hosts` so backend SDK calls use the local TLS proxy, while domestic clients resolve it to the domestic public IP. The script intentionally does not install packages, modify firewall rules, request certificates, or configure DNS. A basic Caddy reverse-proxy configuration is:

```caddyfile
chat.example.ir {
    tls /etc/ssl/oblivions/chat-fullchain.pem /etc/ssl/oblivions/chat-key.pem
    reverse_proxy 127.0.0.1:9090
}

livekit.example.ir {
    tls /etc/ssl/oblivions/livekit-fullchain.pem /etc/ssl/oblivions/livekit-key.pem
    reverse_proxy 127.0.0.1:7880
}
```

Use certificates already installed on the host (or provision them before the outage). Keep the LiveKit signalling port 7880 blocked from direct public access; Caddy exposes it over WSS.

The offline installer starts the application and LiveKit only. It does not install coturn; restrictive carrier NATs may therefore prevent some calls until a domestic TURN relay is provisioned and `TURN_SECRET`/`TURN_URLS` are configured. It also leaves antivirus disabled unless you separately provision and configure ClamAV. Review the generated `/opt/mqvi/.env` before opening registration.


### Prebuilt offline bundles in GitHub Releases

When a version tag (`v*`) is published, the release workflow also creates ready-to-transfer offline runtime archives for Linux AMD64 and ARM64. Each archive contains the compiled Oblivions server, the matching pinned LiveKit server binary, a README with the target architecture, and an internal SHA-256 manifest; a separate `.sha256` file verifies the whole archive. Download the correct architecture archive while connectivity is available, transfer it to the domestic host, and pass it to `deploy/offline-install.sh`. This is the simplest way to prepare the application payload before an international outage. It still does not provision OS packages, TLS, DNS, firewall rules, or coturn.
