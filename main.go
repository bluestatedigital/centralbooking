package main

import (
    "os"
    "fmt"
    "syscall"
    "net/http"
    
    flags "github.com/jessevdk/go-flags"
    log "github.com/Sirupsen/logrus"
    
    "github.com/bluestatedigital/centralbooking/v1"
    "github.com/bluestatedigital/centralbooking/helpers"
    "github.com/bluestatedigital/centralbooking/instance"
    
    "github.com/gorilla/mux"
)

var version string = "undef"

type Options struct {
    Debug      bool   `env:"DEBUG"     long:"debug"    description:"enable debug"`
    LogFile    string `env:"LOG_FILE"  long:"log-file" description:"path to JSON log file"`
    
    HttpPort   int    `env:"HTTP_PORT" long:"port"     description:"port to accept requests on" default:"8080"`
    
    VaultAddr  string `env:"VAULT_ADDR"  long:"vault-addr"  description:"address of the Vault server"     required:"true"`
    VaultToken string `env:"VAULT_TOKEN" long:"vault-token" description:"auth token for this application" required:"true"`
    
    // @todo kludge until I figure out how to retrieve the list of consul servers for the wan pool
    ConsulServerAddresses []string `env:"CONSUL_SERVER_ADDRS" env-delim:"," long:"consul-server-addr" description:"consul server addresses sent to clients so they can join the wan pool" required:"true"`
}

func Log(handler http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Infof("%s %s %s", r.RemoteAddr, r.Method, r.URL)
        handler.ServeHTTP(w, r)
    })
}

func main() {
    var opts Options
    
    _, err := flags.Parse(&opts)
    if err != nil {
        os.Exit(1)
    }
    
    if opts.Debug {
        log.SetLevel(log.DebugLevel)
    }
    
    if opts.LogFile != "" {
        logFp, err := os.OpenFile(opts.LogFile, os.O_WRONLY | os.O_APPEND | os.O_CREATE, 0600)
        checkError(fmt.Sprintf("error opening %s", opts.LogFile), err)
        
        defer logFp.Close()
        
        // ensure panic output goes to log file
        syscall.Dup2(int(logFp.Fd()), 1)
        syscall.Dup2(int(logFp.Fd()), 2)
        
        // log as JSON
        log.SetFormatter(&log.JSONFormatter{})
        
        // send output to file
        log.SetOutput(logFp)
    }
    
    log.Debug("hi there! (tickertape tickertape)")
    log.Infof("version: %s", version)
    
    vaultClient, err := helpers.NewVaultClient(opts.VaultAddr, opts.VaultToken)
    checkError("creating Vault client", err)
    
    router := mux.NewRouter()
    
    registrar := instance.NewRegistrar(vaultClient)
    v1 := v1.NewCentralBooking(
        registrar,
        vaultClient.GetEndpoint(),
        opts.ConsulServerAddresses,
    )
    v1.InstallHandlers(router.PathPrefix("/v1").Subrouter())
    
    httpServer := &http.Server{
        Addr: fmt.Sprintf(":%d", opts.HttpPort),
        Handler: Log(router),
    }
    
    checkError("launching HTTP server", httpServer.ListenAndServe())
}
