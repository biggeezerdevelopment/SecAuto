#!/usr/bin/env python3
"""
Simplified sitecustomize for UV-based integrations
UV handles isolation natively, so this is much simpler
"""

import os
import sys
from pathlib import Path


def setup_secauto_base_api():
    """Make SoarBaseAPI functions available as builtins for integrations"""
    
    # Only run this in integration contexts
    integration_name = os.environ.get('INTEGRATION_NAME')
    if not integration_name:
        return
    
    try:
        # Import SoarBaseAPI functions
        sys.path.insert(0, str(Path(__file__).parent))
        
        from SoarBaseAPI import load_context, return_context
        from secauto_base import use_integration
        
        # Make them available as builtins for backward compatibility
        import builtins
        builtins.base_context = load_context  # Legacy alias
        builtins.load_context = load_context
        builtins.return_context = return_context
        builtins.use_integration = use_integration
        
        # Add integration-specific helper functions
        builtins.get_secauto_config = lambda: {
            'integration_name': integration_name,
            'secauto_root': os.environ.get('SECAUTO_ROOT', ''),
            'function': os.environ.get('INTEGRATION_FUNCTION', '')
        }
        
    except ImportError as e:
        # Fail silently - not all environments need this
        pass
    except Exception as e:
        # Log error but don't break Python startup
        try:
            import logging
            logging.warning(f"SecAuto sitecustomize setup failed: {e}")
        except:
            pass


def setup_integration_logging():
    """Setup simple logging for integrations"""
    integration_name = os.environ.get('INTEGRATION_NAME')
    if not integration_name:
        return
    
    try:
        import logging
        
        # Create simple console logger
        logging.basicConfig(
            level=logging.INFO,
            format=f'[{integration_name}] %(levelname)s: %(message)s',
            handlers=[logging.StreamHandler()]
        )
        
        # Make logger available as builtin
        import builtins
        builtins.logger = logging.getLogger(integration_name)
        
    except Exception:
        # Fail silently
        pass


# Auto-setup when imported
try:
    setup_secauto_base_api()
    setup_integration_logging()
except Exception:
    # Never break Python startup
    pass