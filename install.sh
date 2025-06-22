#!/bin/bash

# Winterflow Agent Installer
# -------------------------
#
# Quick Install:
#   curl -fsSL https://get.winterflow.io/agent | sudo bash
#
# Manual Install:
#   curl -fsSL https://get.winterflow.io/agent > winterflow-install.sh
#   chmod +x winterflow-install.sh
#   sudo ./winterflow-install.sh
#
# Source Code: https://github.com/flowmitry/winterflow-agent
# Website: https://winterflow.io

# Exit on any error
set -e

# Cleanup function
cleanup() {
    if [ -n "${TEMP_AGENT_BINARY}" ] && [ -f "${TEMP_AGENT_BINARY}" ]; then
        log "info" "Cleaning up temporary files..."
        rm -f "${TEMP_AGENT_BINARY}"
    fi
}

# Set trap for cleanup on script exit
trap cleanup EXIT

#######################
# Configuration Variables
#######################

# Installation paths
INSTALL_DIR="/opt/winterflow"
AGENT_BINARY="${INSTALL_DIR}/agent"
CONFIG_FILE="${INSTALL_DIR}/agent.config.json"
SERVICE_FILE="/etc/systemd/system/winterflow-agent.service"
LOGS_DIR="/var/log/winterflow/"

# URLs
GITHUB_API="https://api.github.com/repos/flowmitry/winterflow-agent/releases"

# Required packages (fail if not installed)
REQUIRED_PACKAGES="curl jq"

# User settings
USER="winterflow"

# Temporary file for downloaded binary
TEMP_AGENT_BINARY=""

# ----------------------
# Orchestrator Selection
# ----------------------

# Global variable that will hold user-selected orchestrator. Default is docker_compose
ORCHESTRATOR="docker_compose"

# Ask the user which orchestrator they want to use unless a non-empty config already exists.
ask_orchestrator() {
    if [ -f "${CONFIG_FILE}" ] && [ -s "${CONFIG_FILE}" ]; then
        log "info" "Existing config detected - skipping orchestrator selection prompt"
        return 0
    fi

    read -r -p "Choose orchestrator [docker_compose/docker_swarm] (default: docker_compose): " input_orch
    case "$input_orch" in
        docker_swarm)
            ORCHESTRATOR="docker_swarm";;
        ""|docker_compose)
            ORCHESTRATOR="docker_compose";;
        *)
            log "warn" "Unknown orchestrator '$input_orch'. Using default 'docker_compose'.";;
    esac
    log "info" "Selected orchestrator: ${ORCHESTRATOR}"
}

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
        log "info" "Detected OS: $ID $VERSION_ID"
    else
        log "info" "OS release information not available - continuing"
    fi
    return 0
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

    # Create logs directory if it doesn't exist
    if [ -d "${LOGS_DIR}" ]; then
        log "info" "Directory ${LOGS_DIR} already exists"
    else
        mkdir -p "${LOGS_DIR}"
        log "info" "Created directory ${LOGS_DIR}"
    fi

    # Change ownership to service user
    if id "${USER}" &>/dev/null; then
        chown -R ${USER}:${USER} "${INSTALL_DIR}"
        chmod 755 "${INSTALL_DIR}"
        log "info" "Changed ownership of ${INSTALL_DIR} to ${USER} user"
        
        chown -R ${USER}:${USER} "${LOGS_DIR}"
        chmod 755 "${LOGS_DIR}"
        log "info" "Changed ownership of ${LOGS_DIR} to ${USER} user"
    else
        log "warn" "Could not change ownership of directories: ${USER} user does not exist"
    fi
}

