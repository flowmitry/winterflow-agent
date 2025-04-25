#!/bin/bash

# Winterflow Agent Uninstaller
# ---------------------------
# This script uninstalls the Winterflow Agent from your system
#
# Usage:
#   sudo ./uninstall.sh
#
# Source: https://github.com/winterflowio/agent

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

# Function to stop and disable the service
stop_service() {
    if systemctl is-active --quiet winterflow-agent; then
        log "info" "Stopping Winterflow Agent service..."
        systemctl stop winterflow-agent
    fi
    
    if systemctl is-enabled --quiet winterflow-agent 2>/dev/null; then
        log "info" "Disabling Winterflow Agent service..."
        systemctl disable winterflow-agent
    fi
}

# Function to remove systemd service file
remove_service_file() {
    if [ -f "${SERVICE_FILE}" ]; then
        log "info" "Removing systemd service file..."
        rm -f "${SERVICE_FILE}"
        systemctl daemon-reload
    else
        log "info" "Systemd service file not found"
    fi
}

# Main uninstallation process
log "info" "Starting Winterflow Agent uninstallation..."

# Check if running as root
check_root

# Stop and disable the service
stop_service

# Remove systemd service file
remove_service_file

log "info" "Uninstallation completed successfully!"
echo ""
echo "Note: You can remove the installation directory ${INSTALL_DIR} manually."
echo "If you want to reinstall the agent later, you can run the install script again."
echo ""
echo "For more information, visit: https://docs.winterflow.com" 