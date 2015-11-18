package helpers

import (
    "github.com/bluestatedigital/centralbooking/interfaces"
    "github.com/hashicorp/vault/api"
)

type VaultClient struct {
    vaultClient *api.Client
    config      *api.Config
}

func NewVaultClient(vaultEndpoint string, token string) (interfaces.VaultClient, error) {
    cfg := api.DefaultConfig()
    cfg.ReadEnvironment()
    cfg.Address = vaultEndpoint
    
    vault, err := api.NewClient(cfg)
    if err != nil {
        return nil, err
    }
    
    vault.SetToken(token)
    
    return &VaultClient{
        vaultClient: vault,
        config:      cfg,
    }, nil
}

func (self *VaultClient) GetEndpoint() string {
    return self.config.Address
}

func (self *VaultClient) WithToken(token string) interfaces.VaultClient {
    // swallow the error; we didn't get here if the endpoint was invalid, and
    // that's all NewClient (currently) checks for.
    vc, _ := NewVaultClient(self.GetEndpoint(), token)

    return vc
}

func (self *VaultClient) CreateToken(opts *api.TokenCreateRequest) (*api.Secret, error) {
    return self.vaultClient.Auth().Token().Create(opts)
}

func (self *VaultClient) WriteSecret(path string, data map[string]interface{}) (*api.Secret, error) {
    return self.vaultClient.Logical().Write(path, data)
}
