#!/bin/bash

# IP Marketplace Database Restore Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${PROJECT_ROOT}/backups"

# Load environment variables
if [ -f "${PROJECT_ROOT}/.env" ]; then
    source "${PROJECT_ROOT}/.env"
fi

DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_NAME=${DB_NAME:-ip_marketplace}

echo "🔄 IP Marketplace Database Restore"

show_usage() {
    echo "Usage: $0 [OPTIONS] <backup_file>"
    echo ""
    echo "Options:"
    echo "  -h, --help           Show this help message"
    echo "  --list              List available backups"
    echo "  --latest            Restore from latest backup"
    echo "  --force             Skip confirmation prompt"
    echo "  --create-db         Create database if it doesn't exist"
    echo ""
    echo "Examples:"
    echo "  $0 --list"
    echo "  $0 --latest"
    echo "  $0 backups/db_backup_20240115_143000.sql.gz"
}

list_backups() {
    echo "📋 Available backups in $BACKUP_DIR:"
    echo ""
    
    if [ -d "$BACKUP_DIR" ]; then
        ls -lah "$BACKUP_DIR"/*.sql.gz 2>/dev/null | while read -r line; do
            echo "  $line"
        done
    else
        echo "  No backup directory found"
    fi
    
    echo ""
    echo "To restore from a backup:"
    echo "  $0 <backup_file>"
}

get_latest_backup() {
    find "$BACKUP_DIR" -name "db_backup_*.sql.gz" -type f -printf '%T@ %p\n' 2>/dev/null | \
        sort -n | tail -1 | cut -d' ' -f2-
}

create_database() {
    echo "🏗️  Creating database if not exists..."
    
    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d postgres \
        -c "CREATE DATABASE $DB_NAME;" 2>/dev/null || true
        
    echo "✅ Database creation command executed"
}

restore_database() {
    local backup_file="$1"
    local temp_file=""
    
    if [ ! -f "$backup_file" ]; then
        echo "❌ Backup file not found: $backup_file"
        exit 1
    fi
    
    echo "📂 Backup file: $backup_file"
    echo "🎯 Target database: $DB_HOST:$DB_PORT/$DB_NAME"
    echo ""
    
    # Check if file is compressed
    if [[ "$backup_file" == *.gz ]]; then
        echo "📦 Decompressing backup file..."
        temp_file="/tmp/restore_$(date +%s).sql"
        gunzip -c "$backup_file" > "$temp_file"
        backup_file="$temp_file"
    fi
    
    # Get file size
    local file_size=$(du -h "$backup_file" | cut -f1)
    echo "📊 Backup size: $file_size"
    
    # Drop existing connections
    echo "🔌 Dropping existing database connections..."
    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d postgres \
        -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$DB_NAME' AND pid <> pg_backend_pid();" \
        2>/dev/null || true
    
    # Restore database
    echo "🔄 Restoring database..."
    echo "⏳ This may take several minutes for large databases..."
    
    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        -f "$backup_file" \
        --quiet
    
    if [ $? -eq 0 ]; then
        echo "✅ Database restored successfully"
        
        # Cleanup temp file
        [ -n "$temp_file" ] && [ -f "$temp_file" ] && rm "$temp_file"
        
        # Run post-restore tasks
        post_restore_tasks
    else
        echo "❌ Database restore failed"
        [ -n "$temp_file" ] && [ -f "$temp_file" ] && rm "$temp_file"
        exit 1
    fi
}

post_restore_tasks() {
    echo "🔧 Running post-restore tasks..."
    
    # Update sequences (in case of partial restore)
    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        -c "SELECT setval(pg_get_serial_sequence(schemaname||'.'||tablename, columnname), COALESCE(max_value, 1)) FROM (SELECT schemaname, tablename, columnname, COALESCE(MAX(column_default::text::int), 0) as max_value FROM information_schema.columns c JOIN information_schema.tables t ON c.table_name = t.table_name WHERE column_default LIKE 'nextval%' GROUP BY schemaname, tablename, columnname) as temp;" \
        2>/dev/null || true
    
    # Analyze database
    echo "📊 Analyzing database statistics..."
    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        -c "ANALYZE;" \
        --quiet
    
    echo "✅ Post-restore tasks completed"
}

verify_restore() {
    echo "🔍 Verifying restore..."
    
    # Check if critical tables exist
    local tables=("users" "ip_assets" "products" "transactions")
    
    for table in "${tables[@]}"; do
        local count=$(PGPASSWORD="$DB_PASSWORD" psql \
            -h "$DB_HOST" \
            -p "$DB_PORT" \
            -U "$DB_USER" \
            -d "$DB_NAME" \
            -t -c "SELECT COUNT(*) FROM $table;" 2>/dev/null | xargs)
        
        if [ -n "$count" ] && [ "$count" -ge 0 ]; then
            echo "  ✅ Table '$table': $count records"
        else
            echo "  ❌ Table '$table': Error or missing"
        fi
    done
}

confirm_restore() {
    local backup_file="$1"
    
    echo "⚠️  WARNING: This will overwrite the current database!"
    echo "📂 Backup file: $backup_file"
    echo "�� Target database: $DB_HOST:$DB_PORT/$DB_NAME"
    echo ""
    read -p "Are you sure you want to continue? (yes/no): " -r
    echo ""
    
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        echo "❌ Restore cancelled"
        exit 1
    fi
}

# Parse arguments
FORCE=false
CREATE_DB=false
BACKUP_FILE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        --list)
            list_backups
            exit 0
            ;;
        --latest)
            BACKUP_FILE=$(get_latest_backup)
            if [ -z "$BACKUP_FILE" ]; then
                echo "❌ No backups found"
                exit 1
            fi
            echo "📋 Latest backup: $BACKUP_FILE"
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --create-db)
            CREATE_DB=true
            shift
            ;;
        -*)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
        *)
            BACKUP_FILE="$1"
            shift
            ;;
    esac
done

# Check dependencies
if ! command -v psql &> /dev/null; then
    echo "❌ psql not found. Please install PostgreSQL client tools."
    exit 1
fi

if [ -z "$DB_PASSWORD" ]; then
    echo "❌ DB_PASSWORD not set. Please configure your environment variables."
    exit 1
fi

# Validate backup file
if [ -z "$BACKUP_FILE" ]; then
    echo "❌ No backup file specified"
    echo ""
    show_usage
    exit 1
fi

# Convert relative path to absolute
if [[ ! "$BACKUP_FILE" = /* ]]; then
    BACKUP_FILE="$PROJECT_ROOT/$BACKUP_FILE"
fi

# Main execution
echo "🔄 Starting restore process at $(date)"

if [ "$CREATE_DB" = true ]; then
    create_database
fi

if [ "$FORCE" = false ]; then
    confirm_restore "$BACKUP_FILE"
fi

restore_database "$BACKUP_FILE"
verify_restore

echo ""
echo "🎉 Restore completed successfully at $(date)"
echo "🔍 Please verify your application is working correctly"
