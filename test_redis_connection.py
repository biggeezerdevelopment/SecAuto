#!/usr/bin/env python3
"""
Redis Connection Test Script
Reads database configuration from SoarAuto/config.yaml
"""

import yaml
import redis
import sys
from datetime import datetime
from urllib.parse import urlparse


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


def parse_redis_url(redis_url):
    """Parse Redis URL into connection parameters"""
    try:
        parsed = urlparse(redis_url)
        return {
            'host': parsed.hostname or 'localhost',
            'port': parsed.port or 6379,
            'db': int(parsed.path[1:]) if parsed.path and len(parsed.path) > 1 else 0,
            'password': parsed.password,
            'username': parsed.username
        }
    except Exception as e:
        print(f"❌ Error parsing Redis URL: {e}")
        return None


def test_redis_connection(config):
    """Test Redis connection using config from YAML"""
    
    # Extract Redis configuration
    db_config = config.get('database', {})
    cluster_config = config.get('cluster', {})
    
    # Get primary Redis URL
    redis_url = db_config.get('redis_url', 'redis://localhost:6379/0')
    
    # Get cluster Redis URL if available
    cluster_redis_url = cluster_config.get('redis_url', 'redis://localhost:6379/1')
    
    print("🔗 Redis Connection Test")
    print("=" * 50)
    
    # Test primary Redis connection
    success_primary = test_single_redis("Primary Redis", redis_url, config)
    
    # Test cluster Redis connection if different from primary
    success_cluster = True
    if cluster_redis_url != redis_url:
        print("\n" + "=" * 50)
        success_cluster = test_single_redis("Cluster Redis", cluster_redis_url, config)
    
    return success_primary and success_cluster


