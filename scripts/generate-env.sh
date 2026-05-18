#!/usr/bin/env bash
# ============================================================================
# generate-env.sh - Generate environment files from templates
# ============================================================================
# Usage:
#   ./scripts/generate-env.sh dev          # Interactive generation for dev
#   ./scripts/generate-env.sh staging      # Interactive generation for staging
#   ./scripts/generate-env.sh prod         # Interactive generation for prod
#   ./scripts/generate-env.sh prod --no-interactive  # Use defaults
#
# This script helps generate .env files with secure defaults.

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

# Generate a random secret (base64 string)
generate_secret() {
    openssl rand -base64 32 | tr -d '=' | cut -c1-32
}

# Prompt for user input with default value
prompt() {
    local prompt_text=$1
    local default_value=$2
    local var_name=$3
    
    if [[ -n "${INTERACTIVE:-true}" ]]; then
        read -p "$(echo -e ${BLUE}${prompt_text}${NC}) [${default_value}]: " input
        input=${input:-$default_value}
    else
        input=$default_value
    fi
    
    eval "$var_name=\"$input\""
}

# Generate environment file
generate_env_file() {
    local env=$1
    local interactive=${2:-true}
    
    INTERACTIVE=$interactive
    
    log_info "Generating .env.$env file..."
    echo
    
    case "$env" in
        dev)
            log_info "Development environment configuration"
            echo "Using defaults for development..."
            cp "$INFRA_DIR/.env.dev" "$INFRA_DIR/.env.dev.backup.$(date +%s)" 2>/dev/null || true
            log_success ".env.dev is ready to use (already configured)"
            ;;
        staging)
            log_info "Staging environment configuration"
            echo "Staging setup requires manual configuration of secrets."
            echo
            
            prompt "Database password" "staging_secure_password_change_me" "db_password"
            prompt "JWT secret" "$(generate_secret)" "jwt_secret"
            prompt "Admin password" "staging_admin_password_change_me" "admin_password"
            
            # Update .env.staging
            local env_file="$INFRA_DIR/.env.staging"
            if [[ -f "$env_file" ]]; then
                cp "$env_file" "$env_file.backup.$(date +%s)"
            fi
            
            sed -i.bak "s/staging_secure_password_change_me/$db_password/g" "$env_file"
            sed -i.bak "s/staging-secret-key-change-before-production/$jwt_secret/g" "$env_file"
            sed -i.bak "s/staging_admin_password_change_me/$admin_password/g" "$env_file"
            
            rm -f "$env_file.bak"
            log_success ".env.staging updated with new secrets"
            ;;
        prod)
            log_info "Production environment configuration"
            log_warn "Production requires secure, unique secrets!"
            echo
            
            prompt "Production domain" "quickpulse.example.com" "prod_domain"
            prompt "Database password" "$(generate_secret)" "db_password"
            prompt "JWT secret" "$(generate_secret)" "jwt_secret"
            prompt "Admin email" "admin@$prod_domain" "admin_email"
            prompt "Admin password" "$(generate_secret)" "admin_password"
            
            # Update .env.prod
            local env_file="$INFRA_DIR/.env.prod"
            if [[ -f "$env_file" ]]; then
                cp "$env_file" "$env_file.backup.$(date +%s)"
            fi
            
            sed -i.bak "s|quickpulse.example.com|$prod_domain|g" "$env_file"
            sed -i.bak "s/CHANGE_ME_SECURE_PASSWORD_HERE/$db_password/g" "$env_file"
            sed -i.bak "s/CHANGE_ME_SECURE_JWT_KEY_HERE_MINIMUM_32_CHARS/$jwt_secret/g" "$env_file"
            sed -i.bak "s/admin@quickpulse.example.com/$admin_email/g" "$env_file"
            
            rm -f "$env_file.bak"
            log_success ".env.prod updated with new secrets"
            
            log_warn "IMPORTANT: Review .env.prod and ensure all values are set correctly before deploying!"
            ;;
        *)
            log_error "Unknown environment: $env"
            return 1
            ;;
    esac
    
    echo
    log_success "$env environment file generated: $INFRA_DIR/.env.$env"
}

# Main
main() {
    local target="${1:-all}"
    local interactive="${2:-true}"
    
    # Check for --no-interactive flag
    if [[ "${2:-}" == "--no-interactive" ]]; then
        interactive="false"
    fi
    
    case "$target" in
        dev|development)
            generate_env_file "dev" "$interactive"
            ;;
        staging)
            generate_env_file "staging" "$interactive"
            ;;
        prod|production)
            generate_env_file "prod" "$interactive"
            ;;
        all)
            generate_env_file "dev" "$interactive"
            generate_env_file "staging" "$interactive"
            generate_env_file "prod" "$interactive"
            ;;
        *)
            log_error "Unknown environment: $target"
            echo "Usage: $0 {dev|staging|prod|all} [--no-interactive]"
            exit 1
            ;;
    esac
}

main "$@"
