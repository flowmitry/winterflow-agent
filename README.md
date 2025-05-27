# WinterFlow Agent - in development

@TODO

## Agent Installation

Run as root on your server:

```sh
curl -fsSL https://get.winterflow.io/agent | sudo bash
```

## Directory Structure

The following directories and files are part of the WinterFlow Agent's directory structure:

* `/opt/winterflow` - The root directory of the WinterFlow Agent.
* `/opt/winterflow/agent` - This directory contains the agent binary.
* `/opt/winterflow/agent.config.json` - The configuration file for the agent.
* `/opt/winterflow/apps` - The directory holds the Docker Compose files to run your apps.
* `/opt/winterflow/ansible` - This directory holds the main Ansible recipes.
* `/opt/winterflow/ansible/inventory` - This directory holds the main Ansible inventory.
* `/opt/winterflow/ansible/inventory/defaults.yml` - The default Ansible inventory file
* `/opt/winterflow/ansible/inventory/custom.yml` - User-defined Ansible inventory file to override the `defaults.yml`
* `/opt/winterflow/ansible/playbooks` - This directory holds the main Ansible playbooks.
* `/opt/winterflow/ansible/roles` - This directory holds the main Ansible roles.
* `/opt/winterflow/ansible/apps_roles/` - This directory contains roles for applications along with their
  configurations.