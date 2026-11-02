# Oblivions

**A self-hostable communication platform for communities — text, voice and video.**

Oblivions is an open-source project built around the MQVI communication server. It brings together community servers, channels, direct messages, voice/video calls through a web browser, with a focus on user control and privacy.

> **Status: pre-release / beta.** The project is being prepared for its first public release. It has not received an independent security audit. Do not rely on it as the sole channel for sensitive or safety-critical communications.

## What it includes

- Community servers, text channels, roles and permissions
- Direct messages, friend requests, moderation and reporting
- Voice/video rooms backed by LiveKit
- Encrypted voice/video calls
- Optional end-to-end encryption for supported message conversations
- File uploads, user profiles, invitations and platform administration
- Browser-based web client

Oblivions is currently a web-only service. Use a supported web browser to access it. See [the setup guide](SETUP.md) before deploying.

## Try it

The hosted instance is currently offline while the project is prepared for public release. There is no public demo URL at this time.

## Install your own server

The supported path is a Linux server with a public IP address. A domain is recommended; the installer can also configure HTTPS using an IP-derived `sslip.io` hostname.

**Requirements**
- Ubuntu 22.04+ or Debian 12+ (other systemd-based Linux distributions may work)
- x86-64 or ARM64
- 2 vCPU and 4 GB RAM recommended
- Root access and outbound internet
- A domain pointing to the server, or a reachable public IP

**One-command installer** (available after the first tagged release):

```bash
curl -fsSL https://raw.githubusercontent.com/TwelveII/oblivions/main/deploy/install.sh -o oblivions-install.sh
less oblivions-install.sh
sudo bash oblivions-install.sh
```

The installer asks how to expose the service, generates unique secrets, configures the application and LiveKit, and installs systemd services. Review the script before running it as root. Full instructions, firewall requirements, updates, backups and troubleshooting are in **[SETUP.md](SETUP.md)**.

## Development

You will need Go (the version in `server/go.mod`), Node.js 22 and npm.

```bash
git clone https://github.com/TwelveII/oblivions.git
cd oblivions

cd client
npm ci
npm run dev
```

In another terminal:

```bash
cd server
go run .
```

For a production build, build the client first, copy `client/dist/` into `server/static/dist/`, then build the Go server from `server/`. See [SETUP.md](SETUP.md) for deployment and [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## Security and privacy

- The project has **not** had an independent security audit. Cryptographic design and implementation should be reviewed before high-risk use.
- End-to-end encryption is only available in the conversation types and modes that explicitly support it. Do not assume every message is end-to-end encrypted by default.
- A self-hosted operator controls that server's infrastructure, availability, backups and metadata. Self-hosting does not automatically protect devices, network endpoints or account recovery.
- Report suspected vulnerabilities privately. Please do not publish exploit details in public issues. See [SECURITY.md](SECURITY.md).

Read the Privacy and Terms pages on the website before using a deployment.

## Contributing

Issues and pull requests are welcome. Please include reproduction steps, expected behaviour and relevant logs with bug reports. Do not include passwords, tokens, private keys or private user data.

## License

See [LICENSE](LICENSE) for the project's license and obligations.
