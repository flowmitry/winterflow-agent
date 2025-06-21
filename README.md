# WinterFlow Agent - in development

@TODO

## Requirements

- Recommended OS: Ubuntu 22+ and Debian 12+.
- System Resources: at least 1 vCPU and 2GB RAM for Docker.

Note: Only the Debian OS family is supported.

## Agent Installation

Run on your server as root (use sudo):

```sh
curl -fsSL https://get.winterflow.io/agent | sudo bash
```

The installation process includes the setup of necessary dependencies(`curl jq`) and the generation of a unique
6-character code. This code is required for your server's registration
at [https://app.winterflow.io](https://app.winterflow.io).

## Manual Agent installation

You can manually download and execute the [./install.sh](./install.sh) script with `sudo`.

Use `sudo source install.sh --force` if you use Debian-based distributive outside of recommended.

### Manual Agent Registration

If automatic registration failed during installation, you can manually register:

```sh
sudo -u winterflow /opt/winterflow/agent --register
```

## Service Management

The WinterFlow Agent runs as a systemd service. Here are the commands to manage it:

### Manager Winterflow agent service
```sh
sudo systemctl start winterflow-agent
sudo systemctl stop winterflow-agent
sudo systemctl restart winterflow-agent
sudo systemctl status winterflow-agent
```

### View service logs
```sh
# Follow logs in real-time
sudo journalctl -u winterflow-agent -f

# View logs from last hour
sudo journalctl -u winterflow-agent --since "1 hour ago"
```

### Log Files Location
- Standard output: `/var/log/winterflow/winterflow_agent.log`
- Error output: `/var/log/winterflow/winterflow_agent_error.log`


## Agent Uninstallation

To completely remove the WinterFlow Agent from your system, run the following commands as root (use `sudo`):

### 1. Stop and disable the service
```sh
sudo systemctl stop winterflow-agent
sudo systemctl disable winterflow-agent
```

### 2. Remove the systemd service file
```sh
sudo rm -f /etc/systemd/system/winterflow-agent.service
sudo systemctl daemon-reload
```

### 3. Remove installation directories
```sh
sudo rm -rf /opt/winterflow
sudo rm -rf /var/log/winterflow
```

### 4. Remove the winterflow user (optional)
```sh
sudo userdel -r winterflow
```

### 5. Remove sudoers configuration
```sh
sudo rm -f /etc/sudoers.d/winterflow
```

**Note:** The uninstallation will not remove the packages installed during setup (`curl` and `ansible`) as they may be used by other applications on your system.

## Directory Structure

The following directories and files are part of the WinterFlow Agent's directory structure:

* `/opt/winterflow` - The root directory of the WinterFlow Agent.
* `/opt/winterflow/agent` - This directory contains the agent binary.
* `/opt/winterflow/agent.config.json` - The configuration file for the agent.
* `/opt/winterflow/apps` - The directory holds the Docker Compose files to run your apps.
* `/opt/winterflow/apps_templates` - This directory holds your Docker Compose apps templates