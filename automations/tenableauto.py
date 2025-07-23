import sys
import os
import json
sys.path.append(os.path.join(os.path.dirname(__file__), '..', 'integrations'))

from redis_integration import RedisIntegration


def main():
    active_users = 0
    inactive_users = 0
    redisconfig = context.get('redis', [])
    redis = RedisIntegration(redisconfig)
    users = context.get('users')
    if not users:
        result = {"usercontext": {
            'success': False,
            'error': 'No Usersseen',
            'results': [],
            'summary': {
                'active_users': 0,
                'inactive_users': 0,
            }
        }}
        #redis.set_cache(f"usercontext_{context.get('clientname')}", json.dumps(result))
        return_context(result)
    else:
        for user in users:
            if user.get('enabled') == True:
                active_users += 1
            else:
                inactive_users += 1
        
        result = {"usercontext": {
            'success': True,
            'summary': {
                'active_users': active_users,
                'inactive_users': inactive_users
            }
        }}
        data = {'active_users': active_users, 'inactive_users': inactive_users}
        redis.set_cache(f"tenable_user_count_{context.get('clientname')}", json.dumps(data))
        return_context(result)
main()