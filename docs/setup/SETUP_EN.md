# OpenSourceBackup — Setup Guide

> *Creating backups is easy. Proving recoverability is the difference.*

🇩🇪 [Deutsche Version](SETUP_DE.md) | 🇬🇧 English

---

## Choose your installation type

| | Local | Proxmox |
|---|---|---|
| **Best for** | Testing, single machine | Home lab, always-on server |
| **Requirements** | Windows or Linux, Docker | Proxmox VE 7/8 |
| **Effort** | ~5 minutes | ~10 minutes |

---

# Option A — Local Installation

Run the Control Plane directly on your Windows or Linux machine for testing and development.

## Prerequisites

- **Docker Desktop** (Windows/Mac) or Docker (Linux)
- **Git** (optional)
- Port **8080** free

## Step 1 — Download and start

**Windows (PowerShell):**
```powershell
# Create folder
mkdir C:\opensourcebackup
cd C:\opensourcebackup

# Download docker-compose file
Invoke-WebRequest "https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/deployments/docker-compose/dev.yml" -OutFile docker-compose.yml

# Start PostgreSQL + Redis
docker compose -f docker-compose.yml up -d
```

**Linux:**
```bash
mkdir ~/opensourcebackup && cd ~/opensourcebackup
curl -fsSL https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/deployments/docker-compose/dev.yml -o docker-compose.yml
docker compose -f docker-compose.yml up -d
```

## Step 2 — Download and run the server

**Windows:**
```powershell
# Download server binary
Invoke-WebRequest "https://github.com/cerberus8484/opensourcebackup/releases/latest/download/opensourcebackup-server-windows-amd64.exe" -OutFile opensourcebackup-server.exe

# Set configuration
$env:DATABASE_URL = "postgres://opensourcebackup:dev_password@localhost:5432/opensourcebackup?sslmode=disable"
$env:LISTEN_ADDR  = ":8080"

# Start
.\opensourcebackup-server.exe
```

**Linux:**
```bash
# Download server binary
curl -fsSL https://github.com/cerberus8484/opensourcebackup/releases/latest/download/opensourcebackup-server-linux-amd64 \
  -o opensourcebackup-server && chmod +x opensourcebackup-server

# Start
DATABASE_URL="postgres://opensourcebackup:dev_password@localhost:5432/opensourcebackup?sslmode=disable" \
LISTEN_ADDR=":8080" \
./opensourcebackup-server
```

## Step 3 — Open the dashboard

```
http://localhost:8080/ui/
```

> ✅ You should see the **Backup Health** dashboard.

## Step 4 — Install the agent

Open the dashboard → **Agents → + Enroll Agent** → follow the wizard.

---

# Option B — Proxmox Installation

Install OpenSourceBackup on your Proxmox server so it runs 24/7 and can back up all your systems.

## Recommended: Debian 12 LXC Container

A lightweight container keeps the Control Plane isolated from the Proxmox host.

## Step 1 — Create LXC Container

In the **Proxmox shell** (or under Datacenter → your node → Shell):

```bash
# Update template list
pveam update

# See available Debian 12 templates
pveam available | grep "debian-12-standard"

# Download the template (use the name shown above)
pveam download local debian-12-standard_12.12-1_amd64.tar.zst

# Create container (adjust storage if needed)
TEMPLATE=$(pveam list local | grep "debian-12-standard" | awk '{print $1}' | tail -1)

pct create 200 $TEMPLATE \
  --hostname opensourcebackup \
  --memory 2048 \
  --cores 2 \
  --rootfs local-lvm:20 \
  --net0 name=eth0,bridge=vmbr0,ip=dhcp \
  --features nesting=1 \
  --unprivileged 1

# Start and enter the container
pct start 200
pct enter 200
```

> 💡 **Container ID 200** — change if already in use.
> **nesting=1** is required for Docker inside the container.

## Step 2 — Run the install script

Inside the container (you are now root inside the LXC):

```bash
curl -fsSL \
  https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-server.sh \
  | bash
```

**The script will automatically:**

| Step | What happens |
|---|---|
| 1 | Install Docker |
| 2 | Start PostgreSQL 16 + Redis 7 |
| 3 | Build the server binary (Go 1.22) |
| 4 | Build the Web UI |
| 5 | Run database migrations |
| 6 | Create systemd service |
| 7 | Show access URL and credentials |

**This takes about 5–10 minutes** (building from source).

## Step 3 — Note your access details

At the end you will see:

```
╔══════════════════════════════════════════════════════╗
║   ✓  OpenSourceBackup — Installation Complete        ║
╠══════════════════════════════════════════════════════╣
║   🌐  Web Dashboard: http://192.168.x.x:8080/ui/     ║
║   🔑  Username: (not required — Auth coming in v2)   ║
║   🔑  Password: (not required — Auth coming in v2)   ║
╚══════════════════════════════════════════════════════╝
```

Your credentials are also saved in:
```bash
cat /root/opensourcebackup-credentials.txt
```

## Step 4 — Open the dashboard

Open in your browser (replace with your Proxmox IP):

```
http://192.168.x.x:8080/ui/
```

## Alternative: Direct on Proxmox host

If you prefer to install directly on the Proxmox host (not recommended for production):

```bash
# On the Proxmox host as root
curl -fsSL \
  https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-server.sh \
  | bash
```

> ⚠️ This mixes OpenSourceBackup with the Proxmox system. Use LXC for production.

---

# Install the Agent

Once the dashboard is running, install agents on the systems you want to back up.

## Windows Agent

**In the dashboard:** Agents → + Enroll Agent → select system → Windows → copy command

```powershell
# Download agent
Invoke-WebRequest "http://<server-ip>:8080/downloads/agent/v0.1.0/windows-amd64" `
  -OutFile opensourcebackup-agent.exe

# Set variables (use values from the wizard)
$env:CONTROL_PLANE_URL  = "http://<server-ip>:8080"
$env:ENROLLMENT_TOKEN   = "<token from wizard>"
$env:RESTIC_PASSWORD    = "<your backup password>"
$env:RESTIC_REPO        = "Z:\BackupFolder"

# Start — enrolls automatically on first run
.\opensourcebackup-agent.exe
```

## Linux Agent

```bash
CONTROL_PLANE_URL=http://<server-ip>:8080 \
ENROLLMENT_TOKEN=<token from wizard> \
RESTIC_PASSWORD=<your backup password> \
RESTIC_REPO=/mnt/nas/backups \
bash <(curl -fsSL http://<server-ip>:8080/scripts/install-agent.sh)
```

---

# First Backup

1. **Repositories** → `+ New Repository` → select type (Local, NAS, S3…) → enter path
2. **Policies** → `+ New Policy` → select repository, include paths, schedule
3. **Jobs** → `+ New Job` → select system and policy → click ▶ Run Backup
4. Watch the live progress in the Job Details panel

---

# Next Steps

- **Restore Tests** → verify your backups can actually be restored
- **Scheduler** → set up automatic backup schedules
- **TLS/HTTPS** → enable for production: `make certs && make run-https`
- **Multiple Agents** → back up all your servers, VMs, workstations

---

*[github.com/cerberus8484/opensourcebackup](https://github.com/cerberus8484/opensourcebackup)*
