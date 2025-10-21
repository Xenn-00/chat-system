package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Valid RSA private key for testing (2048-bit)
const testPrivateKeyPEM = `
-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAp3bPIMFEIgfqdci/B/eOjeNf8KtYxWR6kPZlKNQ7Yec2Rzgi
i0oIdZzFht3/p0XHYZtvtzmtHtdfA7Jbp5SlMRxvxBPwhos7T9d/cb2Zskd6Uhq9
inkhgBCoTYlyr9lFaOXyLBUnL5oG3/4+OV0aNRSyPfMhfE8BEj68MrG8+BFuWWtq
g0qwvlXKXX3hHfdowOfY/TlHEmz7vzUCy7sTdqn/IUwPOiP3Feow52EApn67AhaH
duqtOzOOsaiwWX3uKNG81+rKQvjBNJmGtbasBdg7oMoUKYuLGRhgafvcI3SjlY4F
Y9alEI2ncoxbVoNWXC5YpqngvuwOO3WmT2sSiQIDAQABAoIBADnkWLZ6GZOqKOOP
Ans+mYlzkTciBQ44LibvBwmWVPEDfUAhp89/SG1gROja1gZ9mO+lTHmK9s4ypiYh
Ao5sVK8lpX2jZwMcHuT7GpO3d+qpyx+XHu8/8NTU7VngqkUgV15sH8wdg+5w0O+e
dORfyy+OeA/yfSD8LuKfzW+5Ahq/W4rAUIlmdTPX50PzSJ7ER/l6nY4ZynsQwwwp
v8hicu3Xzj6AX9RC7lgH0TIRsk3Md+37Qdak+ofnU/EnUhZk0YqQzmM/aYBG+xR7
3yC3oOO8wdKB1rnZgkJhz+dtdzUw8lmT9KYUjLWSkpFwotSZef9MApBXgutjSt15
QSZiggECgYEA66rPVa7SYOHR3/jL8NWtamaen6vCNj0dGi/q43n6XCwJleE/3ZxW
yZh+ESjOdH6JZZPmXSfnOXuTRIljzK8rYhyfDIRYuORl2nTt/I/sk5jXn+h0WhNk
DMyQQSc8y3ChReBEWVC49HjzB5ZqKKQYSwqOwQIobQ2lWygKhhwl2xsCgYEAtemY
KD7gdp1zsx4/iWOfnMA5DdR0nlZzVmKrnU55LPuuMgmlKW2uDV223gFus5MAHc88
pFIw0UCwm5ki3S7Doe4pAXSuII8zfwakcq7u9Wea8OGDiyEuC7lSZr/iJ3K7xyvQ
vjid1s42Az/k3JTPZihVzwF/ybdaVcdwkKgFHysCgYBB2GS7tO/U3+Nq57HbpWgh
jXCOfkfyLZsfAPpo+mDINgmrldbpTVA2XWQD2Vnt1JkBB5TavFZviiZ4hMacnujJ
LeQGdEfxyOboZblE0tWv24mLhUBVFoviw5keix8CXILC6klOhy5WKCEHIrCgkFC1
TsraBIdVCPYFhSeDlwPAtwKBgA7ojQLHXGf8MW49jWF6G6uiCUr73W7YkO1EeuIS
e1XXbohFSBbkGT6ZLpJ1NZhb9Md8o3CoF74eehrWawgLfBb6SLwIzvh2I/dGGRYZ
BhZwnj8djEVLu9VUI8+t7B/lhEQncB0W0MC4964+f0ggnfq2VYn4inuCnlGnXa9N
RdYvAoGAXUzcHUuJIQzOfw+3NxdIiA+BWNfHht7aWt3XnL2JkNbneQ82ABmdst7V
bT7APoULu1nIk95iQ8YNm182dFyCVUI0rTy9W3D5+dRchyzg5Plm8swtsXJkFhGb
7QFSMDBgs64rreOse4pw8cc34CStM5tD4RMZ/6byqXlQ1g3tH3Q=
-----END RSA PRIVATE KEY-----
`

// Valid RSA public key for testing (corresponding to above private key)
const testPublicKeyPEM = `
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAp3bPIMFEIgfqdci/B/eO
jeNf8KtYxWR6kPZlKNQ7Yec2Rzgii0oIdZzFht3/p0XHYZtvtzmtHtdfA7Jbp5Sl
MRxvxBPwhos7T9d/cb2Zskd6Uhq9inkhgBCoTYlyr9lFaOXyLBUnL5oG3/4+OV0a
NRSyPfMhfE8BEj68MrG8+BFuWWtqg0qwvlXKXX3hHfdowOfY/TlHEmz7vzUCy7sT
dqn/IUwPOiP3Feow52EApn67AhaHduqtOzOOsaiwWX3uKNG81+rKQvjBNJmGtbas
Bdg7oMoUKYuLGRhgafvcI3SjlY4FY9alEI2ncoxbVoNWXC5YpqngvuwOO3WmT2sS
iQIDAQAB
-----END PUBLIC KEY-----
`

