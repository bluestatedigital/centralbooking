package v1_test

import (
    "github.com/bluestatedigital/centralbooking/interfaces"
    "github.com/bluestatedigital/centralbooking/v1"
    "github.com/bluestatedigital/centralbooking/instance"
    
    vaultapi "github.com/hashicorp/vault/api"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    
    "github.com/stretchr/testify/mock"
    
    "strings"
    "io/ioutil"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "github.com/gorilla/mux"
)

var _ = Describe("CentralBooking v1", func() {
    var cb *v1.CentralBooking
    var router *mux.Router
    var resp *httptest.ResponseRecorder

    var mockVaultClient interfaces.MockVaultClient
    var mockVaultClientTemp interfaces.MockVaultClient
    
    BeforeEach(func() {
        router = mux.NewRouter()
        
        resp = httptest.NewRecorder()
        
        mockVaultClient = interfaces.MockVaultClient{}

        mockVaultClientTemp = interfaces.MockVaultClient{}

        cb = v1.NewCentralBooking(
            instance.NewRegistrar(&mockVaultClient),
            "https://vault.example.com/",
            []string{ "127.0.0.1:8300", "127.0.0.2:8300" },
        )
        cb.InstallHandlers(router.PathPrefix("/v1").Subrouter())
    })
    
    Describe("instance registration", func() {
        endpoint := "http://example.com/v1/register/instance"
        
        It("should fail with invalid GET verb", func() {
            req, err := http.NewRequest("GET", endpoint, nil)
            Expect(err).To(BeNil())

            router.ServeHTTP(resp, req)
            Expect(resp.Code).To(Equal(404))
        })
        
        It("should fail if policies not provided", func() {
            req, err := http.NewRequest(
                "POST", endpoint,
                strings.NewReader(`{
                    "environment": "dev",
                    "provider":    "aws",
                    "account":     "gen",
                    "region":      "us-east-1",
                    "instance_id": "i-04c9c4c4",
                    "role":        "cluster-server"
                }`),
            )
            Expect(err).To(BeNil())

            router.ServeHTTP(resp, req)
            Expect(resp.Code).To(Equal(400))
            
            mockVaultClient.AssertExpectations(GinkgoT())
            mockVaultClientTemp.AssertExpectations(GinkgoT())
        })
        
        It("should fail if root policy requested", func() {
            req, err := http.NewRequest(
                "POST", endpoint,
                strings.NewReader(`{
                    "environment": "dev",
                    "provider":    "aws",
                    "account":     "gen",
                    "region":      "us-east-1",
                    "instance_id": "i-04c9c4c4",
                    "role":        "cluster-server",
                    "policies":    [ "root" ]
                }`),
            )
            Expect(err).To(BeNil())

            router.ServeHTTP(resp, req)
            Expect(resp.Code).To(Equal(400))
            
            mockVaultClient.AssertExpectations(GinkgoT())
            mockVaultClientTemp.AssertExpectations(GinkgoT())
        })

        Describe("in aws", func() {
            It("processes request successfully", func() {
                // @todo retrieves instance detail from aws
                // @todo retrieves coord cluster consul server addresses from *somewhere*

                // generates perm vault token
                mockVaultClient.
                    On("CreateToken", &vaultapi.TokenCreateRequest{
                        DisplayName: "perm instance dev/aws/gen/us-east-1/i-04c9c4c4",
                        Policies: []string{ "instance-management" },
                        Metadata: map[string]string{
                            "environment": "dev",
                            "provider":    "aws",
                            "account":     "gen",
                            "region":      "us-east-1",
                            "instance_id": "i-04c9c4c4",
                            "role":        "cluster-server",
                        },
                        Lease: "72h",
                        NoParent: true,
                    }).
                    Return(
                        &vaultapi.Secret{
                            LeaseID: "",
                            LeaseDuration: 0,
                            Renewable: false,
                            Auth: &vaultapi.SecretAuth{
                                ClientToken: "generated-perm-token",
                                Policies: []string{ "instance-management" },
                                Metadata: map[string]string{
                                    "environment": "dev",
                                    "provider":    "aws",
                                    "account":     "gen",
                                    "region":      "us-east-1",
                                    "instance_id": "i-04c9c4c4",
                                    "role":        "cluster-server",
                                },
                                LeaseDuration: 0,
                                Renewable: false,
                            },
                        },
                        nil,
                    ).
                    Once()
                
                // generates temp vault token
                mockVaultClient.
                    On("CreateToken", &vaultapi.TokenCreateRequest{
                        DisplayName: "temp instance dev/aws/gen/us-east-1/i-04c9c4c4",
                        // Policies: []string{ "cubbyhole-read" }, @todo no root policy!
                        Metadata: map[string]string{
                            "environment": "dev",
                            "provider":    "aws",
                            "account":     "gen",
                            "region":      "us-east-1",
                            "instance_id": "i-04c9c4c4",
                            "role":        "cluster-server",
                        },
                        Lease: "15s",
                        NumUses: 2,
                    }).
                    Return(
                        &vaultapi.Secret{
                            LeaseID: "",
                            LeaseDuration: 0,
                            Renewable: false,
                            Auth: &vaultapi.SecretAuth{
                                ClientToken: "generated-temp-token",
                                Metadata: map[string]string{
                                    "environment": "dev",
                                    "provider":    "aws",
                                    "account":     "gen",
                                    "region":      "us-east-1",
                                    "instance_id": "i-04c9c4c4",
                                    "role":        "cluster-server",
                                },
                                LeaseDuration: 15,
                                Renewable: true, // huh, that's weird
                            },
                        },
                        nil,
                    ).
                    Once()
                
                // writes perm token payload to temp cubbyhole
                mockVaultClient.On("WithToken", "generated-temp-token").Return(&mockVaultClientTemp)
                mockVaultClientTemp.
                    On("WriteSecret", "cubbyhole/perm", mock.AnythingOfType("map[string]interface {}")).
                    Return(nil, nil).
                    Once()

                // returns payload with temp token, consul server addresses, vault endpoint

                req, err := http.NewRequest(
                    "POST", endpoint,
                    strings.NewReader(`{
                        "environment": "dev",
                        "provider":    "aws",
                        "account":     "gen",
                        "region":      "us-east-1",
                        "instance_id": "i-04c9c4c4",
                        "role":        "cluster-server",
                        
                        "policies":    ["instance-management"]
                    }`),
                )
                Expect(err).To(BeNil())

                router.ServeHTTP(resp, req)
                Expect(resp.Code).To(Equal(200))
                
                mockVaultClient.AssertExpectations(GinkgoT())
                mockVaultClientTemp.AssertExpectations(GinkgoT())
                
                var respPayload map[string]interface{}
                respBytes, _ := ioutil.ReadAll(resp.Body)
                Expect(json.Unmarshal(respBytes, &respPayload)).To(BeNil())
                
                Expect(respPayload["temp_token"]).To(Equal("generated-temp-token"), "temp token")
                Expect(respPayload["vault_endpoint"]).To(Equal("https://vault.example.com/"), "vault endpoint")
                Expect(respPayload["consul_servers"]).To(ContainElement("127.0.0.2:8300"), "missing consul servers")

                // validate the payload of the cubbyhole/perm secret
                writePermSecretCall := mockVaultClientTemp.Calls[0]
                Expect(writePermSecretCall.Method).To(Equal("WriteSecret"))
                
                permData := writePermSecretCall.Arguments.Get(1).(map[string]interface{})
                Expect(permData["payload"]).To(BeAssignableToTypeOf(&vaultapi.Secret{}))
                
                secretPayload := permData["payload"].(*vaultapi.Secret)
                Expect(secretPayload.Auth.ClientToken).To(Equal("generated-perm-token"), "incorrect perm token")

                // the rest of the payload should be fine
            })
        })
    })
})
