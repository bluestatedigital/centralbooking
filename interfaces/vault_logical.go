package interfaces

import (
    "github.com/hashicorp/vault/api"
)

type VaultLogical interface {
    Read(path string) (*api.Secret, error)
    Write(path string, data map[string]interface{}) (*api.Secret, error)
}
