import sys
import os
import json
sys.path.append(os.path.join(os.path.dirname(__file__), '..', 'integrations'))

from redis_integration import RedisIntegration


def main():
    active_users = 0
    inactive_users = 0
    redisconfig = context.get('redis', [])
    userlist = context.get('USER_LIST_OUTPUT').get('USER_LIST').get('USER')
    usernames = []
    for i in userlist:
        data = {"login":i.get('USER_LOGIN').get('#text'),
                "status":i.get('USER_STATUS').get('#text')}
        usernames.append(data)
    redis = RedisIntegration(redisconfig)
    if not usernames:
        result = {"usercontext": {
            'success': False,
            'error': 'No Usersseen',
            'results': [],
            'summary': {
                'active_users': 0,
                'inactive_users': 0,
            }
        }}
        return_context(result)
    else:
        for user in usernames:
            if user.get('status') == 'Active':
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
        redis.set_cache(f"qualys_user_count_{context.get('clientname')}", json.dumps(data))
        return_context(result)
main()