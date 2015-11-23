// implements the v1 centralbooking api
package v1

import (
    "net"
    "net/http"
    "io/ioutil"
    "encoding/json"
    
    log "github.com/Sirupsen/logrus"

    "github.com/gorilla/mux"
    
    "github.com/bluestatedigital/centralbooking/instance"
)

type CentralBooking struct {
    registrar         *instance.Registrar
    vaultEndpoint     string
    consulServerAddrs []string
}

// returns a new CentralBooking instance
func NewCentralBooking(registrar *instance.Registrar, vaultEndpoint string, consulServerAddrs []string) *CentralBooking {
    return &CentralBooking{
        registrar:         registrar,
        vaultEndpoint:     vaultEndpoint,
        consulServerAddrs: consulServerAddrs,
    }
}

// install handlers into the provided router
func (self *CentralBooking) InstallHandlers(router *mux.Router) {
    router.
        Methods("POST").
        Path("/register/instance").
        HandlerFunc(self.RegisterInstance)

    // apeing vault
    router.
        Methods("GET").
        Path("/sys/health").
        HandlerFunc(self.CheckHealth)
}

// returns the index view
func (self *CentralBooking) RegisterInstance(resp http.ResponseWriter, req *http.Request) {
    var err error
    var remoteAddr string
    
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
    
    type payloadType struct {
        Environment string
        Provider    string
        Account     string
        Region      string
        Instance_ID string
        Role        string
        Policies    []string
    }
    
    var payload payloadType

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
    
    logEntry.Info("registering instance")
    regResp, err := self.registrar.Register(&instance.RegisterRequest{
        Env:        payload.Environment,
        Provider:   payload.Provider,
        Account:    payload.Account,
        Region:     payload.Region,
        InstanceID: payload.Instance_ID,
        Role:       payload.Role,
        Policies:   payload.Policies,
    })
    
    if err != nil {
        sc := http.StatusInternalServerError
        
        if _, ok := err.(*instance.ValidationError); ok {
            sc = http.StatusBadRequest
        }
        
        http.Error(resp, err.Error(), sc)
        return
    }

    respBytes, err := json.Marshal(map[string]interface{}{
        "temp_token":     regResp.TempToken,
        "vault_endpoint": self.vaultEndpoint,
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

func (self *CentralBooking) CheckHealth(resp http.ResponseWriter, req *http.Request) {
    resp.WriteHeader(http.StatusOK)

    // http://labs.omniti.com/labs/jsend
    resp.Write([]byte(`{"status":"success"}`))
}