def test_single_redis(name, redis_url, config):
    """Test a single Redis connection"""
    print(f"\n🔍 {name} Connection Test")
    print(f"URL: {redis_url}")
    
    # Parse Redis URL
    conn_params = parse_redis_url(redis_url)
    if not conn_params:
        return False
    
    print(f"Host: {conn_params['host']}:{conn_params['port']}")
    print(f"Database: {conn_params['db']}")
    if conn_params['username']:
        print(f"Username: {conn_params['username']}")
    print("=" * 30)
    
    try:
        # Create Redis connection
        print("\n⏳ Attempting to connect...")
        r = redis.Redis(
            host=conn_params['host'],
            port=conn_params['port'],
            db=conn_params['db'],
            password=conn_params['password'],
            username=conn_params['username'],
            decode_responses=True,
            socket_timeout=5,
            socket_connect_timeout=5
        )
        
        # Test connection with ping
        r.ping()
        print("✅ Connection successful!")
        
        # Get Redis info
        info = r.info()
        print(f"\n📊 Redis Server Info:")
        print(f"Version: {info['redis_version']}")
        print(f"Mode: {info['redis_mode']}")
        print(f"OS: {info['os']}")
        print(f"Architecture: {info['arch_bits']}-bit")
        print(f"Process ID: {info['process_id']}")
        print(f"Uptime: {format_uptime(info['uptime_in_seconds'])}")
        
        # Memory information
        print(f"\n💾 Memory Usage:")
        print(f"Used Memory: {format_bytes(info['used_memory'])}")
        print(f"Used Memory RSS: {format_bytes(info['used_memory_rss'])}")
        print(f"Max Memory: {format_bytes(info.get('maxmemory', 0)) if info.get('maxmemory', 0) > 0 else 'Unlimited'}")
        
        # Database information
        print(f"\n📁 Database Info:")
        total_keys = 0
        for db_num in range(16):  # Redis typically has 16 databases by default
            keys_info = info.get(f'db{db_num}')
            if keys_info:
                keys = keys_info['keys']
                expires = keys_info.get('expires', 0)
                avg_ttl = keys_info.get('avg_ttl', 0)
                print(f"  DB{db_num}: {keys:,} keys, {expires:,} with expiry")
                if avg_ttl > 0:
                    print(f"         Average TTL: {format_uptime(avg_ttl / 1000)}")
                total_keys += keys
        
        if total_keys == 0:
            print("  No databases with keys found")
        else:
            print(f"  Total Keys: {total_keys:,}")
        
        # Connection information
        print(f"\n🔗 Connection Info:")
        print(f"Connected Clients: {info['connected_clients']}")
        print(f"Total Connections Received: {info['total_connections_received']:,}")
        print(f"Total Commands Processed: {info['total_commands_processed']:,}")
        
        # Test basic operations
        print(f"\n🧪 Testing Basic Operations...")
        test_key = f"test_connection_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
        
        try:
            # Test SET
            r.set(test_key, "test_value", ex=60)  # Expire in 60 seconds
            print("✅ SET operation successful")
            
            # Test GET
            value = r.get(test_key)
            if value == "test_value":
                print("✅ GET operation successful")
            else:
                print("⚠️ GET operation returned unexpected value")
            
            # Test EXISTS
            if r.exists(test_key):
                print("✅ EXISTS operation successful")
            else:
                print("⚠️ EXISTS operation failed")
            
            # Test TTL
            ttl = r.ttl(test_key)
            if ttl > 0:
                print(f"✅ TTL operation successful (expires in {ttl}s)")
            else:
                print("⚠️ TTL operation returned unexpected value")
            
            # Test DELETE
            r.delete(test_key)
            if not r.exists(test_key):
                print("✅ DELETE operation successful")
            else:
                print("⚠️ DELETE operation failed")
                
        except redis.RedisError as e:
            print(f"⚠️ Basic operations test failed: {e}")
        
        # Test configuration settings from YAML
        print(f"\n⚙️ Configuration Validation:")
        db_config = config.get('database', {})
        
        # Check TTL settings
        cache_ttl = db_config.get('cache_ttl', 3600)
        job_ttl = db_config.get('job_ttl', 86400)
        temp_data_ttl = db_config.get('temp_data_ttl', 300)
        
        print(f"Cache TTL: {format_uptime(cache_ttl)}")
        print(f"Job TTL: {format_uptime(job_ttl)}")
        print(f"Temp Data TTL: {format_uptime(temp_data_ttl)}")
        
        # Test cluster configuration if applicable
        if 'cluster' in config and config['cluster'].get('enabled', False):
            cluster_config = config['cluster']
            print(f"\n🔗 Cluster Configuration:")
            print(f"Node ID: {cluster_config.get('node_id', 'N/A')}")
            print(f"Cluster Name: {cluster_config.get('cluster_name', 'N/A')}")
            print(f"Heartbeat Interval: {cluster_config.get('heartbeat_interval', 30)}s")
            print(f"Election Timeout: {cluster_config.get('election_timeout', 300)}s")
        
        print(f"\n✅ {name} test completed successfully!")
        return True
        
    except redis.ConnectionError as e:
        print(f"\n❌ Connection failed: {e}")
        print("\nPossible causes:")
        print("1. Redis service is not running")
        print("2. Incorrect host/port configuration")
        print("3. Network/firewall issues")
        print("4. Redis requires authentication but none provided")
        return False
        
    except redis.AuthenticationError as e:
        print(f"\n❌ Authentication failed: {e}")
        print("Check Redis password/username in configuration")
        return False
        
    except redis.ResponseError as e:
        print(f"\n❌ Redis error: {e}")
        return False
        
    except Exception as e:
        print(f"\n❌ Unexpected error: {e}")
        return False


def format_bytes(bytes_value):
    """Format bytes to human readable size"""
    if bytes_value == 0:
        return "0 B"
    
    for unit in ['B', 'KB', 'MB', 'GB', 'TB']:
        if bytes_value < 1024.0:
            return f"{bytes_value:.2f} {unit}"
        bytes_value /= 1024.0
    return f"{bytes_value:.2f} PB"


def format_uptime(seconds):
    """Format seconds to human readable uptime"""
    if seconds < 60:
        return f"{seconds:.0f}s"
    elif seconds < 3600:
        minutes = seconds / 60
        return f"{minutes:.1f}m"
    elif seconds < 86400:
        hours = seconds / 3600
        return f"{hours:.1f}h"
    else:
        days = seconds / 86400
        return f"{days:.1f}d"


def main():
    """Main function"""
    print("🔗 Redis Connection Test Tool")
    print("Reading configuration from: SoarAuto/config.yaml\n")
    
    # Load configuration
    config = load_config()
    
    # Test connection
    success = test_redis_connection(config)
    
    # Exit with appropriate code
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()