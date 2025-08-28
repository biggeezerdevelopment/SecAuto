# Generate self-signed certificates for SecAuto HTTPS development
# This script creates certificates suitable for development and testing on Windows

param(
    [string]$CertDir = "certs",
    [string]$CertFile = "server.crt",
    [string]$KeyFile = "server.key",
    [int]$Days = 365,
    [string]$CommonName = "localhost"
)

Write-Host "🔐 Generating self-signed certificates for SecAuto HTTPS..." -ForegroundColor Green

# Create certificate directory
if (!(Test-Path $CertDir)) {
    New-Item -ItemType Directory -Path $CertDir -Force | Out-Null
}

# Check if OpenSSL is available
$opensslPath = Get-Command openssl -ErrorAction SilentlyContinue
if ($opensslPath) {
    Write-Host "📝 Using OpenSSL to generate certificates..." -ForegroundColor Yellow
    
    # Generate private key
    Write-Host "🔑 Generating private key..."
    & openssl genrsa -out "$CertDir/$KeyFile" 2048
    
    # Generate certificate signing request
    Write-Host "📋 Generating certificate signing request..."
    & openssl req -new -key "$CertDir/$KeyFile" -out "$CertDir/server.csr" -subj "/C=US/ST=CA/L=San Francisco/O=SecAuto Development/OU=IT Department/CN=$CommonName"
    
    # Create extensions file
    @"
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = 127.0.0.1
IP.1 = 127.0.0.1
IP.2 = ::1
"@ | Out-File -FilePath "$CertDir/server.ext" -Encoding ASCII
    
    # Generate self-signed certificate
    Write-Host "🔒 Generating self-signed certificate..."
    & openssl x509 -req -in "$CertDir/server.csr" -signkey "$CertDir/$KeyFile" -out "$CertDir/$CertFile" -days $Days -extensions v3_req -extfile "$CertDir/server.ext"
    
    # Clean up temporary files
    Remove-Item "$CertDir/server.csr", "$CertDir/server.ext" -ErrorAction SilentlyContinue
    
} else {
    Write-Host "📝 Using PowerShell to generate certificates..." -ForegroundColor Yellow
    
    # Create self-signed certificate using PowerShell (Windows 10/Server 2016+)
    $cert = New-SelfSignedCertificate -DnsName $CommonName, "127.0.0.1" -CertStoreLocation "cert:\LocalMachine\My" -NotAfter (Get-Date).AddDays($Days)
    
    # Export certificate
    $certPath = Join-Path $CertDir $CertFile
    $keyPath = Join-Path $CertDir $KeyFile
    
    # Export certificate to file
    Export-Certificate -Cert $cert -FilePath $certPath -Type CERT | Out-Null
    
    # Export private key (requires password for PKCS#12, then convert)
    $password = ConvertTo-SecureString -String "temp" -Force -AsPlainText
    $pfxPath = Join-Path $CertDir "temp.pfx"
    Export-PfxCertificate -Cert $cert -FilePath $pfxPath -Password $password | Out-Null
    
    # Convert PFX to PEM format (if OpenSSL is available)
    if (Get-Command openssl -ErrorAction SilentlyContinue) {
        & openssl pkcs12 -in $pfxPath -out $keyPath -nodes -nocerts -passin pass:temp
        Remove-Item $pfxPath -ErrorAction SilentlyContinue
    } else {
        Write-Warning "⚠️  OpenSSL not found. Private key exported as PFX format: $pfxPath"
        Write-Warning "   Install OpenSSL to convert to PEM format, or use the PFX file directly."
    }
    
    # Remove certificate from store
    Remove-Item "cert:\LocalMachine\My\$($cert.Thumbprint)" -ErrorAction SilentlyContinue
}

Write-Host "✅ Certificates generated successfully!" -ForegroundColor Green
Write-Host "📁 Certificate location: $CertDir/$CertFile" -ForegroundColor Cyan
Write-Host "🔑 Private key location: $CertDir/$KeyFile" -ForegroundColor Cyan
Write-Host "⏰ Valid for: $Days days" -ForegroundColor Cyan
Write-Host ""
Write-Host "🔧 To enable HTTPS in SecAuto:" -ForegroundColor Yellow
Write-Host "1. Edit config.yaml and set:" -ForegroundColor White
Write-Host "   security:" -ForegroundColor Gray
Write-Host "     tls:" -ForegroundColor Gray
Write-Host "       enabled: true" -ForegroundColor Gray
Write-Host "       cert_file: `"$CertDir/$CertFile`"" -ForegroundColor Gray
Write-Host "       key_file: `"$CertDir/$KeyFile`"" -ForegroundColor Gray
Write-Host ""
Write-Host "⚠️  Note: This is a self-signed certificate for development only." -ForegroundColor Red
Write-Host "   Browsers will show security warnings. For production, use certificates" -ForegroundColor Red
Write-Host "   from a trusted Certificate Authority or enable autocert." -ForegroundColor Red
Write-Host ""
Write-Host "🌐 Test your HTTPS setup:" -ForegroundColor Yellow
Write-Host "   curl -k https://localhost:9443/health" -ForegroundColor White