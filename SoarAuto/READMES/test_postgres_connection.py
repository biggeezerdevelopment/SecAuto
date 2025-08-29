#!/usr/bin/env python3
"""
PostgreSQL Connection Test Script
Reads database configuration from SoarAuto/config.yaml
"""

import yaml
import psycopg2
from psycopg2 import sql
import sys
from datetime import datetime


def load_config(config_path="SoarAuto/config.yaml"):
    """Load configuration from YAML file"""
    try:
        with open(config_path, 'r') as f:
            config = yaml.safe_load(f)
        return config
    except FileNotFoundError:
        print(f"❌ Config file not found: {config_path}")
        sys.exit(1)
    except yaml.YAMLError as e:
        print(f"❌ Error parsing YAML: {e}")
        sys.exit(1)


def test_postgres_connection(config):
    """Test PostgreSQL connection using config from YAML"""
    
    # Extract PostgreSQL configuration
    db_config = config.get('database', {}).get('postgres', {})
    
    if not db_config:
        print("❌ No PostgreSQL configuration found in config.yaml")
        return False
    
    # Build connection parameters
    conn_params = {
        'host': db_config.get('host', 'localhost'),
        'port': db_config.get('port', 5432),
        'database': db_config.get('database', 'soar_auto'),
        'user': db_config.get('username', 'postgres'),
        'password': db_config.get('password', ''),
    }
    
    # Handle SSL mode
    ssl_mode = db_config.get('ssl_mode', 'disable')
    if ssl_mode != 'disable':
        conn_params['sslmode'] = ssl_mode
    
    print("🔍 PostgreSQL Connection Test")
    print("=" * 50)
    print(f"Host: {conn_params['host']}:{conn_params['port']}")
    print(f"Database: {conn_params['database']}")
    print(f"User: {conn_params['user']}")
    print(f"SSL Mode: {ssl_mode}")
    print("=" * 50)
    
    try:
        # Attempt to connect
        print("\n⏳ Attempting to connect...")
        conn = psycopg2.connect(**conn_params)
        cursor = conn.cursor()
        
        print("✅ Connection successful!")
        
        # Get database version
        cursor.execute("SELECT version();")
        version = cursor.fetchone()[0]
        print(f"\n📊 Database Version:\n{version.split(',')[0]}")
        
        # Get current database info
        cursor.execute("SELECT current_database(), current_user, pg_database_size(current_database());")
        db_info = cursor.fetchone()
        print(f"\n📁 Current Database: {db_info[0]}")
        print(f"👤 Current User: {db_info[1]}")
        print(f"💾 Database Size: {format_bytes(db_info[2])}")
        
        # List schemas
        cursor.execute("""
            SELECT schema_name 
            FROM information_schema.schemata 
            WHERE schema_name NOT IN ('pg_catalog', 'information_schema')
            ORDER BY schema_name;
        """)
        schemas = cursor.fetchall()
        print(f"\n📂 Available Schemas ({len(schemas)}):")
        for schema in schemas:
            print(f"  - {schema[0]}")
        
        # List tables in public schema
        cursor.execute("""
            SELECT table_name 
            FROM information_schema.tables 
            WHERE table_schema = 'public' 
            ORDER BY table_name;
        """)
        tables = cursor.fetchall()
        
        if tables:
            print(f"\n📋 Tables in 'public' schema ({len(tables)}):")
            for table in tables[:10]:  # Show first 10 tables
                # Get row count for each table
                cursor.execute(sql.SQL("SELECT COUNT(*) FROM {}").format(
                    sql.Identifier(table[0])
                ))
                count = cursor.fetchone()[0]
                print(f"  - {table[0]}: {count:,} rows")
            
            if len(tables) > 10:
                print(f"  ... and {len(tables) - 10} more tables")
        else:
            print("\n📋 No tables found in 'public' schema")
        
        # Test write permissions (create a test table and drop it)
        print("\n🔐 Testing write permissions...")
        test_table = f"test_connection_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
        try:
            cursor.execute(sql.SQL("""
                CREATE TABLE {} (
                    id SERIAL PRIMARY KEY,
                    test_field VARCHAR(100),
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                );
            """).format(sql.Identifier(test_table)))
            
            cursor.execute(sql.SQL("INSERT INTO {} (test_field) VALUES (%s)").format(
                sql.Identifier(test_table)
            ), ("test_value",))
            
            cursor.execute(sql.SQL("DROP TABLE {};").format(sql.Identifier(test_table)))
            conn.commit()
            print("✅ Write permissions confirmed (CREATE, INSERT, DROP)")
        except psycopg2.Error as e:
            conn.rollback()
            print(f"⚠️  Limited permissions: {str(e).split('DETAIL')[0].strip()}")
        
        # Close connection
        cursor.close()
        conn.close()
        
        print("\n✅ All tests completed successfully!")
        return True
        
    except psycopg2.OperationalError as e:
        print(f"\n❌ Connection failed: {e}")
        print("\nPossible causes:")
        print("1. PostgreSQL service is not running")
        print("2. Incorrect host/port configuration")
        print("3. Network/firewall issues")
        return False
        
    except psycopg2.ProgrammingError as e:
        print(f"\n❌ Database error: {e}")
        return False
        
    except Exception as e:
        print(f"\n❌ Unexpected error: {e}")
        return False


def format_bytes(bytes_value):
    """Format bytes to human readable size"""
    for unit in ['B', 'KB', 'MB', 'GB', 'TB']:
        if bytes_value < 1024.0:
            return f"{bytes_value:.2f} {unit}"
        bytes_value /= 1024.0
    return f"{bytes_value:.2f} PB"


def main():
    """Main function"""
    print("🐘 PostgreSQL Connection Test Tool")
    print("Reading configuration from: SoarAuto/config.yaml\n")
    
    # Load configuration
    config = load_config()
    
    # Test connection
    success = test_postgres_connection(config)
    
    # Exit with appropriate code
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()