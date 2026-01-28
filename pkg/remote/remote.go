// Package remote provides SSH/SFTP remote file operations
package remote

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Config holds SSH connection configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	KeyFile  string
}

// Client wraps SSH and SFTP clients for remote operations
type Client struct {
	config     *Config
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

// NewClient creates a new remote client with the given configuration
func NewClient(config *Config) *Client {
	if config.Port == 0 {
		config.Port = 22
	}
	return &Client{config: config}
}

// Connect establishes SSH and SFTP connections
func (c *Client) Connect() error {
	var authMethods []ssh.AuthMethod

	// Try key-based auth first
	if c.config.KeyFile != "" {
		key, err := os.ReadFile(c.config.KeyFile)
		if err == nil {
			signer, err := ssh.ParsePrivateKey(key)
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	// Try default SSH keys
	if len(authMethods) == 0 {
		homeDir, _ := os.UserHomeDir()
		keyPaths := []string{
			path.Join(homeDir, ".ssh", "id_rsa"),
			path.Join(homeDir, ".ssh", "id_ed25519"),
			path.Join(homeDir, ".ssh", "id_ecdsa"),
		}
		for _, keyPath := range keyPaths {
			key, err := os.ReadFile(keyPath)
			if err != nil {
				continue
			}
			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				continue
			}
			authMethods = append(authMethods, ssh.PublicKeys(signer))
			break
		}
	}

	// Fall back to password if provided
	if c.config.Password != "" {
		authMethods = append(authMethods, ssh.Password(c.config.Password))
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("no authentication method available")
	}

	sshConfig := &ssh.ClientConfig{
		User:            c.config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: proper host key verification
	}

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	c.sshClient = sshClient

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	c.sftpClient = sftpClient

	return nil
}

// Close closes all connections
func (c *Client) Close() error {
	if c.sftpClient != nil {
		c.sftpClient.Close()
	}
	if c.sshClient != nil {
		c.sshClient.Close()
	}
	return nil
}

// ReadFile reads a file from the remote host
func (c *Client) ReadFile(remotePath string) ([]byte, error) {
	if c.sftpClient == nil {
		return nil, fmt.Errorf("not connected")
	}

	file, err := c.sftpClient.Open(remotePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open remote file %s: %w", remotePath, err)
	}
	defer file.Close()

	return io.ReadAll(file)
}

// WriteFile writes data to a file on the remote host
func (c *Client) WriteFile(remotePath string, data []byte, perm os.FileMode) error {
	if c.sftpClient == nil {
		return fmt.Errorf("not connected")
	}

	// Ensure parent directory exists
	dir := path.Dir(remotePath)
	c.sftpClient.MkdirAll(dir)

	file, err := c.sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file %s: %w", remotePath, err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to remote file %s: %w", remotePath, err)
	}

	return c.sftpClient.Chmod(remotePath, perm)
}

// Stat returns file info for a remote path
func (c *Client) Stat(remotePath string) (os.FileInfo, error) {
	if c.sftpClient == nil {
		return nil, fmt.Errorf("not connected")
	}
	return c.sftpClient.Stat(remotePath)
}

// ReadDir reads a directory on the remote host
func (c *Client) ReadDir(remotePath string) ([]os.FileInfo, error) {
	if c.sftpClient == nil {
		return nil, fmt.Errorf("not connected")
	}
	return c.sftpClient.ReadDir(remotePath)
}

// FileExists checks if a file exists on the remote host
func (c *Client) FileExists(remotePath string) bool {
	_, err := c.Stat(remotePath)
	return err == nil
}

// GetHomeDir gets the home directory on the remote host
func (c *Client) GetHomeDir() (string, error) {
	if c.sftpClient == nil {
		return "", fmt.Errorf("not connected")
	}
	return c.sftpClient.Getwd()
}

// RunCommand executes a command on the remote host
func (c *Client) RunCommand(command string) ([]byte, error) {
	if c.sshClient == nil {
		return nil, fmt.Errorf("not connected")
	}

	session, err := c.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	return session.CombinedOutput(command)
}
