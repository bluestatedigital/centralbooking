// implements the v1 centralbooking api
package v1

import (
    "fmt"
    "net"
    "net/http"
    "io/ioutil"
    "encoding/json"
    
    log "github.com/Sirupsen/logrus"

    "github.com/gorilla/mux"
    
    "bitbucket.org/bluestatedigital/centralbooking/interfaces"
    
    vaultapi "github.com/hashicorp/vault/api"
)

type CentralBooking struct {
    vaultClient       interfaces.VaultClient
    consulServerAddrs []string
}

// returns a new CentralBooking instance
func NewCentralBooking(vaultClient interfaces.VaultClient, consulServerAddrs []string) *CentralBooking {
    return &CentralBooking{
        vaultClient:       vaultClient,
        consulServerAddrs: consulServerAddrs,
    }
}

// install handlers into the provided router
func (self *CentralBooking) InstallHandlers(router *mux.Router) {
    router.
        Methods("POST").
        Path("/register/instance").
        HandlerFunc(self.RegisterInstance)
}

// returns the index view
func (self *CentralBooking) RegisterInstance(resp http.ResponseWriter, req *http.Request) {
    var err error
    var remoteAddr string
    
    vaultEndpoint := self.vaultClient.GetEndpoint()
    
    if xff, ok := req.Header["X-Forwarded-For"]; ok {
        remoteAddr = xff[0]
    } else {
        remoteAddr, _, err = net.SplitHostPort(req.RemoteAddr)
        if err != nil {
            log.Errorf("unable to parse RemoteAddr: %s", err)
            remoteAddr = req.RemoteAddr
        }
    }
    
    logEntry := log.WithField("remote_ip", remoteAddr)
    
    type RegisterRequest struct {
        Environment string
        Provider    string
        Account     string
        Region      string
        Instance_ID string
        Role        string
        Policies    []string
    }
    
    var payload RegisterRequest

    body, err := ioutil.ReadAll(req.Body)
    if err != nil {
        log.Errorf("unable to read body: %s", err)
        http.Error(resp, "unable to read body", http.StatusBadRequest)
        return
    }
    
    err = json.Unmarshal(body, &payload)
    if err != nil {
        log.Errorf("unable to decode payload: %s", err)
        http.Error(resp, "unable to decode payload", http.StatusBadRequest)
        return
    }
    
    metadata := map[string]string{
        "environment": payload.Environment,
        "provider":    payload.Provider,
        "account":     payload.Account,
        "region":      payload.Region,
        "instance_id": payload.Instance_ID,
        "role":        payload.Role,
    }
    
    logEntry = logEntry.WithFields(log.Fields{
        "environment": payload.Environment,
        "provider":    payload.Provider,
        "account":     payload.Account,
        "region":      payload.Region,
        "instance_id": payload.Instance_ID,
        "role":        payload.Role,
    })
    
    // at least one policy must be provided
    if len(payload.Policies) == 0 {
        http.Error(resp, "no policies specified", http.StatusBadRequest)
        return
    }
    
    // disallow creating tokens with the root policy
    for _, p := range payload.Policies {
        if p == "root" {
            http.Error(resp, "nice try, bucko", http.StatusForbidden)
            return
        }
    }
    
    logEntry.Info("registering instance")

    logEntry.Debug("creating perm token")    
    permSecret, err := self.vaultClient.CreateToken(&vaultapi.TokenCreateRequest{
        DisplayName: fmt.Sprintf(
            "perm instance %s/%s/%s/%s/%s",
            payload.Environment,
            payload.Provider,
            payload.Account,
            payload.Region,
            payload.Instance_ID,
        ),
        Policies: payload.Policies,
        Metadata: metadata,
        Lease: "72h",
        NoParent: true,
    })
    
    if err != nil {
        logEntry.Errorf("error creating token: %+v", err)
    }
    
    logEntry.Debug("creating temp token")    
    tempSecret, err := self.vaultClient.CreateToken(&vaultapi.TokenCreateRequest{
        DisplayName: fmt.Sprintf(
            "temp instance %s/%s/%s/%s/%s",
            payload.Environment,
            payload.Provider,
            payload.Account,
            payload.Region,
            payload.Instance_ID,
        ),
        Metadata: metadata,
        Lease: "15s",
        NumUses: 2,
    })
    
    if err != nil {
        logEntry.Errorf("error creating token: %+v", err)
    }
    
    logEntry.Debug("writing to cubbyhole/perm")    
    self.vaultClient.
        WithToken(tempSecret.Auth.ClientToken).
        WriteSecret("cubbyhole/perm", map[string]interface{}{
            "payload": permSecret,
        })
    
    respBytes, err := json.Marshal(map[string]interface{}{
        "temp_token":     tempSecret.Auth.ClientToken,
        "vault_endpoint": vaultEndpoint,
        "consul_servers": self.consulServerAddrs,
    })
    if err != nil {
        log.Errorf("unable to marshal response body: %s", err)
        http.Error(resp, "failed generating response body", http.StatusInternalServerError)
        return
    }

    resp.Header().Add("Content-Type", "application/json")
    resp.WriteHeader(http.StatusOK)
    resp.Write(respBytes)
}
