package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const tokensFile = "tokens.yaml"

// TokenSet holds OAuth2 tokens for a single service.
type TokenSet struct {
	AccessToken  string    `yaml:"access_token"`
	RefreshToken string    `yaml:"refresh_token"`
	ExpiresAt    time.Time `yaml:"expires_at"`
	TokenType    string    `yaml:"token_type"`
	Scope        string    `yaml:"scope"`
}

// IsExpired returns true if the access token is expired or expires within 60 seconds.
func (t *TokenSet) IsExpired() bool {
	if t.ExpiresAt.IsZero() {
		return true
	}
	return time.Now().Add(60 * time.Second).After(t.ExpiresAt)
}

// Tokens holds all token sets organized by profile and service.
type Tokens struct {
	Profiles map[string]map[string]*TokenSet `yaml:"profiles"`
}

// LoadTokens reads the tokens file.
func LoadTokens() (*Tokens, error) {
	dir := DefaultConfigDir()
	path := filepath.Join(dir, tokensFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Tokens{Profiles: make(map[string]map[string]*TokenSet)}, nil
		}
		return nil, fmt.Errorf("reading tokens: %w", err)
	}

	var t Tokens
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parsing tokens: %w", err)
	}
	if t.Profiles == nil {
		t.Profiles = make(map[string]map[string]*TokenSet)
	}
	return &t, nil
}

// Save writes the tokens file with 0600 permissions.
func (t *Tokens) Save() error {
	dir := DefaultConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshaling tokens: %w", err)
	}

	path := filepath.Join(dir, tokensFile)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("opening tokens file: %w", err)
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// Get returns the TokenSet for the given profile and service.
// Returns nil, nil if not found.
func (t *Tokens) Get(profile, service string) (*TokenSet, error) {
	services, ok := t.Profiles[profile]
	if !ok {
		return nil, nil
	}
	ts, ok := services[service]
	if !ok {
		return nil, nil
	}
	return ts, nil
}

// Set stores a TokenSet for the given profile and service.
func (t *Tokens) Set(profile, service string, ts *TokenSet) {
	if t.Profiles == nil {
		t.Profiles = make(map[string]map[string]*TokenSet)
	}
	if t.Profiles[profile] == nil {
		t.Profiles[profile] = make(map[string]*TokenSet)
	}
	t.Profiles[profile][service] = ts
}

// Delete removes the TokenSet for the given profile and service.
func (t *Tokens) Delete(profile, service string) {
	if services, ok := t.Profiles[profile]; ok {
		delete(services, service)
	}
}

// DeleteProfile removes all tokens for the given profile.
func (t *Tokens) DeleteProfile(profile string) {
	delete(t.Profiles, profile)
}

// ListServices returns the services that have tokens for the given profile.
func (t *Tokens) ListServices(profile string) []string {
	services, ok := t.Profiles[profile]
	if !ok {
		return nil
	}
	result := make([]string, 0, len(services))
	for svc := range services {
		result = append(result, svc)
	}
	return result
}
