#!/bin/bash
#
# SecAuto PostgreSQL Database Installation Script
#
# This script sets up the default soar_auto database in PostgreSQL with:
# - Database creation
# - User creation with proper permissions
# - Schema migrations
# - Initial configuration
#
# Usage:
#   ./install_postgresql.sh [options]
#
# Options:
#   --db-name <name>        Database name (default: soar_auto)
#   --db-user <user>        Database user (default: ddfelts)
#   --db-password <pass>    Database password (will prompt if not provided)
#   --db-host <host>        Database host (default: localhost)
#   --db-port <port>        Database port (default: 5432)
#   --admin-user <user>     PostgreSQL admin user (default: postgres)
#   --skip-create-db        Skip database creation (assume it exists)
#   --skip-create-user      Skip user creation (assume user exists)
#   --migrations-dir <dir>  Directory containing migrations (default: ../SoarAuto/migrations)
#   --help                  Show this help message

set -e

# Default configuration
DB_NAME="soar_auto"
DB_USER="ddfelts"
DB_PASSWORD=""
DB_HOST="localhost"
DB_PORT="5432"
ADMIN_USER="postgres"
SKIP_CREATE_DB=false
SKIP_CREATE_USER=false
MIGRATIONS_DIR=""
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_help() {
    cat << EOF
SecAuto PostgreSQL Database Installation Script

This script sets up the default soar_auto database in PostgreSQL with proper
schema, users, and permissions for SecAuto to function correctly.

Usage:
    $0 [options]

Options:
    --db-name <name>        Database name (default: soar_auto)
    --db-user <user>        Database user (default: ddfelts)  
    --db-password <pass>    Database password (will prompt if not provided)
    --db-host <host>        Database host (default: localhost)
    --db-port <port>        Database port (default: 5432)
    --admin-user <user>     PostgreSQL admin user (default: postgres)
    --skip-create-db        Skip database creation (assume it exists)
    --skip-create-user      Skip user creation (assume user exists)
    --migrations-dir <dir>  Directory containing migrations (auto-detected)
    --help                  Show this help message

Examples:
    # Basic installation with defaults
    $0

    # Custom database name and user
    $0 --db-name my_soar_auto --db-user myuser

    # Install on remote PostgreSQL server
    $0 --db-host 192.168.1.100 --admin-user postgres

    # Skip user creation (user already exists)
    $0 --skip-create-user --db-password mypassword

Environment Variables:
    PGPASSWORD                 PostgreSQL admin password
    SOAR_AUTO_DB_PASSWORD      Default database user password

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --db-name)
            DB_NAME="$2"
            shift 2
            ;;
        --db-user)
            DB_USER="$2"
            shift 2
            ;;
        --db-password)
            DB_PASSWORD="$2"
            shift 2
            ;;
        --db-host)
            DB_HOST="$2"
            shift 2
            ;;
        --db-port)
            DB_PORT="$2"
            shift 2
            ;;
        --admin-user)
            ADMIN_USER="$2"
            shift 2
            ;;
        --skip-create-db)
            SKIP_CREATE_DB=true
            shift
            ;;
        --skip-create-user)
            SKIP_CREATE_USER=true
            shift
            ;;
        --migrations-dir)
            MIGRATIONS_DIR="$2"
            shift 2
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Auto-detect migrations directory if not provided
if [[ -z "$MIGRATIONS_DIR" ]]; then
    # Try different possible locations
    POSSIBLE_DIRS=(
        "$SCRIPT_DIR/../SoarAuto/migrations"
        "$SCRIPT_DIR/../migrations"
        "$SCRIPT_DIR/../../SoarAuto/migrations"
        "$(pwd)/SoarAuto/migrations"
        "$(pwd)/migrations"
    )
    
    for dir in "${POSSIBLE_DIRS[@]}"; do
        if [[ -d "$dir" && -f "$dir/001_client_integration_configs.sql" ]]; then
            MIGRATIONS_DIR="$dir"
            break
        fi
    done
fi

if [[ -z "$MIGRATIONS_DIR" || ! -d "$MIGRATIONS_DIR" ]]; then
    log_error "Could not find migrations directory. Please specify with --migrations-dir"
    exit 1
fi

log_info "Using migrations directory: $MIGRATIONS_DIR"

# Prompt for password if not provided
if [[ -z "$DB_PASSWORD" && "$SKIP_CREATE_USER" == "false" ]]; then
    if [[ -n "$SOAR_AUTO_DB_PASSWORD" ]]; then
        DB_PASSWORD="$SOAR_AUTO_DB_PASSWORD"
        log_info "Using password from SOAR_AUTO_DB_PASSWORD environment variable"
    else
        echo -n "Enter password for database user '$DB_USER': "
        read -s DB_PASSWORD
        echo
        if [[ -z "$DB_PASSWORD" ]]; then
            log_error "Password is required"
            exit 1
        fi
    fi
fi

