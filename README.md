# WinterFlow Agent

@TODO

## Agent Installation

Run as root on your server:

```sh
curl -fsSL https://winterflowio.github.io/agent/install.sh | sudo bash
```

## Directory Structure

The following directories and files are part of the WinterFlow Agent's directory structure:

* `/opt/winterflow` - The root directory of the WinterFlow Agent.
* `/opt/winterflow/agent` - This directory contains the agent binary.
* `/opt/winterflow/agent.config.json` - The configuration file for the agent.
* `/opt/winterflow/ansible` - This directory holds the main Ansible recipes.
* `/opt/winterflow/ansible/inventory` - This directory holds the main Ansible inventory.
* `/opt/winterflow/ansible/playbooks` - This directory holds the main Ansible playbooks.
* `/opt/winterflow/ansible/roles` - This directory holds the main Ansible roles.
* `/opt/winterflow/ansible/apps/` - This directory contains roles for applications along with their configurations.
* `/opt/winterflow/ansible/apps/configs` - This directory contains configuration files for applications.
* `/opt/winterflow/ansible/apps/inventory` - This directory stores application-specific variables and secrets.
* `/opt/winterflow/ansible/apps/roles` - This directory stores roles for applications.