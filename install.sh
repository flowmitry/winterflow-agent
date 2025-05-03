#!/bin/bash

# Winterflow Agent Installer
# -------------------------
# This script installs the Winterflow Agent on Ubuntu 20.04+ or Debian 12+
#
# Quick Install:
#   curl -fsSL https://winterflowio.github.io/agent/install.sh | sudo bash
#
# Manual Install:
#   curl -fsSL https://winterflowio.github.io/agent/install.sh > winterflow-install.sh
#   chmod +x winterflow-install.sh
#   sudo ./winterflow-install.sh
#
# Source Code: https://github.com/winterflowio/agent

# Exit on any error
set -e

#######################
# Configuration Variables
#######################

# Installation paths
INSTALL_DIR="/opt/winterflow"
AGENT_BINARY="${INSTALL_DIR}/agent"
CONFIG_FILE="${INSTALL_DIR}/agent.config.json"
SERVICE_FILE="/etc/systemd/system/winterflow-agent.service"
INSTALL_SCRIPT="${INSTALL_DIR}/install.sh"

# URLs
GITHUB_API="https://api.github.com/repos/winterflowio/agent/releases/latest"
INSTALL_SCRIPT_URL="https://winterflowio.github.io/agent/install.sh"

# Required packages
REQUIRED_PACKAGES="curl"

# Minimum required versions
MIN_UBUNTU_VERSION=20
MIN_DEBIAN_VERSION=12

#######################
# Utility Functions
#######################

# Function to log messages
log() {
    local level="$1"
    local message="$2"
    local RED='\033[0;31m'
    local GREEN='\033[0;32m'
    local YELLOW='\033[0;33m'
    local NC='\033[0m' # No Color
    
    case "$level" in
        "info")
            echo -e "[${GREEN}INFO${NC}] $message"
            ;;
        "warn")
            echo -e "[${YELLOW}WARN${NC}] $message"
            ;;
        "error")
            echo -e "[${RED}ERROR${NC}] $message"
            ;;
        *)
            echo "$message"
            ;;
    esac
}

# Function to check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log "error" "Please run as root (use sudo)"
        exit 1
    fi
}

# Function to check OS version
check_os_version() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        if [ "$ID" = "ubuntu" ]; then
            version=$(echo "$VERSION_ID" | awk -F. '{print $1}')
            if [ "$version" -lt 20 ]; then
                log "error" "This script requires Ubuntu 20.04 or newer"
                exit 1
            fi
            log "info" "Detected Ubuntu $VERSION_ID"
        elif [ "$ID" = "debian" ]; then
            version=$(echo "$VERSION_ID" | awk -F. '{print $1}')
            if [ "$version" -lt 12 ]; then
                log "error" "This script requires Debian 12 or newer"
                exit 1
            fi
            log "info" "Detected Debian $VERSION_ID"
        else
            log "error" "This script requires Ubuntu 20.04+ or Debian 12+"
            exit 1
        fi
    else
        log "error" "Cannot determine OS version"
        exit 1
    fi
}

# Function to get system architecture
get_arch() {
    arch=$(uname -m)
    case $arch in
        x86_64)
            echo "amd64"
            ;;
        aarch64)
            echo "arm64"
            ;;
        *)
            log "error" "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Function to create required directories
create_directories() {
    if [ -d "${INSTALL_DIR}" ]; then
        log "info" "Directory ${INSTALL_DIR} already exists"
    else
        mkdir -p "${INSTALL_DIR}"
        log "info" "Created directory ${INSTALL_DIR}"
    fi
}

# Function to create empty config file
create_config_file() {
    if [ ! -f "${CONFIG_FILE}" ]; then
        log "info" "Creating empty configuration file..."
        echo "{}" > "${CONFIG_FILE}"
        chmod 600 "${CONFIG_FILE}"
    else
        log "info" "Configuration file already exists"
    fi
}

