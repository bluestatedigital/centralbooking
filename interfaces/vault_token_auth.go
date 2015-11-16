package interfaces

import (
    "github.com/hashicorp/vault/api"
)

type VaultTokenAuth interface {
    Create(opts *api.TokenCreateRequest) (*api.Secret, error)
}