# Check if PostgreSQL tools are available
if ! command -v psql &> /dev/null; then
    log_error "psql command not found. Please install PostgreSQL client tools."
    exit 1
fi

# Test connection to PostgreSQL as admin user
log_info "Testing connection to PostgreSQL server at $DB_HOST:$DB_PORT..."
if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$ADMIN_USER" -c "SELECT version();" postgres > /dev/null 2>&1; then
    log_error "Cannot connect to PostgreSQL server. Please check:"
    log_error "  - PostgreSQL is running"
    log_error "  - Connection parameters are correct"
    log_error "  - Admin user '$ADMIN_USER' has proper permissions"
    log_error "  - PGPASSWORD environment variable is set (if needed)"
    exit 1
fi
log_success "Connected to PostgreSQL server successfully"

# Create database user if not skipping
if [[ "$SKIP_CREATE_USER" == "false" ]]; then
    log_info "Creating database user '$DB_USER'..."
    
    # Check if user already exists
    USER_EXISTS=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$ADMIN_USER" -tAc "SELECT 1 FROM pg_roles WHERE rolname='$DB_USER';" postgres)
    
    if [[ "$USER_EXISTS" == "1" ]]; then
        log_warning "User '$DB_USER' already exists. Updating password..."
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$ADMIN_USER" -c "ALTER USER \"$DB_USER\" WITH PASSWORD '$DB_PASSWORD';" postgres
    else
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$ADMIN_USER" -c "CREATE USER \"$DB_USER\" WITH PASSWORD '$DB_PASSWORD';" postgres
    fi
    
    log_success "Database user '$DB_USER' configured successfully"
fi

# Create database if not skipping
if [[ "$SKIP_CREATE_DB" == "false" ]]; then
    log_info "Creating database '$DB_NAME'..."
    
    # Check if database already exists
    DB_EXISTS=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$ADMIN_USER" -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME';" postgres)
    
    if [[ "$DB_EXISTS" == "1" ]]; then
        log_warning "Database '$DB_NAME' already exists. Skipping creation..."
    else
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$ADMIN_USER" -c "CREATE DATABASE \"$DB_NAME\" OWNER \"$DB_USER\";" postgres
        log_success "Database '$DB_NAME' created successfully"
    fi
fi

# Grant permissions to user
log_info "Granting permissions to user '$DB_USER' on database '$DB_NAME'..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$ADMIN_USER" -c "GRANT ALL PRIVILEGES ON DATABASE \"$DB_NAME\" TO \"$DB_USER\";" postgres
psql -h "$DB_HOST" -p "$DB_PORT" -U "$ADMIN_USER" -c "ALTER DATABASE \"$DB_NAME\" OWNER TO \"$DB_USER\";" postgres

# Run migrations
log_info "Running database migrations..."

# Set environment for user connection
export PGPASSWORD="$DB_PASSWORD"

# Process migration files in order
MIGRATION_FILES=($(find "$MIGRATIONS_DIR" -name "*.sql" | sort))

if [[ ${#MIGRATION_FILES[@]} -eq 0 ]]; then
    log_warning "No migration files found in $MIGRATIONS_DIR"
else
    for migration_file in "${MIGRATION_FILES[@]}"; do
        migration_name=$(basename "$migration_file")
        log_info "Running migration: $migration_name"
        
        if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -f "$migration_file" "$DB_NAME"; then
            log_success "Migration $migration_name completed successfully"
        else
            log_error "Migration $migration_name failed"
            exit 1
        fi
    done
fi

# Test database connection with created user
log_info "Testing database connection with user '$DB_USER'..."
if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "SELECT current_database(), current_user, version();" "$DB_NAME" > /dev/null 2>&1; then
    log_success "Database connection test successful"
else
    log_error "Database connection test failed"
    exit 1
fi

# Create a sample config snippet
log_info "Generating configuration snippet..."
cat << EOF

${GREEN}===============================================================================${NC}
${GREEN}                    SecAuto PostgreSQL Installation Complete!${NC}
${GREEN}===============================================================================${NC}

Database Configuration:
  Host:     $DB_HOST
  Port:     $DB_PORT  
  Database: $DB_NAME
  User:     $DB_USER

Add the following to your SecAuto config.yaml file:

database:
  postgres:
    host: "$DB_HOST"
    port: $DB_PORT
    database: "$DB_NAME"
    username: "$DB_USER"
    password: "$DB_PASSWORD"
    ssl_mode: "disable"
    encryption_key: "$(openssl rand -base64 32)"

${YELLOW}Security Recommendations:${NC}
1. Store the database password securely (consider using environment variables)
2. Enable SSL/TLS for production deployments
3. Configure firewall rules to restrict database access
4. Regularly backup your database
5. Monitor database performance and logs

${BLUE}Next Steps:${NC}
1. Update your SecAuto config.yaml with the database configuration above
2. Restart SecAuto to use the PostgreSQL database
3. Verify integration functionality

EOF

log_success "PostgreSQL installation completed successfully!"