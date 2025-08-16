package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/acme/autocert"

	"SoarAuto/pkg/config"
	"SoarAuto/pkg/types"
)

// TLSManager handles TLS configuration and certificate management
type TLSManager struct {
	config     *config.TLSConfig
	logger     types.Logger
	autocertManager *autocert.Manager
}

// NewTLSManager creates a new TLS manager
func NewTLSManager(cfg *config.TLSConfig, logger types.Logger) *TLSManager {
	manager := &TLSManager{
		config: cfg,
		logger: logger,
	}

	// Initialize autocert manager if enabled
	if cfg.AutoCert.Enabled {
		manager.setupAutoCert()
	}

	return manager
}

// setupAutoCert configures automatic certificate management
func (tm *TLSManager) setupAutoCert() {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(tm.config.AutoCert.CacheDir, 0700); err != nil {
		tm.logger.Error("Failed to create autocert cache directory", map[string]interface{}{
			"component": "tls_manager",
			"error":     err.Error(),
			"cache_dir": tm.config.AutoCert.CacheDir,
		})
		return
	}

	tm.autocertManager = &autocert.Manager{
		Cache:      autocert.DirCache(tm.config.AutoCert.CacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(tm.config.AutoCert.Domains...),
		Email:      tm.config.AutoCert.Email,
	}

	tm.logger.Info("Autocert manager initialized", map[string]interface{}{
		"component": "tls_manager",
		"domains":   tm.config.AutoCert.Domains,
		"cache_dir": tm.config.AutoCert.CacheDir,
	})
}

// GetTLSConfig returns a configured TLS config
func (tm *TLSManager) GetTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{}

	// Set TLS version constraints
	if tm.config.MinVersion != "" {
		minVersion, err := tm.parseTLSVersion(tm.config.MinVersion)
		if err != nil {
			return nil, fmt.Errorf("invalid min TLS version: %v", err)
		}
		tlsConfig.MinVersion = minVersion
	}

	if tm.config.MaxVersion != "" {
		maxVersion, err := tm.parseTLSVersion(tm.config.MaxVersion)
		if err != nil {
			return nil, fmt.Errorf("invalid max TLS version: %v", err)
		}
		tlsConfig.MaxVersion = maxVersion
	}

	// Set cipher suites if specified
	if len(tm.config.CipherSuites) > 0 {
		cipherSuites, err := tm.parseCipherSuites(tm.config.CipherSuites)
		if err != nil {
			return nil, fmt.Errorf("invalid cipher suites: %v", err)
		}
		tlsConfig.CipherSuites = cipherSuites
	}

	// Configure client authentication if enabled
	if tm.config.ClientAuth.Enabled {
		if err := tm.setupClientAuth(tlsConfig); err != nil {
			return nil, fmt.Errorf("failed to setup client auth: %v", err)
		}
	}

	// Use autocert if enabled
	if tm.config.AutoCert.Enabled && tm.autocertManager != nil {
		tlsConfig.GetCertificate = tm.autocertManager.GetCertificate
		tlsConfig.NextProtos = []string{"h2", "http/1.1"}
	} else if tm.config.CertFile != "" && tm.config.KeyFile != "" {
		// Load certificate and key files
		cert, err := tls.LoadX509KeyPair(tm.config.CertFile, tm.config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	} else {
		return nil, fmt.Errorf("no certificate configuration provided")
	}

	tm.logger.Info("TLS configuration created", map[string]interface{}{
		"component":     "tls_manager",
		"min_version":   tm.config.MinVersion,
		"max_version":   tm.config.MaxVersion,
		"client_auth":   tm.config.ClientAuth.Enabled,
		"autocert":      tm.config.AutoCert.Enabled,
	})

	return tlsConfig, nil
}

// setupClientAuth configures client certificate authentication
func (tm *TLSManager) setupClientAuth(tlsConfig *tls.Config) error {
	if tm.config.ClientAuth.CAFile == "" {
		return fmt.Errorf("CA file required for client authentication")
	}

	// Load CA certificate
	caCert, err := ioutil.ReadFile(tm.config.ClientAuth.CAFile)
	if err != nil {
		return fmt.Errorf("failed to read CA file: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig.ClientCAs = caCertPool

	if tm.config.ClientAuth.RequireCert {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	} else {
		tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
	}

	tm.logger.Info("Client authentication configured", map[string]interface{}{
		"component":    "tls_manager",
		"ca_file":      tm.config.ClientAuth.CAFile,
		"require_cert": tm.config.ClientAuth.RequireCert,
	})

	return nil
}

// parseTLSVersion converts string version to TLS constant
func (tm *TLSManager) parseTLSVersion(version string) (uint16, error) {
	switch strings.ToLower(version) {
	case "1.0":
		return tls.VersionTLS10, nil
	case "1.1":
		return tls.VersionTLS11, nil
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unsupported TLS version: %s", version)
	}
}

// parseCipherSuites converts string cipher suite names to constants
func (tm *TLSManager) parseCipherSuites(suites []string) ([]uint16, error) {
	var cipherSuites []uint16
	
	cipherMap := map[string]uint16{
		"TLS_RSA_WITH_RC4_128_SHA":                      tls.TLS_RSA_WITH_RC4_128_SHA,
		"TLS_RSA_WITH_3DES_EDE_CBC_SHA":                 tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_RSA_WITH_AES_128_CBC_SHA":                  tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":                  tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_CBC_SHA256":               tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":               tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":               tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":              tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_RC4_128_SHA":                tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":           tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":       tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	}

	for _, suite := range suites {
		if cipherSuite, exists := cipherMap[suite]; exists {
			cipherSuites = append(cipherSuites, cipherSuite)
		} else {
			return nil, fmt.Errorf("unsupported cipher suite: %s", suite)
		}
	}

	return cipherSuites, nil
}

// CreateSelfSignedCert creates a self-signed certificate for development
func (tm *TLSManager) CreateSelfSignedCert(certPath, keyPath string, hosts []string) error {
	// This would contain the logic to generate self-signed certificates
	// For brevity, this is a placeholder - you'd implement certificate generation here
	tm.logger.Info("Self-signed certificate creation requested", map[string]interface{}{
		"component": "tls_manager",
		"cert_path": certPath,
		"key_path":  keyPath,
		"hosts":     hosts,
	})
	
	return fmt.Errorf("self-signed certificate generation not implemented - use openssl or similar tool")
}

// GetHTTPSRedirectHandler returns a handler that redirects HTTP to HTTPS
func (tm *TLSManager) GetHTTPSRedirectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		
		// Remove port from host if present
		if colonIndex := strings.LastIndex(host, ":"); colonIndex != -1 {
			host = host[:colonIndex]
		}
		
		// Add HTTPS port if not default
		httpsURL := "https://" + host
		if tm.config.Port != 443 {
			httpsURL += ":" + strconv.Itoa(tm.config.Port)
		}
		httpsURL += r.RequestURI
		
		tm.logger.Debug("Redirecting HTTP to HTTPS", map[string]interface{}{
			"component":   "tls_manager",
			"original_url": r.URL.String(),
			"redirect_url": httpsURL,
		})
		
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	}
}

// GetAutocertHandler returns the autocert HTTP handler for ACME challenges
func (tm *TLSManager) GetAutocertHandler() http.Handler {
	if tm.autocertManager != nil {
		return tm.autocertManager.HTTPHandler(nil)
	}
	return nil
}

// ValidateCertificates validates that certificate files exist and are valid
func (tm *TLSManager) ValidateCertificates() error {
	if tm.config.AutoCert.Enabled {
		// For autocert, just validate the cache directory
		if err := os.MkdirAll(tm.config.AutoCert.CacheDir, 0700); err != nil {
			return fmt.Errorf("failed to create autocert cache directory: %v", err)
		}
		return nil
	}

	// Validate certificate files
	if tm.config.CertFile == "" || tm.config.KeyFile == "" {
		return fmt.Errorf("certificate and key files must be specified")
	}

	// Check if files exist
	if _, err := os.Stat(tm.config.CertFile); os.IsNotExist(err) {
		return fmt.Errorf("certificate file not found: %s", tm.config.CertFile)
	}

	if _, err := os.Stat(tm.config.KeyFile); os.IsNotExist(err) {
		return fmt.Errorf("key file not found: %s", tm.config.KeyFile)
	}

	// Try to load the certificate to validate it
	_, err := tls.LoadX509KeyPair(tm.config.CertFile, tm.config.KeyFile)
	if err != nil {
		return fmt.Errorf("invalid certificate or key: %v", err)
	}

	tm.logger.Info("Certificate validation successful", map[string]interface{}{
		"component": "tls_manager",
		"cert_file": tm.config.CertFile,
		"key_file":  tm.config.KeyFile,
	})

	return nil
}

// GetCertificateInfo returns information about the loaded certificate
func (tm *TLSManager) GetCertificateInfo() (map[string]interface{}, error) {
	if tm.config.AutoCert.Enabled {
		return map[string]interface{}{
			"type":     "autocert",
			"domains":  tm.config.AutoCert.Domains,
			"cache_dir": tm.config.AutoCert.CacheDir,
		}, nil
	}

	if tm.config.CertFile == "" || tm.config.KeyFile == "" {
		return nil, fmt.Errorf("no certificate configuration")
	}

	cert, err := tls.LoadX509KeyPair(tm.config.CertFile, tm.config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %v", err)
	}

	if len(cert.Certificate) == 0 {
		return nil, fmt.Errorf("no certificate data")
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	return map[string]interface{}{
		"type":        "file",
		"subject":     x509Cert.Subject.String(),
		"issuer":      x509Cert.Issuer.String(),
		"not_before":  x509Cert.NotBefore,
		"not_after":   x509Cert.NotAfter,
		"dns_names":   x509Cert.DNSNames,
		"ip_addresses": x509Cert.IPAddresses,
		"serial_number": x509Cert.SerialNumber.String(),
	}, nil
}