# Function to create empty config file
create_config_file() {
    if [ ! -f "${CONFIG_FILE}" ]; then
        log "info" "Creating empty configuration file..."
        echo "{}" > "${CONFIG_FILE}"
        chmod 600 "${CONFIG_FILE}"
        
        # Set ownership to service user
        if id "${USER}" &>/dev/null; then
            chown ${USER}:${USER} "${CONFIG_FILE}"
            log "info" "Changed ownership of ${CONFIG_FILE} to ${USER} user"
        else
            log "warn" "Could not change ownership of config file: ${USER} user does not exist"
        fi
    else
        log "info" "Configuration file already exists"
        
        # Ensure existing config file has correct ownership
        if id "${USER}" &>/dev/null; then
            chown ${USER}:${USER} "${CONFIG_FILE}"
            log "info" "Ensured ownership of ${CONFIG_FILE} is set to ${USER} user"
        fi
    fi
}

# Function to download agent binary to /tmp/
download_agent_binary() {
    # Get system architecture
    local arch
    arch=$(get_arch)
    log "info" "Detected architecture: $arch (from uname -m: $(uname -m))"

    # Set temporary file path
    TEMP_AGENT_BINARY="/tmp/winterflow-agent-$(date +%s)"

    # Get the latest stable release download URL for the current architecture (ignoring pre-releases)
    log "info" "Fetching latest stable release information..."
    local download_url
    
    # Try multiple possible binary name patterns
    local binary_patterns=(
        "winterflow-agent-linux-${arch}"
        "winterflow-agent_linux_${arch}"
        "winterflow-agent-${arch}"
        "agent-linux-${arch}"
        "agent_linux_${arch}"
        "agent-${arch}"
    )
    
    # Get all releases and filter for non-prerelease
    local releases_json
    releases_json=$(curl -s "${GITHUB_API}")
    
    # Check if we got valid JSON
    if [ -z "${releases_json}" ]; then
        log "error" "Failed to fetch release information from GitHub API - empty response"
        log "info" "Please check your internet connection and try again"
        return 1
    fi
    
    if ! echo "${releases_json}" | grep -q '"tag_name"'; then
        log "error" "Invalid response from GitHub API"
        log "info" "Response received:"
        echo "${releases_json}" | head -5
        return 1
    fi
    
    # Try to find a matching binary for each pattern
    for pattern in "${binary_patterns[@]}"; do
        log "info" "Searching for binary pattern: ${pattern}"
        
        # Use jq if available for more reliable JSON parsing
        if command -v jq >/dev/null 2>&1; then
            download_url=$(echo "${releases_json}" | \
                          jq -r --arg pattern "${pattern}" \
                          '.[] | select(.prerelease == false) | .assets[] | select(.name | contains($pattern)) | .browser_download_url' | \
                          head -n 1)
        else
            # Fallback to grep method
            download_url=$(echo "${releases_json}" | \
                          grep -v '"prerelease": true' | \
                          grep -o "\"browser_download_url\": \"[^\"]*${pattern}[^\"]*\"" | \
                          head -n 1 | \
                          cut -d '"' -f4)
        fi
        
        if [ -n "${download_url}" ] && [ "${download_url}" != "null" ]; then
            log "info" "Found matching release with pattern: ${pattern}"
            log "info" "Download URL: ${download_url}"
            break
        fi
    done

    if [ -z "${download_url}" ]; then
        log "error" "Could not find release for any of the expected binary patterns"
        log "info" "Searched for patterns:"
        for pattern in "${binary_patterns[@]}"; do
            echo "  - ${pattern}"
        done
        log "info" "Available releases:"
        if command -v jq >/dev/null 2>&1; then
            echo "${releases_json}" | \
                jq -r '.[] | select(.prerelease == false) | .assets[].browser_download_url' | \
                head -n 10
        else
            echo "${releases_json}" | \
                grep -v '"prerelease": true' | \
                grep -o "\"browser_download_url\": \"[^\"]*\"" | \
                head -n 10 | \
                cut -d '"' -f4
        fi
        return 1
    fi

    # Verify the download URL is accessible
    log "info" "Verifying download URL accessibility..."
    if ! curl -I -f -s "${download_url}" >/dev/null; then
        log "error" "Download URL is not accessible: ${download_url}"
        log "info" "Please check your internet connection or try again later"
        return 1
    fi

    log "info" "Downloading Winterflow Agent to ${TEMP_AGENT_BINARY}"
    if ! curl -L -f -S --progress-bar -o "${TEMP_AGENT_BINARY}" "${download_url}"; then
        log "error" "Failed to download the agent binary"
        log "info" "Please check your internet connection or try again later"
        rm -f "${TEMP_AGENT_BINARY}"
        return 1
    fi

    if [ ! -s "${TEMP_AGENT_BINARY}" ]; then
        log "error" "Downloaded file is empty"
        rm -f "${TEMP_AGENT_BINARY}"
        return 1
    fi

    # Make the temporary binary executable
    chmod +x "${TEMP_AGENT_BINARY}"

    # Verify the binary is executable
    if ! [ -x "${TEMP_AGENT_BINARY}" ]; then
        log "error" "Failed to make the binary executable"
        rm -f "${TEMP_AGENT_BINARY}"
        return 1
    fi

    log "info" "Agent binary successfully downloaded to ${TEMP_AGENT_BINARY}"
    return 0
}

