# HTTPS Setup Guide for SecAuto

This guide explains how to configure HTTPS/TLS support in SecAuto for secure communications.

## 🔒 Overview

SecAuto supports multiple HTTPS configurations:

- **Self-signed certificates** (development)
- **Custom certificates** (production)
- **Automatic certificates** (Let's Encrypt via autocert)
- **Client certificate authentication** (mutual TLS)

## 🚀 Quick Start

### 1. Generate Development Certificates

For development and testing, generate self-signed certificates:

**Linux/macOS:**
```bash
cd SoarAuto
./scripts/generate-certs.sh
```

**Windows (PowerShell):**
```powershell
cd SoarAuto
.\scripts\generate-certs.ps1
```

### 2. Enable HTTPS in Configuration

Edit `config.yaml`:

```yaml
security:
  tls:
    enabled: true                    # Enable HTTPS
    port: 9443                      # HTTPS port
    cert_file: "certs/server.crt"   # Certificate file
    key_file: "certs/server.key"    # Private key file
    auto_redirect: true             # Redirect HTTP to HTTPS
    min_version: "1.2"              # Minimum TLS version
    max_version: "1.3"              # Maximum TLS version
```

### 3. Start the Server

```bash
go run main.go
```

The server will now be available at:
- **HTTPS**: `https://localhost:9443`
- **HTTP redirect**: `http://localhost:9090` (redirects to HTTPS)

## 📋 Configuration Options

### Basic TLS Configuration

```yaml
security:
  tls:
    enabled: true                    # Enable/disable HTTPS
    port: 9443                      # HTTPS port (default: 443 for production)
    cert_file: "certs/server.crt"   # Path to certificate file
    key_file: "certs/server.key"    # Path to private key file
    auto_redirect: true             # Redirect HTTP to HTTPS
    min_version: "1.2"              # Minimum TLS version (1.0, 1.1, 1.2, 1.3)
    max_version: "1.3"              # Maximum TLS version
    cipher_suites: []               # Custom cipher suites (empty = use Go defaults)
```

### Automatic Certificates (Let's Encrypt)

For production with automatic certificate management:

```yaml
security:
  tls:
    enabled: true
    port: 443                       # Standard HTTPS port
    auto_cert:
      enabled: true                 # Enable automatic certificates
      domains: ["yourdomain.com"]   # Domains for certificates
      cache_dir: "certs/autocert"   # Certificate cache directory
      email: "admin@yourdomain.com" # Email for Let's Encrypt registration
```

### Client Certificate Authentication (Mutual TLS)

For enhanced security with client certificates:

```yaml
security:
  tls:
    enabled: true
    port: 9443
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"
    client_auth:
      enabled: true                 # Enable client certificate verification
      ca_file: "certs/ca.crt"      # CA certificate for client verification
      require_cert: true           # Require client certificates (false = optional)
```

## 🛠️ Certificate Management

### Self-Signed Certificates (Development)

**Advantages:**
- Quick setup for development
- No external dependencies
- Works offline

**Disadvantages:**
- Browser security warnings
- Not trusted by clients
- Manual certificate management

**Generate with OpenSSL:**
```bash
# Generate private key
openssl genrsa -out certs/server.key 2048

# Generate certificate
openssl req -new -x509 -key certs/server.key -out certs/server.crt -days 365 \
  -subj "/C=US/ST=CA/L=San Francisco/O=SecAuto/CN=localhost"
```

### Custom Certificates (Production)

**For production environments:**

1. **Obtain certificates** from a trusted Certificate Authority (CA)
2. **Place certificate files** in the `certs/` directory
3. **Update configuration** with correct file paths
4. **Set appropriate permissions**:
   ```bash
   chmod 600 certs/server.key  # Private key - restricted access
   chmod 644 certs/server.crt  # Certificate - readable
   ```

### Automatic Certificates (Let's Encrypt)

**Requirements:**
- Domain name pointing to your server
- Port 80 accessible for ACME challenges
- Port 443 accessible for HTTPS traffic

**Configuration:**
```yaml
security:
  tls:
    enabled: true
    port: 443
    auto_cert:
      enabled: true
      domains: ["api.yourdomain.com", "secauto.yourdomain.com"]
      cache_dir: "certs/autocert"
      email: "admin@yourdomain.com"
```

**Features:**
- Automatic certificate issuance
- Automatic renewal (90-day certificates)
- Multiple domain support
- ACME challenge handling

## 🔧 Advanced Configuration

### Custom Cipher Suites

For specific security requirements:

```yaml
security:
  tls:
    enabled: true
    cipher_suites:
      - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
      - "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256"
      - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
```

### TLS Version Constraints

```yaml
security:
  tls:
    enabled: true
    min_version: "1.2"  # Disable older, less secure versions
    max_version: "1.3"  # Use latest TLS version
```

### HTTP to HTTPS Redirection

```yaml
security:
  tls:
    enabled: true
    auto_redirect: true  # Automatically redirect HTTP to HTTPS
```

When enabled:
- HTTP requests to port 9090 redirect to HTTPS port 9443
- Maintains original request path and query parameters
- Uses 301 (Moved Permanently) status code

## 🧪 Testing HTTPS Setup

### Basic Connectivity Test

```bash
# Test HTTPS endpoint (ignore certificate warnings for self-signed)
curl -k https://localhost:9443/health

# Test HTTP redirect (if enabled)
curl -v http://localhost:9090/health
```

### Certificate Information

```bash
# View certificate details
openssl x509 -in certs/server.crt -text -noout

# Test TLS connection
openssl s_client -connect localhost:9443 -servername localhost
```

### Browser Testing

1. **Navigate to** `https://localhost:9443/health`
2. **Accept security warning** (for self-signed certificates)
3. **Verify HTTPS connection** (lock icon in address bar)

## 🚨 Security Considerations

### Development vs Production

**Development:**
- Self-signed certificates are acceptable
- Use non-standard ports (9443 instead of 443)
- Certificate warnings are expected

**Production:**
- Use certificates from trusted CAs
- Use standard ports (443 for HTTPS, 80 for HTTP)
- Implement proper certificate management
- Enable security headers
- Consider client certificate authentication

### Best Practices

1. **Use strong TLS versions** (1.2 minimum, 1.3 preferred)
2. **Secure private keys** with appropriate file permissions
3. **Regular certificate renewal** (automated with Let's Encrypt)
4. **Monitor certificate expiration**
5. **Use HSTS headers** for enhanced security
6. **Implement certificate pinning** for critical applications

### File Permissions

```bash
# Secure private key
chmod 600 certs/server.key
chown secauto:secauto certs/server.key

# Public certificate
chmod 644 certs/server.crt
chown secauto:secauto certs/server.crt

# Certificate directory
chmod 755 certs/
```

## 🔍 Troubleshooting

### Common Issues

**Certificate not found:**
```
Error: certificate file not found: certs/server.crt
```
- Verify file paths in configuration
- Check file permissions
- Ensure certificates are generated

**Invalid certificate:**
```
Error: invalid certificate or key: tls: private key does not match public key
```
- Verify certificate and key pair match
- Regenerate certificates if necessary

**Port already in use:**
```
Error: listen tcp :9443: bind: address already in use
```
- Check for other services using the port
- Use different port in configuration
- Stop conflicting services

**Browser security warnings:**
```
NET::ERR_CERT_AUTHORITY_INVALID
```
- Expected for self-signed certificates
- Add certificate to browser trust store for development
- Use proper CA certificates for production

### Debug Logging

Enable debug logging for TLS issues:

```yaml
logging:
  level: "DEBUG"
  component_levels:
    tls_manager: "DEBUG"
```

### Certificate Validation

```bash
# Validate certificate and key pair
openssl x509 -noout -modulus -in certs/server.crt | openssl md5
openssl rsa -noout -modulus -in certs/server.key | openssl md5
# The MD5 hashes should match
```

## 📚 Additional Resources

- [Go TLS Documentation](https://golang.org/pkg/crypto/tls/)
- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [OpenSSL Documentation](https://www.openssl.org/docs/)
- [Mozilla TLS Configuration](https://ssl-config.mozilla.org/)

## 🆘 Support

For HTTPS-related issues:

1. **Check logs** for TLS-specific error messages
2. **Verify configuration** syntax and file paths
3. **Test certificates** with OpenSSL tools
4. **Review firewall** and network settings
5. **Consult documentation** for specific error codes

---

**Security Note**: Always use proper certificates from trusted Certificate Authorities in production environments. Self-signed certificates should only be used for development and testing purposes.