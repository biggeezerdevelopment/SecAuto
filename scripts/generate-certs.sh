#!/bin/bash

# Generate self-signed certificates for SecAuto HTTPS development
# This script creates certificates suitable for development and testing

set -e

# Configuration
CERT_DIR="certs"
CERT_FILE="server.crt"
KEY_FILE="server.key"
DAYS=365
COUNTRY="US"
STATE="CA"
CITY="San Francisco"
ORG="SecAuto Development"
OU="IT Department"
CN="localhost"

# Additional Subject Alternative Names (SANs)
SANS="DNS:localhost,DNS:127.0.0.1,IP:127.0.0.1,IP:::1"

echo "🔐 Generating self-signed certificates for SecAuto HTTPS..."

# Create certificate directory
mkdir -p "$CERT_DIR"

# Generate private key
echo "📝 Generating private key..."
openssl genrsa -out "$CERT_DIR/$KEY_FILE" 2048

# Generate certificate signing request
echo "📋 Generating certificate signing request..."
openssl req -new -key "$CERT_DIR/$KEY_FILE" -out "$CERT_DIR/server.csr" \
    -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$OU/CN=$CN"

# Create extensions file for SAN
cat > "$CERT_DIR/server.ext" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = 127.0.0.1
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# Generate self-signed certificate
echo "🔒 Generating self-signed certificate..."
openssl x509 -req -in "$CERT_DIR/server.csr" -signkey "$CERT_DIR/$KEY_FILE" \
    -out "$CERT_DIR/$CERT_FILE" -days $DAYS \
    -extensions v3_req -extfile "$CERT_DIR/server.ext"

# Set appropriate permissions
chmod 600 "$CERT_DIR/$KEY_FILE"
chmod 644 "$CERT_DIR/$CERT_FILE"

# Clean up temporary files
rm "$CERT_DIR/server.csr" "$CERT_DIR/server.ext"

echo "✅ Certificates generated successfully!"
echo "📁 Certificate location: $CERT_DIR/$CERT_FILE"
echo "🔑 Private key location: $CERT_DIR/$KEY_FILE"
echo "⏰ Valid for: $DAYS days"
echo ""
echo "🔧 To enable HTTPS in SecAuto:"
echo "1. Edit config.yaml and set:"
echo "   security:"
echo "     tls:"
echo "       enabled: true"
echo "       cert_file: \"$CERT_DIR/$CERT_FILE\""
echo "       key_file: \"$CERT_DIR/$KEY_FILE\""
echo ""
echo "⚠️  Note: This is a self-signed certificate for development only."
echo "   Browsers will show security warnings. For production, use certificates"
echo "   from a trusted Certificate Authority or enable autocert."
echo ""
echo "🌐 Test your HTTPS setup:"
echo "   curl -k https://localhost:9443/health"