# Function to handle agent binary download and installation
handle_agent_binary() {
    local service_was_running="$1"
    
    # Get system architecture
    local arch
    arch=$(get_arch)
    log "info" "Detected architecture: $arch"
    
    # Create a temporary file for downloading
    local temp_binary
    temp_binary=$(mktemp)
    
    # Get the latest release download URL for the current architecture
    log "info" "Fetching latest release information..."
    local download_url
    download_url=$(curl -s "${GITHUB_API}" | \
                  grep -o "\"browser_download_url\": \"[^\"]*linux-${arch}[^\"]*\"" | \
                  head -n 1 | \
                  cut -d'"' -f4)

    if [ -z "${download_url}" ]; then
        log "error" "Could not find release for linux-${arch}"
        rm -f "${temp_binary}"
        if [ "${service_was_running}" = true ]; then
            log "info" "Restarting Winterflow Agent service..."
            systemctl start winterflow-agent
        fi
        return 1
    fi
    
    log "info" "Downloading Winterflow Agent from ${download_url}"
    if ! curl -L -f -S --progress-bar -o "${temp_binary}" "${download_url}"; then
        log "error" "Failed to download the agent binary"
        rm -f "${temp_binary}"
        if [ "${service_was_running}" = true ]; then
            log "info" "Restarting Winterflow Agent service..."
            systemctl start winterflow-agent
        fi
        return 1
    fi

    if [ ! -s "${temp_binary}" ]; then
        log "error" "Downloaded file is empty"
        rm -f "${temp_binary}"
        if [ "${service_was_running}" = true ]; then
            log "info" "Restarting Winterflow Agent service..."
            systemctl start winterflow-agent
        fi
        return 1
    fi

    # Make the temporary binary executable
    chmod +x "${temp_binary}"
    
    # Verify the binary is executable
    if ! [ -x "${temp_binary}" ]; then
        log "error" "Failed to make the binary executable"
        rm -f "${temp_binary}"
        if [ "${service_was_running}" = true ]; then
            log "info" "Restarting Winterflow Agent service..."
            systemctl start winterflow-agent
        fi
        return 1
    fi

    # Move the temporary binary to the final location
    mv "${temp_binary}" "${AGENT_BINARY}"
    
    log "info" "Agent binary successfully installed"
    return 0
}

# Function to manage systemd service
manage_systemd_service() {
    local service_was_running="$1"
    
    # Create systemd service file if it doesn't exist
    if [ ! -f "${SERVICE_FILE}" ]; then
        log "info" "Creating systemd service..."
        cat > "${SERVICE_FILE}" << EOF
[Unit]
Description=Winterflow Agent
After=network.target

[Service]
Type=simple
ExecStart=${AGENT_BINARY} --config ${CONFIG_FILE}
Restart=always
RestartSec=10
User=root
Group=root
WorkingDirectory=${INSTALL_DIR}
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=winterflow-agent

[Install]
WantedBy=multi-user.target
EOF
    else
        log "info" "Systemd service file already exists, preserving existing configuration"
    fi

    # Reload systemd
    log "info" "Reloading systemd configuration..."
    systemctl daemon-reload

    # Restart service if it was running before
    if [ "$service_was_running" = true ]; then
        log "info" "Restarting Winterflow Agent service..."
        systemctl start winterflow-agent
    fi
}

# Function to display next steps
display_next_steps() {
    log "info" "Installation completed successfully!"
    echo ""
    echo "Next steps:"
    echo "1. Configure your agent by editing ${CONFIG_FILE}"
    echo "2. Start the agent with: sudo systemctl start winterflow-agent"
    echo "3. Enable auto-start on boot with: sudo systemctl enable winterflow-agent"
    echo "4. Check agent status with: sudo systemctl status winterflow-agent"
    echo ""
    echo "For more information, visit: https://docs.winterflow.com"
}

# Function to save installation script
save_installation_script() {
    log "info" "Saving installation script to ${INSTALL_SCRIPT}"
    
    # Download the installation script
    if ! curl -fsSL "${INSTALL_SCRIPT_URL}" > "${INSTALL_SCRIPT}"; then
        log "error" "Failed to download installation script"
        return 1
    fi
    
    # Make the saved script executable
    chmod +x "${INSTALL_SCRIPT}"
    log "info" "Installation script saved and made executable"
}

# Main installation process
log "info" "Starting Winterflow Agent installation..."

# Check if running as root
check_root

# Check OS version
log "info" "Checking OS version..."
check_os_version

# Update package repositories
log "info" "Updating package repositories..."
apt-get update

# Install required packages (only curl now)
log "info" "Installing required packages..."
apt-get install -y ${REQUIRED_PACKAGES}

# Create required directories
create_directories

# Create empty config file
create_config_file

# Check if service exists and is running
SERVICE_WAS_RUNNING=false
if systemctl is-active --quiet winterflow-agent; then
    log "info" "Stopping Winterflow Agent service..."
    systemctl stop winterflow-agent
    SERVICE_WAS_RUNNING=true
elif systemctl is-enabled --quiet winterflow-agent 2>/dev/null; then
    log "info" "Winterflow Agent service is registered but not running"
fi

# Handle agent binary download and installation
if ! handle_agent_binary "${SERVICE_WAS_RUNNING}"; then
    log "error" "Failed to install Winterflow Agent"
    exit 1
fi

# Manage systemd service
manage_systemd_service "${SERVICE_WAS_RUNNING}"

# Save installation and uninstallation scripts for future use
if ! save_installation_script; then
    log "warn" "Failed to save installation script, but agent installation was successful"
fi

# Display next steps
display_next_steps
