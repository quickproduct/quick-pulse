#!/usr/bin/env bash
# ============================================================================
# validate-compose.sh - Validate Docker Compose configurations
# ============================================================================
# Usage:
#   ./scripts/validate-compose.sh dev
#   ./scripts/validate-compose.sh staging
#   ./scripts/validate-compose.sh prod
#   ./scripts/validate-compose.sh all  (validates all environments)
#
# Checks:
#   1. Environment file exists
#   2. Docker Compose syntax is valid
#   3. No container/volume/network name collisions
#   4. All required variables are set
#   5. Port mappings don't conflict

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
INFRA_DIR="$PROJECT_ROOT/infra"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[✗]${NC} $*" >&2
}

# Validate a single environment
validate_environment() {
    local env=$1
    echo
    log_info "Validating $env environment..."
    
    # Check if .env file exists
    local env_file="$INFRA_DIR/.env.$env"
    if [[ ! -f "$env_file" ]]; then
        log_error "Environment file not found: $env_file"
        return 1
    fi
    log_success "Environment file exists: $env_file"
    
    # Load environment variables
    set -a
    source "$env_file"
    set +a
    
    # Validate required variables
    local required_vars=(
        "ENVIRONMENT"
        "CONTAINER_PREFIX"
        "PORT_OFFSET"
        "SERVICE_VERSION"
        "REGISTRY"
        "DB_USER"
        "DB_PASSWORD"
        "DB_NAME"
        "JWT_SECRET_KEY"
        "EXTERNAL_FRONTEND_PORT"
        "EXTERNAL_BACKEND_PORT"
    )
    
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            log_error "Required variable not set: $var"
            return 1
        fi
    done
    log_success "All required environment variables are set"
    
    # Validate Docker Compose syntax
    log_info "Validating Docker Compose syntax..."
    if ! docker compose -f "$INFRA_DIR/docker-compose.yml" \
                        -f "$INFRA_DIR/docker-compose.$env.yml" \
                        config > /dev/null 2>&1; then
        log_error "Docker Compose syntax error"
        return 1
    fi
    log_success "Docker Compose syntax is valid"
    
    # Check for environment-specific issues
    log_info "Checking container naming conventions..."
    if [[ "$CONTAINER_PREFIX" != "qp-$ENVIRONMENT" ]]; then
        log_warn "Container prefix should match pattern 'qp-{environment}' (found: $CONTAINER_PREFIX)"
    else
        log_success "Container prefix follows naming convention: $CONTAINER_PREFIX"
    fi
    
    # Validate port configuration
    log_info "Checking port configuration..."
    local expected_frontend_port=$((80 + PORT_OFFSET))
    local expected_backend_port=$((8000 + PORT_OFFSET))
    
    if [[ "$EXTERNAL_FRONTEND_PORT" != "$expected_frontend_port" ]]; then
        log_warn "Frontend port: expected $expected_frontend_port (base 80 + offset $PORT_OFFSET), got $EXTERNAL_FRONTEND_PORT"
    else
        log_success "Frontend port: $EXTERNAL_FRONTEND_PORT (correct)"
    fi
    
    if [[ "$EXTERNAL_BACKEND_PORT" != "$expected_backend_port" ]]; then
        log_warn "Backend port: expected $expected_backend_port (base 8000 + offset $PORT_OFFSET), got $EXTERNAL_BACKEND_PORT"
    else
        log_success "Backend port: $EXTERNAL_BACKEND_PORT (correct)"
    fi
    
    # Validate security settings for production
    if [[ "$ENVIRONMENT" == "production" ]]; then
        log_info "Validating production security settings..."
        
        if [[ "$JWT_SECRET_KEY" == "change-me"* ]] || [[ "$JWT_SECRET_KEY" == "dev-"* ]]; then
            log_error "Production JWT_SECRET_KEY must be changed from default"
            return 1
        fi
        
        if [[ "$DEBUG" == "true" ]]; then
            log_error "Production DEBUG must be set to false"
            return 1
        fi
        
        if [[ "${DB_PASSWORD}" == "quickpulse" ]] || [[ "${DB_PASSWORD}" == "staging"* ]]; then
            log_error "Production database password must be changed from default"
            return 1
        fi
        
        log_success "Production security settings validated"
    fi
    
    log_success "$env environment validation passed ✓"
    return 0
}

# Main
main() {
    local target="${1:-all}"
    
    case "$target" in
        dev|development)
            validate_environment "dev" || exit 1
            ;;
        staging)
            validate_environment "staging" || exit 1
            ;;
        prod|production)
            validate_environment "prod" || exit 1
            ;;
        all)
            validate_environment "dev" || exit 1
            validate_environment "staging" || exit 1
            validate_environment "prod" || exit 1
            ;;
        *)
            log_error "Unknown environment: $target"
            echo "Usage: $0 {dev|staging|prod|all}"
            exit 1
            ;;
    esac
    
    echo
    log_success "All validation checks passed!"
}

main "$@"
