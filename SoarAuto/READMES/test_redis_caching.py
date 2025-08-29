#!/usr/bin/env python3
"""
Test script to verify Redis caching functionality for client integration configs
"""

import redis
import json
import time

def test_redis_caching():
    # Connect to Redis
    r = redis.Redis(host='localhost', port=6379, db=0, decode_responses=True)
    
    print("Testing Redis caching for client integration configs...")
    print("-" * 50)
    
    # List all keys before test
    print("\n1. Checking existing keys in Redis:")
    keys_before = r.keys("client:*:integration:*")
    if keys_before:
        print(f"   Found {len(keys_before)} existing keys:")
        for key in keys_before:
            print(f"   - {key}")
            try:
                value = r.get(key)
                if value:
                    config = json.loads(value)
                    print(f"     ClientID: {config.get('ClientID', 'N/A')}")
                    print(f"     Integration: {config.get('IntegrationName', 'N/A')}")
                    print(f"     Enabled: {config.get('Enabled', 'N/A')}")
            except Exception as e:
                print(f"     Error reading value: {e}")
    else:
        print("   No integration config keys found in Redis")
    
    print("\n2. Monitoring new keys (will check every 5 seconds for 30 seconds):")
    print("   Make an API request to trigger config retrieval...")
    
    start_time = time.time()
    checked_keys = set(keys_before)
    
    while time.time() - start_time < 30:
        time.sleep(5)
        current_keys = r.keys("client:*:integration:*")
        new_keys = set(current_keys) - checked_keys
        
        if new_keys:
            print(f"\n   NEW KEYS FOUND at {time.time() - start_time:.1f}s:")
            for key in new_keys:
                print(f"   + {key}")
                try:
                    value = r.get(key)
                    ttl = r.ttl(key)
                    if value:
                        config = json.loads(value)
                        print(f"     ClientID: {config.get('ClientID', 'N/A')}")
                        print(f"     Integration: {config.get('IntegrationName', 'N/A')}")
                        print(f"     Enabled: {config.get('Enabled', 'N/A')}")
                        print(f"     TTL: {ttl} seconds")
                except Exception as e:
                    print(f"     Error reading value: {e}")
            checked_keys.update(new_keys)
        else:
            print(f"   [{time.time() - start_time:.1f}s] No new keys yet...")
    
    print("\n3. Final key count:")
    final_keys = r.keys("client:*:integration:*")
    print(f"   Total integration config keys in Redis: {len(final_keys)}")
    
    if len(final_keys) > len(keys_before):
        print(f"   SUCCESS: {len(final_keys) - len(keys_before)} new keys were cached!")
    else:
        print("   WARNING: No new keys were cached during the test")

if __name__ == "__main__":
    try:
        test_redis_caching()
    except redis.ConnectionError:
        print("ERROR: Could not connect to Redis. Make sure Redis is running on localhost:6379")
    except Exception as e:
        print(f"ERROR: {e}")