# Winterflow Agent

@TODO

## Agent Installation

Run as root on your server:

```sh
curl -fsSL https://winterflowio.github.io/agent/install.sh | sudo bash
```

## Directory Structure

The following directories and files are part of the Winterflow Agent's directory structure:

* `/opt/winterflow` - The root directory of the Winterflow Agent.
* `/opt/winterflow/agent` - This directory contains the agent binary.
* `/opt/winterflow/agent.config.json` - The configuration file for the agent.
* `/opt/winterflow/ansible` - This directory holds the main Ansible recipes.
* `/opt/winterflow/ansible_apps/` - This directory contains roles for applications along with their configurations.
* `/opt/winterflow/ansible_apps/apps.json` - A JSON file listing all installed applications.
* `/opt/winterflow/ansible_apps/roles` - This directory stores roles for applications.
* `/opt/winterflow/ansible_apps/inventory` - This directory stores application-specific variables and secrets.
* `/opt/winterflow/ansible_apps/configs` - This directory contains configuration files for applications.