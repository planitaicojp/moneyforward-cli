package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const credentialsFile = "credentials.yaml"

// profileCredentials holds sensitive credentials for a single profile.
type profileCredentials struct {
	ClientSecret string `yaml:"client_secret"`
}

// Credentials holds sensitive data organized by profile.
type Credentials struct {
	Profiles map[string]*profileCredentials `yaml:"profiles"`
}

// LoadCredentials reads the credentials file.
func LoadCredentials() (*Credentials, error) {
	dir := DefaultConfigDir()
	path := filepath.Join(dir, credentialsFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Credentials{Profiles: make(map[string]*profileCredentials)}, nil
		}
		return nil, fmt.Errorf("reading credentials: %w", err)
	}

	var c Credentials
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}
	if c.Profiles == nil {
		c.Profiles = make(map[string]*profileCredentials)
	}
	return &c, nil
}

// Save writes the credentials file with 0600 permissions.
func (c *Credentials) Save() error {
	dir := DefaultConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}

	path := filepath.Join(dir, credentialsFile)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("opening credentials file: %w", err)
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// GetClientSecret returns the client secret for the given profile.
// Returns empty string if not found.
func (c *Credentials) GetClientSecret(profile string) string {
	if cred, ok := c.Profiles[profile]; ok {
		return cred.ClientSecret
	}
	return ""
}

// SetClientSecret stores the client secret for the given profile.
func (c *Credentials) SetClientSecret(profile, secret string) {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*profileCredentials)
	}
	c.Profiles[profile] = &profileCredentials{ClientSecret: secret}
}

// DeleteProfile removes credentials for the given profile.
func (c *Credentials) DeleteProfile(profile string) {
	delete(c.Profiles, profile)
}
