#!/bin/bash

# Exit on any error
set -e

#######################
# Configuration Variables
#######################

# Installation paths
INSTALL_DIR="/opt/winterflow"
AGENT_BINARY="${INSTALL_DIR}/agent"
SERVICE_FILE="/etc/systemd/system/winterflow-agent.service"
PLAYBOOKS_DIR="${INSTALL_DIR}/playbooks"

# API endpoint for downloading the agent
DOWNLOAD_API_URL="https://get.winterflow.com/agent"

# Repository URLs (using HTTPS for better compatibility)
PLAYBOOKS_REPO="https://github.com/winterflowio/winterflow-playbooks.git"

# Required packages
REQUIRED_PACKAGES="ansible git curl"

# Minimum required versions
MIN_UBUNTU_VERSION=20
MIN_DEBIAN_VERSION=12

# Colors for better readability
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

#######################
# Utility Functions
#######################

# Function to log messages
log() {
    local level="$1"
    local message="$2"
    
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

# Function to handle agent binary download and installation
handle_agent_binary() {
    local service_was_running="$1"
    
    # Get system architecture and OS
    local arch
    arch=$(get_arch)
    
    local os_id
    os_id=$(. /etc/os-release && echo "$ID")
    
    log "info" "Detected architecture: $arch"
    log "info" "Detected OS: $os_id"
    
    # Construct binary URL
    local binary_url="${DOWNLOAD_API_URL}?os=${os_id}&arch=${arch}"
    
    # Create a temporary file for downloading
    local temp_binary
    temp_binary=$(mktemp)
    
    log "info" "Downloading Winterflow Agent from $binary_url"
    if ! curl -L -f -S --progress-bar -o "${temp_binary}" "${binary_url}"; then
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
ExecStart=${AGENT_BINARY}
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

# Function to clone or update repositories
handle_repositories() {
    log "info" "Setting up playbook repository..."
    
    # Handle winterflow-playbooks repository
    if [ -d "${PLAYBOOKS_DIR}/.git" ]; then
        log "info" "Updating winterflow-playbooks repository..."
        cd "${PLAYBOOKS_DIR}" || return 1
        if ! git pull; then
            log "warn" "Failed to update winterflow-playbooks repository"
        fi
    else
        log "info" "Cloning winterflow-playbooks repository..."
        if ! git clone "${PLAYBOOKS_REPO}" "${PLAYBOOKS_DIR}"; then
            log "warn" "Failed to clone winterflow-playbooks repository"
        fi
    fi
    
    # Set proper permissions
    chown -R root:root "${PLAYBOOKS_DIR}"
    
    return 0
}

# Function to display next steps
display_next_steps() {
    log "info" "Installation completed successfully!"
    echo ""
    echo "Next steps:"
    echo "1. Configure your agent by editing ${INSTALL_DIR}/config.yaml"
    echo "2. Start the agent with: sudo systemctl start winterflow-agent"
    echo "3. Enable auto-start on boot with: sudo systemctl enable winterflow-agent"
    echo "4. Check agent status with: sudo systemctl status winterflow-agent"
    echo ""
    echo "For more information, visit: https://docs.winterflow.com"
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

# Install required packages
log "info" "Installing required packages..."
apt-get install -y ${REQUIRED_PACKAGES}

# Create required directories
create_directories

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

# Clone or update repositories
handle_repositories

# Display next steps
display_next_steps
