# CORS Implementation in SecAuto

## Overview

Cross-Origin Resource Sharing (CORS) has been implemented in SecAuto to allow web applications to make requests to the SecAuto API from different origins. The implementation is fully configurable through the `config.yaml` file.

## Configuration

CORS settings are configured in the `security.cors` section of `config.yaml`:

```yaml
security:
  cors:
    enabled: true
    # Allow all origins (use specific domains in production)
    allowed_origins: ["*"]
    # Allowed HTTP methods
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    # Allowed headers
    allowed_headers: ["Content-Type", "Authorization", "X-API-Key", "Accept", "Origin"]
    # Cache preflight requests for 24 hours
    max_age: 86400
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `true` | Enable/disable CORS functionality |
| `allowed_origins` | array | `["*"]` | List of allowed origins (use `"*"` for all origins) |
| `allowed_methods` | array | `["GET", "POST", "PUT", "DELETE", "OPTIONS"]` | Allowed HTTP methods |
| `allowed_headers` | array | `["Content-Type", "Authorization", "X-API-Key", "Accept", "Origin"]` | Allowed request headers |
| `max_age` | integer | `86400` | Cache duration for preflight requests in seconds |

## Implementation Details

### Files Modified/Created

1. **`cors_middleware.go`** - New file containing CORS middleware implementation
2. **`main.go`** - Updated to include CORS middleware in all routes
3. **`config.yaml`** - Enhanced with detailed CORS configuration
4. **`test_cors.html`** - Test file to verify CORS functionality

### Middleware Architecture

The CORS implementation uses a middleware pattern that:

1. **Checks CORS Configuration**: Only applies CORS headers if enabled in config
2. **Validates Origins**: Compares the request origin against allowed origins
3. **Sets CORS Headers**: Adds appropriate CORS headers to responses
4. **Handles Preflight**: Responds to OPTIONS requests with proper CORS headers
5. **Integrates with Existing Middleware**: Works alongside logging, validation, rate limiting, and authentication

### CORS Headers Set

- `Access-Control-Allow-Origin`: The allowed origin (or `*` for all)
- `Access-Control-Allow-Methods`: Comma-separated list of allowed HTTP methods
- `Access-Control-Allow-Headers`: Comma-separated list of allowed headers
- `Access-Control-Max-Age`: Cache duration for preflight requests
- `Access-Control-Allow-Credentials`: Set to `true` for preflight requests

## Security Considerations

### Production Recommendations

1. **Restrict Origins**: Replace `["*"]` with specific domains:
   ```yaml
   allowed_origins: ["https://yourdomain.com", "https://app.yourdomain.com"]
   ```

2. **Limit Methods**: Only allow necessary HTTP methods:
   ```yaml
   allowed_methods: ["GET", "POST"]  # Only if you don't need PUT/DELETE
   ```

3. **Restrict Headers**: Only allow required headers:
   ```yaml
   allowed_headers: ["Content-Type", "X-API-Key"]
   ```

4. **Use HTTPS**: Always use HTTPS in production to protect CORS headers

### Security Best Practices

- **Never use `"*"` for `Access-Control-Allow-Origin` with credentials**
- **Validate origins server-side** (implemented in the middleware)
- **Use specific domains** instead of wildcards in production
- **Regularly review and update** allowed origins, methods, and headers

## Testing CORS

### Using the Test File

1. Start your SecAuto server:
   ```bash
   cd SoarAuto
   go run .
   ```

2. Open `test_cors.html` in a web browser

3. Click the test buttons to verify:
   - Health endpoint (no auth required)
   - Authenticated requests
   - Preflight OPTIONS requests
   - Swagger UI accessibility

### Manual Testing

Test CORS with curl:

```bash
# Test preflight request
curl -X OPTIONS \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET" \
  -H "Access-Control-Request-Headers: Content-Type, X-API-Key" \
  http://localhost:8000/health

# Test actual request
curl -X GET \
  -H "Origin: http://localhost:3000" \
  -H "Content-Type: application/json" \
  http://localhost:8000/health
```

### Browser Developer Tools

1. Open browser developer tools (F12)
2. Go to Network tab
3. Make a request to SecAuto API
4. Check for CORS headers in the response

## Troubleshooting

### Common Issues

1. **CORS Error in Browser Console**
   - Check if CORS is enabled in config
   - Verify allowed origins include your domain
   - Ensure allowed methods include the HTTP method you're using

2. **Preflight Fails**
   - Check that OPTIONS is in allowed_methods
   - Verify allowed_headers includes your custom headers
   - Ensure max_age is set appropriately

3. **Swagger UI CORS Issues**
   - Swagger UI is now wrapped with CORS middleware
   - Check browser console for specific error messages

### Debug Steps

1. **Check Configuration**:
   ```bash
   # Verify CORS is enabled
   grep -A 10 "cors:" config.yaml
   ```

2. **Test with curl**:
   ```bash
   # Test basic CORS headers
   curl -I -H "Origin: http://localhost:3000" http://localhost:8000/health
   ```

3. **Check Server Logs**:
   - Look for CORS-related log messages
   - Verify middleware is being applied

## Integration with Other Middleware

The CORS middleware is applied **first** in the middleware chain:

```
CORS → Logging → Validation → Rate Limiting → Authentication → Handler
```

This ensures that:
- CORS headers are always set, even for failed requests
- Preflight requests are handled before authentication
- CORS works with all existing middleware

## API Endpoints with CORS

All API endpoints now support CORS:

- `/health` - Health check (no auth required)
- `/playbook` - Playbook execution
- `/jobs` - Job management
- `/plugins` - Plugin management
- `/integrations` - Integration management
- `/docs` - Swagger UI documentation
- `/api-docs` - OpenAPI specification

## Performance Impact

- **Minimal overhead**: CORS checks are fast string comparisons
- **Caching**: Preflight responses are cached based on `max_age`
- **Conditional execution**: CORS is only applied when enabled

## Future Enhancements

Potential improvements for future versions:

1. **Dynamic CORS**: Allow CORS settings to be updated without restart
2. **Origin Validation**: Add regex patterns for origin validation
3. **Credential Support**: Add support for `Access-Control-Allow-Credentials`
4. **Expose Headers**: Add `Access-Control-Expose-Headers` support
5. **Metrics**: Add CORS-related metrics and monitoring

## References

- [MDN CORS Documentation](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [CORS Specification](https://fetch.spec.whatwg.org/#cors-protocol)
- [SecAuto API Documentation](http://localhost:8000/docs) 