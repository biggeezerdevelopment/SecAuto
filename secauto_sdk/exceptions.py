"""
SecAuto SDK Exception Classes

Custom exception classes for handling various error conditions
when interacting with the SecAuto API.
"""

class SecAutoError(Exception):
    """Base exception class for SecAuto SDK errors."""
    
    def __init__(self, message, details=None):
        super().__init__(message)
        self.message = message
        self.details = details or {}
    
    def __str__(self):
        if self.details:
            return f"{self.message}: {self.details}"
        return self.message


class SecAutoAPIError(SecAutoError):
    """Exception raised for API-related errors."""
    
    def __init__(self, message, status_code=None, response_body=None, details=None):
        super().__init__(message, details)
        self.status_code = status_code
        self.response_body = response_body
    
    def __str__(self):
        base_msg = super().__str__()
        if self.status_code:
            return f"HTTP {self.status_code}: {base_msg}"
        return base_msg


class SecAutoConnectionError(SecAutoError):
    """Exception raised for connection-related errors."""
    
    def __init__(self, message, original_exception=None):
        super().__init__(message)
        self.original_exception = original_exception
    
    def __str__(self):
        base_msg = super().__str__()
        if self.original_exception:
            return f"{base_msg} (Original: {self.original_exception})"
        return base_msg


class SecAutoAuthenticationError(SecAutoAPIError):
    """Exception raised for authentication-related errors."""
    pass


class SecAutoValidationError(SecAutoError):
    """Exception raised for validation errors."""
    
    def __init__(self, message, validation_errors=None):
        super().__init__(message)
        self.validation_errors = validation_errors or []
    
    def __str__(self):
        base_msg = super().__str__()
        if self.validation_errors:
            error_details = "; ".join([f"{err.get('field', 'field')}: {err.get('message', 'error')}" 
                                     for err in self.validation_errors])
            return f"{base_msg} - Validation errors: {error_details}"
        return base_msg


class SecAutoNotFoundError(SecAutoAPIError):
    """Exception raised when a resource is not found."""
    pass


class SecAutoTimeoutError(SecAutoConnectionError):
    """Exception raised when a request times out."""
    pass
