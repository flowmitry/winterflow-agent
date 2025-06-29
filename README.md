# WinterFlow Agent

A lightweight agent for managing Docker applications on Unix systems.

## Requirements

### System Requirements
- **OS**: Any modern Unix system (Linux, macOS, BSD)
- **Resources**: Minimum 1 vCPU and 2GB RAM for Docker operations
- **Software Dependencies**:
  - [Docker](https://docs.docker.com/engine/install/)
  - [Docker Compose (plugin)](https://docs.docker.com/compose/install/linux/)
  - `jq` (JSON processor)
  - `curl` (HTTP client)

## Installation

### Automatic Installation

The recommended way to install the WinterFlow Agent is using the automatic installer:

```bash
curl -fsSL https://get.winterflow.io/agent | sudo bash
```

**Important**: The installation process generates a unique 6-character registration code. You'll need this code to register your server at [https://app.winterflow.io](https://app.winterflow.io).

### Manual Installation

If you prefer to install manually or the automatic installation fails, follow these steps:

1. **Verify system requirements** and install dependencies
2. **Create the winterflow user** and add it to the `docker` group
3. **Create the installation directory**: `/opt/winterflow`
4. **Download the agent binary** for your architecture from [GitHub Releases](https://github.com/flowmitry/winterflow-agent/releases) to `/opt/winterflow/agent`
5. **Make the binary executable**: `chmod +x /opt/winterflow/agent`
6. **Register your server**: `./agent --register`
7. **Create and configure** the systemd service
8. **Start the service** and complete automatic registration

For detailed installation steps and troubleshooting, refer to the [install.sh](./install.sh) file.

## Service Management

The WinterFlow Agent runs as a systemd service. Use the following commands to manage it:

### Control the Service

```bash
sudo systemctl start|stop|restart|status winterflow-agent
```

### View Service Logs

```bash
# Follow logs in real-time
sudo journalctl -u winterflow-agent -f
```

## Application Restoration

If you re-install the agent, migrate the `/opt/winterflow` directory to a new machine, or re-register your agent, you can safely restore all application templates (not app's data).

### Prerequisites
- The agent must already be **registered** (`agent_status` = `registered` in `agent.config.json`)
- The directory `/opt/winterflow/apps_templates` contains your previous application templates

### Restore Command

```bash
./agent --restore
```

### Restoration Process

1. **Backup creation** – A full copy of `apps_templates` is made to `apps_templates.bak`
   - If the backup directory already exists, the command aborts to avoid overwriting previous data
2. **UUID regeneration** – Every application ID is replaced with a fresh UUID to avoid collisions
3. **Version pruning** – Only the newest version directory of each app is kept and renamed to `1` for consistency
4. **Cloud notification** – Sent to WinterFlow server to recreate *all* applications

If the server responds with `200 OK`, the restore has succeeded. All applications will appear in the dashboard moments later.

## Uninstallation

To completely remove the WinterFlow Agent from your system, run the following commands as root (use `sudo`):

### Stop and Disable the Service

```bash
sudo systemctl stop winterflow-agent
sudo systemctl disable winterflow-agent
```

### Remove Systemd Service File

```bash
sudo rm -f /etc/systemd/system/winterflow-agent.service
sudo systemctl daemon-reload
```

### Remove Installation Directories

```bash
sudo rm -rf /opt/winterflow
sudo rm -rf /var/log/winterflow
```

### Remove User (Optional)

```bash
sudo userdel -r winterflow
```

### Remove Sudoers Configuration (if added)

```bash
sudo rm -f /etc/sudoers.d/winterflow
```

## Directory Structure

The WinterFlow Agent uses the following directory structure:

| Directory/File | Description |
|----------------|-------------|
| `/opt/winterflow` | Root installation directory |
| `/opt/winterflow/agent` | Agent binary executable |
| `/opt/winterflow/agent.config.json` | Agent configuration file |
| `/opt/winterflow/.certs` | Private/public key certificates |
| `/opt/winterflow/apps_templates` | Application version templates |
| `/opt/winterflow/apps` | Docker Compose files for running applications |

## Support

For support and documentation, visit:
- **Web Application**: [https://app.winterflow.io](https://app.winterflow.io)
- **Documentation**: [https://winterflow.io](https://winterflow.io)
- **GitHub Repository**: [https://github.com/flowmitry/winterflow-agent](https://github.com/flowmitry/winterflow-agent)
