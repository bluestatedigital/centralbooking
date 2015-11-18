package instance

// instance registration

import (
    log "github.com/Sirupsen/logrus"
    "fmt"
    "errors"
    
    "github.com/bluestatedigital/centralbooking/interfaces"
    
    vaultapi "github.com/hashicorp/vault/api"
)

type Registrar struct {
    vaultClient interfaces.VaultClient
}

func NewRegistrar(vaultClient interfaces.VaultClient) *Registrar {
    return &Registrar{
        vaultClient: vaultClient,
    }
}

func (self *Registrar) Register(req *RegisterRequest) (*RegisterResponse, error) {
    var err error

    logEntry := log.WithField("remote_ip", req.RemoteAddr)
    
    metadata := map[string]string{
        "environment": req.Env,
        "provider":    req.Provider,
        "account":     req.Account,
        "region":      req.Region,
        "instance_id": req.InstanceID,
        "role":        req.Role,
    }
    
    logEntry = logEntry.WithFields(log.Fields{
        "environment": req.Env,
        "provider":    req.Provider,
        "account":     req.Account,
        "region":      req.Region,
        "instance_id": req.InstanceID,
        "role":        req.Role,
    })
    
    // at least one policy must be provided
    if len(req.Policies) == 0 {
        return nil, &ValidationError{"no policies specified"}
    }
    
    // disallow creating tokens with the root policy
    for _, p := range req.Policies {
        if p == "root" {
            return nil, &ValidationError{"illegal policy"}
        }
    }
    
    logEntry.Info("registering instance")

    logEntry.Debug("creating perm token")    
    permSecret, err := self.vaultClient.CreateToken(&vaultapi.TokenCreateRequest{
        DisplayName: fmt.Sprintf(
            "perm instance %s/%s/%s/%s/%s",
            req.Env,
            req.Provider,
            req.Account,
            req.Region,
            req.InstanceID,
        ),
        Policies: req.Policies,
        Metadata: metadata,
        Lease: "72h",
        NoParent: true,
    })
    
    if err != nil {
        logEntry.Errorf("error creating perm token: %+v", err)
        return nil, errors.New("unable to create token")
    }
    
    logEntry.Debug("creating temp token")    
    tempSecret, err := self.vaultClient.CreateToken(&vaultapi.TokenCreateRequest{
        DisplayName: fmt.Sprintf(
            "temp instance %s/%s/%s/%s/%s",
            req.Env,
            req.Provider,
            req.Account,
            req.Region,
            req.InstanceID,
        ),
        Metadata: metadata,
        Lease: "15s",
        NumUses: 2,
    })
    
    if err != nil {
        logEntry.Errorf("error creating temp token: %+v", err)
        return nil, errors.New("unable to create token")
    }

    logEntry.Debug("writing to cubbyhole/perm")    
    self.vaultClient. // @todo check error!
        WithToken(tempSecret.Auth.ClientToken).
        WriteSecret("cubbyhole/perm", map[string]interface{}{
            "payload": permSecret,
        })

    return &RegisterResponse{tempSecret.Auth.ClientToken}, nil
}
