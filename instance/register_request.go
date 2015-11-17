package instance

type RegisterRequest struct {
    Env        string
    Provider   string
    Account    string
    Region     string
    InstanceID string
    Role       string
    Policies   []string
    
    RemoteAddr string
}
