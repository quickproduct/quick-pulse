#!/usr/bin/env bash
# ============================================================================
# show-running-instances.sh - Display all QuickPulse Docker instances
# ============================================================================
# Usage:
#   ./scripts/show-running-instances.sh
#
# Shows:
#   - All running QuickPulse containers with their environment
#   - Container names, images, ports, and status
#   - Grouped by environment (dev, staging, prod)

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $*"
}

log_header() {
    echo
    echo -e "${CYAN}════════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}$*${NC}"
    echo -e "${CYAN}════════════════════════════════════════════════════════════════${NC}"
}

# Main
main() {
    log_header "QuickPulse Docker Instances"
    
    # Check if any QuickPulse containers are running
    local qp_containers=$(docker ps --filter "label=com.quickpulse.service" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}\t{{.Image}}" 2>/dev/null || echo "")
    
    if [[ -z "$qp_containers" ]]; then
        log_info "No QuickPulse containers are currently running"
        echo
        log_info "To start containers, run one of:"
        echo "  make up-dev       # Start development environment"
        echo "  make up-staging   # Start staging environment"
        echo "  make up-prod      # Start production environment"
        return 0
    fi
    
    # Group containers by environment
    local environments=$(docker ps --filter "label=com.quickpulse.service" --format "{{.Labels}}" | grep -o "com.quickpulse.environment=[^,]*" | cut -d= -f2 | sort | uniq)
    
    for env in $environments; do
        log_header "$env environment"
        
        # Get containers for this environment
        docker ps \
            --filter "label=com.quickpulse.environment=$env" \
            --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}" | \
            awk '{printf "%-30s %-40s %-30s %s\n", $1, $2, $3, $4}'
        
        # Get volumes for this environment
        local env_prefix="qp-$env"
        local volumes=$(docker volume ls --filter "name=^${env_prefix}" --format "table {{.Name}}\t{{.Driver}}" 2>/dev/null || echo "")
        
        if [[ -n "$volumes" ]]; then
            echo
            echo -e "${YELLOW}Volumes:${NC}"
            echo "$volumes" | awk '{printf "  %-40s %s\n", $1, $2}'
        fi
        
        # Get networks for this environment
        local networks=$(docker network ls --filter "name=^${env_prefix}" --format "table {{.Name}}\t{{.Driver}}" 2>/dev/null || echo "")
        
        if [[ -n "$networks" ]]; then
            echo
            echo -e "${YELLOW}Networks:${NC}"
            echo "$networks" | awk '{printf "  %-40s %s\n", $1, $2}'
        fi
    done
    
    echo
    log_header "Summary"
    
    # Count containers per environment
    for env in $environments; do
        local count=$(docker ps --filter "label=com.quickpulse.environment=$env" --format "table" | wc -l)
        count=$((count - 1))  # Subtract header line
        echo "  $(printf '%-12s' "$env"): $count containers running"
    done
    
    echo
    log_info "For detailed logs of a container, run:"
    echo "  docker logs -f <container_name>"
    echo
    log_info "To enter a container shell, run:"
    echo "  docker exec -it <container_name> /bin/bash"
}

main "$@"
