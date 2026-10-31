# Self-hosting

Oblivions supports two self-hosting approaches, depending on how much of the platform you want to operate yourself.

## Two paths

**1. Voice server only** *(easiest)*

Run your own LiveKit server for voice and video. Your Oblivions client connects to that server for calls, while the rest of the platform remains on the backend you normally use.

**2. Full server** *(fully independent)*

Run the entire Oblivions platform yourself — accounts, messages, files, and voice. You operate the infrastructure and control your deployment.

| | Voice only | Full server |
| --- | --- | --- |
| What you host | LiveKit voice server | Entire Oblivions platform |
| Setup | LiveKit setup script | Full install script |
| Specs | 1 GB RAM, 1 core | 4 GB RAM, 2 vCPU |

## What you'll need

- A **Linux server** (Ubuntu 22.04+ / Debian 12+ recommended). Hetzner, DigitalOcean, or Contabo are examples of VPS providers.
- For a full server, a **domain is optional** if the installer supports its automatic `sslip.io` hostname option.
- HTTPS is important because browsers restrict microphone, camera, and screen sharing on insecure HTTP pages.

The next two pages explain each approach.