// Invalid PEM for testing error cases
const invalidKeyPEM = `-----BEGIN INVALID KEY-----
This is not a valid PEM key
-----END INVALID KEY-----`

func TestInitSecret_Success(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir() // Go 1.15+ automatic cleanup

	privateKeyPath := filepath.Join(tempDir, "private.pem")
	publicKeyPath := filepath.Join(tempDir, "public.pem")

	// Write test keys to temporary files
	err := os.WriteFile(privateKeyPath, []byte(testPrivateKeyPEM), 0644)
	require.NoError(t, err, "Failed to write test private key")

	err = os.WriteFile(publicKeyPath, []byte(testPublicKeyPEM), 0644)
	require.NoError(t, err, "Failed to write test public key")

	// Change working directory to temp dir (so InitSecret finds the files)
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir) // Restore original directory

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test the function
	jwtSecret, err := InitSecret()

	// Assertions
	require.NoError(t, err, "InitSecret should not return an error")
	require.NotNil(t, jwtSecret, "JwtSecret should not be nil")
	require.NotNil(t, jwtSecret.Private, "Private key should not be nil")
	require.NotNil(t, jwtSecret.Public, "Public key should not be nil")

	// Verify key properties
	assert.Equal(t, 2048, jwtSecret.Private.N.BitLen(), "Private key should be 2048-bit")
	assert.Equal(t, 2048, jwtSecret.Public.N.BitLen(), "Public key should be 2048-bit")
}

func TestInitSecret_MissingPrivateKey(t *testing.T) {
	tempDir := t.TempDir()

	// Only create public key file
	publicKeyPath := filepath.Join(tempDir, "public.pem")
	err := os.WriteFile(publicKeyPath, []byte(testPublicKeyPEM), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test the function
	jwtSecret, err := InitSecret()

	// Assertions
	assert.Error(t, err, "InitSecret should return error when private key is missing")
	assert.Nil(t, jwtSecret, "JwtSecret should be nil on error")
}

func TestInitSecret_MissingPublicKey(t *testing.T) {
	tempDir := t.TempDir()

	// Only create private key file
	privateKeyPath := filepath.Join(tempDir, "private.pem")
	err := os.WriteFile(privateKeyPath, []byte(testPrivateKeyPEM), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test the function
	jwtSecret, err := InitSecret()

	// Assertions
	assert.Error(t, err, "InitSecret should return error when public key is missing")
	assert.Nil(t, jwtSecret, "JwtSecret should be nil on error")
}

func TestInitSecret_InvalidPrivateKey(t *testing.T) {
	tempDir := t.TempDir()

	// Create files with invalid keys
	privateKeyPath := filepath.Join(tempDir, "private.pem")
	publicKeyPath := filepath.Join(tempDir, "public.pem")

	err := os.WriteFile(privateKeyPath, []byte(invalidKeyPEM), 0644)
	require.NoError(t, err)

	err = os.WriteFile(publicKeyPath, []byte(testPublicKeyPEM), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test the function
	jwtSecret, err := InitSecret()

	// Assertions
	assert.Error(t, err, "InitSecret should return error with invalid private key")
	assert.Nil(t, jwtSecret, "JwtSecret should be nil on error")
	assert.Contains(t, err.Error(), "invalid private key", "Error message should mention invalid private key")
}

func TestInitSecret_InvalidPublicKey(t *testing.T) {
	tempDir := t.TempDir()

	// Create files with invalid public key
	privateKeyPath := filepath.Join(tempDir, "private.pem")
	publicKeyPath := filepath.Join(tempDir, "public.pem")

	err := os.WriteFile(privateKeyPath, []byte(testPrivateKeyPEM), 0644)
	require.NoError(t, err)

	err = os.WriteFile(publicKeyPath, []byte(invalidKeyPEM), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test the function
	jwtSecret, err := InitSecret()

	// Assertions
	assert.Error(t, err, "InitSecret should return error with invalid public key")
	assert.Nil(t, jwtSecret, "JwtSecret should be nil on error")
	assert.Contains(t, err.Error(), "invalid public key", "Error message should mention invalid public key")
}

// Test empty files
func TestInitSecret_EmptyFiles(t *testing.T) {
	tempDir := t.TempDir()

	privateKeyPath := filepath.Join(tempDir, "private.pem")
	publicKeyPath := filepath.Join(tempDir, "public.pem")

	// Create empty files
	err := os.WriteFile(privateKeyPath, []byte(""), 0644)
	require.NoError(t, err)

	err = os.WriteFile(publicKeyPath, []byte(""), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test the function
	jwtSecret, err := InitSecret()

	// Assertions
	assert.Error(t, err, "InitSecret should return error with empty files")
	assert.Nil(t, jwtSecret, "JwtSecret should be nil on error")
}
