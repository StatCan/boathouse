package agent

import (
	"time"

	vault "github.com/hashicorp/vault/api"
)

// Agent is an agent.
type Agent struct {
	vault *vault.Client
}

// NewAgent generates a new Boathouse agent.
func NewAgent(vault *vault.Client) (*Agent, error) {
	return &Agent{
		vault: vault,
	}, nil
}

// IssueCredentialRequest represents a request for credentials.
type IssueCredentialRequest struct {
	// Path is the Vault path
	Path string `json:"path"`

	// TTL is the requested time
	TTL time.Duration `json:"ttl"`
}

type Lease struct {
	ID       string        `json:"id"`
	Duration time.Duration `json:"duration"`
}

type IssueCredentialResponse struct {
	Lease     Lease  `json:"lease"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}
