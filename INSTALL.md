# Installation Guide

> OpenSourceBackup — Backup Control Plane with Restore Assurance
>
> *Creating backups is easy. Proving recoverability is the difference.*

---

## Table of Contents

1. [Requirements](#requirements)
2. [Server Installation (Proxmox / Linux)](#server-installation)
3. [Access the Dashboard](#access-the-dashboard)
4. [Install Agent on Windows](#install-agent-on-windows)
5. [Install Agent on Linux](#install-agent-on-linux)
6. [First Backup](#first-backup)
7. [Restore Test](#restore-test)
8. [Management Commands](#management-commands)
9. [Troubleshooting](#troubleshooting)

---

## Requirements

| Component | Requirement |
|---|---|
| **Server** | Debian 12 / Ubuntu 22.04 (or Proxmox VE 8 host/LXC) |
| **RAM** | 512 MB minimum, 2 GB recommended |
| **Disk** | 5 GB for the server, separate storage for backups |
| **Network** | Port 8080 (or custom) accessible from agents |
| **Agents** | Windows 10/11/Server, Linux (any distro) |

---

## Server Installation

### Option A — Proxmox LXC Container (Recommended)

Create a Debian 12 LXC container on your Proxmox host:

```bash
# On your Proxmox host — find the correct template
pveam update
pveam available | grep "debian-12-standard"

# Download the template (use the name shown above)
pveam download local debian-12-standard_12.12-1_amd64.tar.zst

# Create LXC container
TEMPLATE=$(pveam list local | grep "debian-12-standard" | awk '{print $1}' | tail -1)
pct create 200 $TEMPLATE \
  --hostname opensourcebackup \
  --memory 2048 \
  --cores 2 \
  --rootfs local-lvm:20 \
  --net0 name=eth0,bridge=vmbr0,ip=dhcp \
  --features nesting=1 \
  --unprivileged 1

pct start 200
pct enter 200
```

### Option B — Direct on Debian/Ubuntu

```bash
# Log in as root on your Debian/Ubuntu server
ssh root@your-server
```

### Run the Install Script

Inside the container (or on your Debian/Ubuntu server), run as **root**:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-server.sh \
  | bash
```

**What the script does automatically:**
- Installs Docker (PostgreSQL 16 + Redis 7)
- Downloads or builds the server binary
- Builds the Web UI
- Runs database migrations
- Creates a systemd service
- Saves credentials to `/root/opensourcebackup-credentials.txt`

**The installation takes 5–10 minutes** (building from source).

### Result

At the end of the installation you will see:

```
╔══════════════════════════════════════════════════════════════════╗
║       ✓  OpenSourceBackup successfully installed!               ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                  ║
║   🌐  WEB DASHBOARD                                              ║
║       http://192.168.1.100:8080/ui/                              ║
║                                                                  ║
║   🔑  LOGIN                                                       ║
║       Username : (not required — Auth coming in v2)              ║
║       Password : (not required — Auth coming in v2)              ║
╚══════════════════════════════════════════════════════════════════╝
```

> ⚠️ **Security note:** The API currently has no authentication.
> Only use it inside a trusted internal network (e.g. Proxmox LAN).
> TLS + Login (RBAC) are planned for the next release.

---

## Access the Dashboard

Open in your browser:

```
http://<server-ip>:8080/ui/
```

You will see the **Backup Health** dashboard:

- **Protected Systems** — how many systems are registered
- **Restore Tested %** — how many snapshots have been verified
- **Recent Jobs** — backup history

---

## Install Agent on Windows

The agent runs on the system you want to back up and sends data directly to the backup destination (NAS, S3, local path).

### Step 1 — Register the system

In the dashboard: **Systems → + New System**
- Enter the hostname (e.g. `my-pc`)
- Select Risk Class: `standard` or `critical`

### Step 2 — Get an enrollment token

In the dashboard: **Agents → + Enroll Agent**
- Select your system
- Select platform: **Windows**
- Follow the wizard to Step 4
- Copy the generated install command

### Step 3 — Run the install command on Windows

Open **PowerShell as Administrator** and run the generated command:

```powershell
# Download the agent
Invoke-WebRequest "http://<server-ip>:8080/downloads/agent/v0.1.0/windows-amd64" `
  -OutFile opensourcebackup-agent.exe

# Set environment variables (use values from the wizard)
$env:CONTROL_PLANE_URL  = "http://<server-ip>:8080"
$env:ENROLLMENT_TOKEN   = "<token-from-wizard>"
$env:RESTIC_PASSWORD    = "<your-backup-password>"
$env:RESTIC_REPO        = "Z:\BackupFolder"        # Your NAS path
.\opensourcebackup-agent.exe
```

The agent will:
1. Enroll itself automatically (saves token to `data\agent-token`)
2. Start polling for backup jobs every 30 seconds

### Step 4 — Subsequent starts (no re-enrollment needed)

```powershell
$env:CONTROL_PLANE_URL  = "http://<server-ip>:8080"
$env:RESTIC_PASSWORD    = "<your-backup-password>"
$env:RESTIC_REPO        = "Z:\BackupFolder"
.\opensourcebackup-agent.exe
```

---

## Install Agent on Linux

### Step 1 — Register system + get enrollment token

Same as Windows steps 1 and 2 above.
Select platform: **Linux x64** or **Linux ARM64**.

### Step 2 — Run install command

```bash
# Get enrollment token from the wizard and set variables
CONTROL_PLANE_URL=http://<server-ip>:8080 \
ENROLLMENT_TOKEN=<token-from-wizard> \
RESTIC_PASSWORD=<your-backup-password> \
RESTIC_REPO=/mnt/nas/backups \
bash <(curl -fsSL http://<server-ip>:8080/scripts/install-agent.sh)
```

This installs:
- Agent binary to `/usr/local/bin/opensourcebackup-agent`
- Restic (if not already installed)
- systemd service `opensourcebackup-agent`
- Config file at `/etc/opensourcebackup/agent.env`

---

## First Backup

### Step 1 — Create a Repository

In the dashboard: **Repositories → + New Repository**

| Field | Example |
|---|---|
| Type | `NAS / SMB` for Windows NAS share |
| Location | `Z:\Backups` (Windows) or `/mnt/nas/backups` (Linux) |
| Encryption | `aes256` (recommended) |
| WORM Lock | Enable if your NAS supports Object Lock |

### Step 2 — Create a Policy

In the dashboard: **Policies → + New Policy**

| Field | Example |
|---|---|
| Name | `nightly-documents` |
| Engine | `Restic — Files & folders — Windows, Linux, NAS, S3` |
| Repository | Select the repository from Step 1 |
| Include Paths | `C:\Users\Admin\Documents` |
| Schedule | `Daily at 02:00` |
| Retention | Daily: 7, Weekly: 4, Monthly: 12 |

### Step 3 — Run a Backup

Option A — **Manual**: **Jobs → + New Job** → select system and policy

Option B — **Automatic**: The scheduler runs policies on their cron schedule.
Restart the server after creating a new policy for the schedule to activate:
```bash
systemctl restart opensourcebackup
```

### Step 4 — Watch Progress

**Jobs** page → click the **▶** button on any job to open the live detail panel.

The panel shows:
- Real-time status (Pending → Running → Success)
- Duration, file count, backup size
- Error details if something fails

---

## Restore Test

A restore test proves that a snapshot can actually be recovered.

### Create a Restore Test

1. Go to **Restore Tests → + New Restore Test**
2. Select a snapshot
3. Leave **Target Path** empty (uses a safe sandbox directory automatically)
4. Click **Create Restore Test**

The agent will:
1. Run `restic restore <snapshot-id> --target <sandbox>`
2. Count the restored files and bytes independently
3. Report back to the dashboard

### View Results

The **Dashboard** will show:
```
Restore Tested: X%   ← percentage of snapshots verified
```

**Restore data location:**
```
Windows: data\restore-tests\<test-id>\C\Users\...
Linux:   data/restore-tests/<test-id>/home/...
```

---

## Management Commands

### Server (Linux/Proxmox)

```bash
# View logs
journalctl -u opensourcebackup -f

# Restart
systemctl restart opensourcebackup

# Stop
systemctl stop opensourcebackup

# Database containers
docker compose -f /opt/opensourcebackup/docker-compose.yml ps
docker compose -f /opt/opensourcebackup/docker-compose.yml logs postgres
```

### Agent (Windows)

```powershell
# Stop agent
Stop-Process -Name "opensourcebackup-agent" -Force

# Check if running
Get-Process -Name "opensourcebackup-agent" -ErrorAction SilentlyContinue

# Remove agent (delete token to force re-enrollment)
Remove-Item "data\agent-token" -Force
```

### Agent (Linux — systemd)

```bash
journalctl -u opensourcebackup-agent -f   # Logs
systemctl restart opensourcebackup-agent  # Restart
systemctl stop opensourcebackup-agent     # Stop
```

---

## Troubleshooting

### Dashboard shows blank white page
The Web UI was built with the wrong API URL. Rebuild:
```bash
cd /tmp/osb-web/web
VITE_API_URL=http://<server-ip>:<port> npx vite build --base /ui/
cp -r dist/. /opt/opensourcebackup/web-ui/
systemctl restart opensourcebackup
```

### Agent: "token revoked or invalid"
The agent token is no longer valid. Delete and re-enroll:
```powershell
Remove-Item "data\agent-token" -Force
$env:ENROLLMENT_TOKEN = "<new-token-from-dashboard>"
.\opensourcebackup-agent.exe
```

### Port 8080 already in use
Change the port:
```bash
sed -i 's/LISTEN_ADDR=:8080/LISTEN_ADDR=:8090/' /etc/opensourcebackup/server.env
systemctl restart opensourcebackup
```
Then rebuild the Web UI with the new port.

### PostgreSQL permission denied
```bash
docker compose -f /opt/opensourcebackup/docker-compose.yml down
chown -R 999:999 /var/lib/opensourcebackup/postgres
docker compose -f /opt/opensourcebackup/docker-compose.yml up -d
```

### Restore test fails on Windows (UtimesNano / Access denied)
This is a Windows system directory permission issue — the actual files were restored successfully.
Make sure you are using agent version `v0.1.0` or later which handles this automatically.

---

## Repository Types

| Type | Use Case |
|---|---|
| **Local Path** | Local disk, USB drive, mounted volume |
| **NAS / NFS** | Synology, QNAP, TrueNAS via NFS |
| **NAS / SMB** | Windows share, Synology via SMB/CIFS |
| **MinIO / S3** | Self-hosted MinIO, AWS S3, GCS, Azure |
| **Borg** | Linux servers via SSH |
| **pgBackRest** | PostgreSQL databases |
| **Velero** | Kubernetes clusters |

## Backup Engines

| Engine | Best for |
|---|---|
| **Restic** | Files & folders — Windows, Linux, NAS, S3 |
| **Borg** | Linux servers via SSH — very efficient deduplication |
| **pgBackRest** | PostgreSQL databases — WAL archiving & Point-in-Time Recovery |
| **Velero** | Kubernetes clusters — Deployments, ConfigMaps, Volumes |

---

## Next Steps

After your first successful backup and restore test:

- Set up **automatic schedules** for all your systems
- Configure **retention policies** (daily/weekly/monthly)
- Enable **WORM/Object Lock** on your repository for ransomware protection
- Set up **TLS/HTTPS** for production use (`make certs && make run-https`)
- Monitor the **Dead-Man's Switch** alerts for overdue backups

---

*OpenSourceBackup — [github.com/cerberus8484/opensourcebackup](https://github.com/cerberus8484/opensourcebackup)*