# Function to handle agent binary installation from /tmp/
handle_agent_binary() {
    local service_was_running="$1"

    # Check if the temporary binary exists
    if [ ! -f "${TEMP_AGENT_BINARY}" ]; then
        log "error" "Temporary agent binary not found at ${TEMP_AGENT_BINARY}"
        if [ "${service_was_running}" = true ]; then
            log "info" "Restarting Winterflow Agent service..."
            systemctl start winterflow-agent
        fi
        return 1
    fi

    # Verify the binary is still executable
    if ! [ -x "${TEMP_AGENT_BINARY}" ]; then
        log "error" "Temporary agent binary is not executable"
        rm -f "${TEMP_AGENT_BINARY}"
        if [ "${service_was_running}" = true ]; then
            log "info" "Restarting Winterflow Agent service..."
            systemctl start winterflow-agent
        fi
        return 1
    fi

    log "info" "Installing agent binary from ${TEMP_AGENT_BINARY} to ${AGENT_BINARY}"
    
    # Move the temporary binary to the final location
    if ! mv "${TEMP_AGENT_BINARY}" "${AGENT_BINARY}"; then
        log "error" "Failed to move agent binary to final location"
        rm -f "${TEMP_AGENT_BINARY}"
        if [ "${service_was_running}" = true ]; then
            log "info" "Restarting Winterflow Agent service..."
            systemctl start winterflow-agent
        fi
        return 1
    fi

    # Set ownership and permissions for the agent binary
    if id "${USER}" &>/dev/null; then
        chown ${USER}:${USER} "${AGENT_BINARY}"
        chmod 755 "${AGENT_BINARY}"
        log "info" "Set ownership of ${AGENT_BINARY} to ${USER} user"
    else
        log "warn" "Could not change ownership of agent binary: ${USER} user does not exist"
        chmod 755 "${AGENT_BINARY}"
    fi

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
User=${USER}
Group=${USER}
WorkingDirectory=${INSTALL_DIR}
StandardOutput=${LOGS_DIR}/agent.log
StandardError=${LOGS_DIR}/agent_error.log
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

# Function to ensure all content in INSTALL_DIR is owned by USER
ensure_ownership() {
    log "info" "Ensuring all content in ${INSTALL_DIR} is owned by ${USER} user..."
    
    if id "${USER}" &>/dev/null; then
        # Set ownership recursively for the entire install directory
        chown -R ${USER}:${USER} "${INSTALL_DIR}"
        log "info" "Ensured ownership of all content in ${INSTALL_DIR} is set to ${USER} user"
        
        # Also ensure logs directory ownership
        if [ -d "${LOGS_DIR}" ]; then
            chown -R ${USER}:${USER} "${LOGS_DIR}"
            log "info" "Ensured ownership of ${LOGS_DIR} is set to ${USER} user"
        fi
        
        # Verify key files have correct ownership
        log "info" "Ownership verification for key files:"
        if [ -f "${AGENT_BINARY}" ]; then
            agent_perms=$(ls -la "${AGENT_BINARY}")
            log "info" "  Agent binary: ${agent_perms}"
        fi
        if [ -f "${CONFIG_FILE}" ]; then
            config_perms=$(ls -la "${CONFIG_FILE}")
            log "info" "  Config file: ${config_perms}"
        fi
        
        # Show directory contents ownership
        log "info" "All files in ${INSTALL_DIR}:"
        ls -la "${INSTALL_DIR}" | while read -r line; do
            log "info" "  ${line}"
        done
    else
        log "error" "Cannot set ownership: ${USER} user does not exist!"
        return 1
    fi
}

# Function to run agent registration
run_agent_registration() {
    log "info" "Installation completed successfully!"
    log "info" "Running agent registration as ${USER} user..."
    
    # Verify the agent binary exists and is executable
    if [ ! -f "${AGENT_BINARY}" ]; then
        log "error" "Agent binary not found at ${AGENT_BINARY}"
        return 1
    fi
    
    if [ ! -x "${AGENT_BINARY}" ]; then
        log "error" "Agent binary is not executable"
        return 1
    fi
    
    # Verify the config file exists
    if [ ! -f "${CONFIG_FILE}" ]; then
        log "error" "Config file not found at ${CONFIG_FILE}"
        return 1
    fi
    
    # Run the registration command as the winterflow user with proper working directory
    if sudo -u "${USER}" -H --set-home bash -c "cd ${INSTALL_DIR} && ${AGENT_BINARY} --register ${ORCHESTRATOR}"; then
        echo ""
    else
        log "warn" "Agent registration failed or was interrupted"
        echo ""
        echo "Manual steps:"
        echo "1. Run: sudo -u ${USER} ${AGENT_BINARY} --register ${ORCHESTRATOR}"
        echo "2. Visit https://app.winterflow.io to complete agent registration"
        echo ""
        return 1
    fi
}

# Function to create service user and group
create_service_user() {
    log "info" "Creating ${USER} user and group..."

    # Check if user already exists
    if id "${USER}" &>/dev/null; then
        log "info" "User ${USER} already exists"
    else
        # Create user with home directory
        useradd -m -s /bin/bash ${USER}
        log "info" "Created ${USER} user with home directory"
    fi
}

# Function to verify required packages are installed
check_required_packages() {
    local missing=false

    # Base requirements
    for cmd in $REQUIRED_PACKAGES docker; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            log "error" "Required command '$cmd' is not installed."
            missing=true
        fi
    done

    # Orchestrator-specific requirements
    if [ "$ORCHESTRATOR" = "docker_compose" ]; then
        if command -v docker-compose >/dev/null 2>&1; then
            : # ok
        elif docker compose version >/dev/null 2>&1 2>/dev/null; then
            : # docker compose plugin available
        else
            log "error" "Docker Compose is required but was not found (checked 'docker-compose' binary and 'docker compose' plugin)."
            missing=true
        fi
    fi

    if [ "$missing" = true ]; then
        log "error" "Please install the missing dependencies and run the installer again."
        exit 1
    fi

    log "info" "All required software dependencies are installed."
}

# Main installation process
log "info" "Starting Winterflow Agent installation..."

# Check if running as root
check_root

# Check OS version
log "info" "Checking OS version..."
check_os_version

# Ask for orchestrator early so we can verify related dependencies
ask_orchestrator

# Verify required packages are installed
check_required_packages

# Download agent binary early
log "info" "Downloading Winterflow Agent binary..."
if ! download_agent_binary; then
    log "error" "Failed to download Winterflow Agent binary"
    exit 1
fi

# Create service user and group
create_service_user

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

# Ensure all content in INSTALL_DIR is owned by USER
ensure_ownership

# Start the service before registration
log "info" "Starting Winterflow Agent service..."
systemctl enable winterflow-agent
systemctl start winterflow-agent

# Run agent registration automatically
if ! run_agent_registration; then
    log "warn" "Installation completed but registration failed"
    log "info" "You can manually register the agent later using the instructions above"
else
    log "info" "Installation and registration completed successfully!"
fi

