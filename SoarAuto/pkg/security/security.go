package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateUploadName validates that a file name is safe for upload
// and doesn't contain path traversal sequences
func ValidateUploadName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	// Check for path traversal sequences
	if strings.Contains(name, "..") {
		return fmt.Errorf("name cannot contain '..' sequences")
	}

	// Check for absolute paths
	if filepath.IsAbs(name) {
		return fmt.Errorf("name cannot be an absolute path")
	}

	// Check for directory separators
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("name cannot contain directory separators")
	}

	// Check for control characters and other dangerous characters
	for _, char := range name {
		if char < 32 || char == 127 {
			return fmt.Errorf("name cannot contain control characters")
		}
	}

	// Check for reserved names on Windows
	reservedNames := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	upperName := strings.ToUpper(name)
	for _, reserved := range reservedNames {
		if upperName == reserved || strings.HasPrefix(upperName, reserved+".") {
			return fmt.Errorf("name cannot be a reserved system name")
		}
	}

	return nil
}

// SanitizeFilename sanitizes a filename by removing or replacing dangerous characters
func SanitizeFilename(filename string) string {
	// Remove path components
	filename = filepath.Base(filename)
	
	// Replace dangerous characters with underscores
	dangerous := []string{"..", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range dangerous {
		filename = strings.ReplaceAll(filename, char, "_")
	}
	
	// Remove control characters
	var result strings.Builder
	for _, r := range filename {
		if r >= 32 && r != 127 {
			result.WriteRune(r)
		}
	}
	
	filename = result.String()
	
	// Ensure filename is not empty and not just dots/spaces
	filename = strings.TrimSpace(filename)
	if filename == "" || strings.Trim(filename, ".") == "" {
		filename = "unnamed_file"
	}
	
	return filename
}

// ValidateFileExtension validates that a file has an allowed extension
func ValidateFileExtension(filename string, allowedExtensions []string) error {
	if len(allowedExtensions) == 0 {
		return nil // No restrictions
	}
	
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowed := range allowedExtensions {
		if strings.ToLower(allowed) == ext {
			return nil
		}
	}
	
	return fmt.Errorf("file extension '%s' is not allowed. Allowed extensions: %v", ext, allowedExtensions)
}

// ValidateContentSize validates that content size is within limits
func ValidateContentSize(size int64, maxSize int64) error {
	if maxSize > 0 && size > maxSize {
		return fmt.Errorf("file size %d bytes exceeds maximum allowed size of %d bytes", size, maxSize)
	}
	return nil
}

// SecureJoinPath safely joins a base path with a filename, ensuring the result stays within the base directory
func SecureJoinPath(basePath, filename string) (string, error) {
	// Validate the filename first
	if err := ValidateUploadName(filename); err != nil {
		return "", fmt.Errorf("invalid filename: %v", err)
	}
	
	// Clean the base path
	basePath = filepath.Clean(basePath)
	
	// Join the paths
	fullPath := filepath.Join(basePath, filename)
	
	// Ensure the result is within the base directory
	if !strings.HasPrefix(filepath.Clean(fullPath), basePath) {
		return "", fmt.Errorf("resulting path would escape base directory")
	}
	
	return fullPath, nil
}

// Encryptor provides encryption/decryption functionality for client data
type Encryptor struct {
	// This is a simplified implementation for demo purposes
	// In production, use proper key management and encryption libraries
}

// NewEncryptor creates a new encryptor instance
func NewEncryptor() (*Encryptor, error) {
	return &Encryptor{}, nil
}

// Encrypt encrypts data using the provided key ID
func (e *Encryptor) Encrypt(data []byte, keyID string) ([]byte, error) {
	// This is a placeholder implementation
	// In production, implement proper encryption with key rotation
	return data, nil
}

// Decrypt decrypts data using the provided key ID
func (e *Encryptor) Decrypt(encryptedData []byte, keyID string) ([]byte, error) {
	// This is a placeholder implementation
	// In production, implement proper decryption
	return encryptedData, nil
}

