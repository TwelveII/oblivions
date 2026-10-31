# Voice server only (LiveKit)

Use your existing account normally — only **voice and video** go through your own LiveKit server.

## Linux

SSH into your server and run:

```bash
curl -fsSL https://raw.githubusercontent.com/akinalpfdn/Mqvi/main/deploy/livekit-setup.sh | sudo bash
```

The script downloads LiveKit, opens the firewall ports, generates secure credentials, writes `livekit.yaml`, and starts it as a systemd service.

**Requirements:** any Linux server (Ubuntu 22.04+ / Debian 12+), 1 GB RAM, 1 CPU core.

## Windows

Open **PowerShell as Administrator** and run:

```powershell
irm https://raw.githubusercontent.com/akinalpfdn/Mqvi/main/deploy/livekit-setup.ps1 | iex
```

Same as above, plus it tries to forward your router ports via UPnP and sets LiveKit to auto-start on boot.

**Requirements:** Windows 10/11. If it's your own PC, it must stay on and online.

## Connect it to Oblivions

When the script finishes it prints **3 values** — a **URL**, **API key**, and **API secret**. In Oblivions, create a new server, choose **Self-Hosted**, and paste them in. That's it.

![the self-hosted server form with the 3 LiveKit values](assets/voice-livekit-1.png)

## Troubleshooting

| Problem | Fix |
| --- | --- |
| Voice won't connect | Ports are probably closed — check the firewall **and** your cloud provider's web firewall |
| Connected but no audio | UDP ports **50000–60000** may be blocked; allow UDP there |
| "Connection refused" | LiveKit isn't running — `systemctl status livekit` (Linux) |
| Works on LAN, not the internet | Set `use_external_ip: true` in `livekit.yaml`; forward ports 7880, 7881, 7882, and 50000–60000 |
