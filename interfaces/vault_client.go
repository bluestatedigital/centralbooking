package interfaces

import "github.com/hashicorp/vault/api"

type VaultClient interface {
    GetEndpoint() string
    WithToken(token string) VaultClient
    CreateToken(opts *api.TokenCreateRequest) (*api.Secret, error)
    WriteSecret(path string, data map[string]interface{}) (*api.Secret, error)
}
