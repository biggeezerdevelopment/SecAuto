#!/usr/bin/env python3
"""
Debug script to test SoarBaseAPI imports and builtins
"""

import json
import sys
import os

def main():
    results = {
        "import_test": {
            "timestamp": "2025-08-16T01:57:00Z"
        }
    }
    
    # Test 1: Check if functions are available as builtins
    builtin_functions = [
        'load_context', 'return_context', 'get_client_context', 
        'get_client_integration_config', 'get_integration_config'
    ]
    
    results["builtin_availability"] = {}
    for func_name in builtin_functions:
        try:
            func = globals().get(func_name) or getattr(__builtins__, func_name, None)
            results["builtin_availability"][func_name] = func is not None
        except:
            results["builtin_availability"][func_name] = False
    
    # Test 2: Try direct import
    try:
        sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'server'))
        import SoarBaseAPI
        results["direct_import"] = {
            "success": True,
            "has_get_client_context": hasattr(SoarBaseAPI, 'get_client_context'),
            "has_get_client_integration_config": hasattr(SoarBaseAPI, 'get_client_integration_config'),
            "has_load_context": hasattr(SoarBaseAPI, 'load_context')
        }
        
        # Test 3: Try using the functions
        if hasattr(SoarBaseAPI, 'load_context'):
            try:
                context = SoarBaseAPI.load_context()
                results["context_loading"] = {
                    "success": True,
                    "context": context,
                    "context_type": type(context).__name__
                }
            except Exception as e:
                results["context_loading"] = {
                    "success": False,
                    "error": str(e)
                }
        
        if hasattr(SoarBaseAPI, 'get_client_context'):
            try:
                client_id = SoarBaseAPI.get_client_context()
                results["client_context"] = {
                    "success": True,
                    "client_id": client_id
                }
            except Exception as e:
                results["client_context"] = {
                    "success": False,
                    "error": str(e)
                }
                
    except ImportError as e:
        results["direct_import"] = {
            "success": False,
            "error": str(e)
        }
    
    # Test 4: Check environment
    results["environment"] = {
        "has_secauto_context": bool(os.environ.get('SECAUTO_CONTEXT')),
        "secauto_context": os.environ.get('SECAUTO_CONTEXT', 'not_set')[:100]
    }
    
    print(json.dumps(results, indent=2))

if __name__ == "__main__":
    main()