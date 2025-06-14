# WinterFlow Agent - in development

@TODO

## Agent Installation

Run on your server as root (use sudo):

```sh
curl -fsSL https://get.winterflow.io/agent | sudo bash
```

It will generate a 6-character code that you need to enter at https://app.winterflow.io.

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
* `/opt/winterflow/ansible` - This directory holds the main Ansible recipes.
* `/opt/winterflow/ansible/inventory` - This directory holds the main Ansible inventory.
* `/opt/winterflow/ansible/inventory/defaults.yml` - The default Ansible inventory file
* `/opt/winterflow/ansible/inventory/defaults.override.yml` - User-defined Ansible inventory file to override the
  `defaults.yml`
* `/opt/winterflow/ansible/playbooks` - This directory holds the main Ansible playbooks.
* `/opt/winterflow/ansible/roles` - This directory holds the main Ansible roles.
* `/opt/winterflow/ansible/apps_roles/` - This directory contains roles for applications along with their
  configurations.