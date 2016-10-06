package instance_test

import (
    "github.com/bluestatedigital/centralbooking/instance"
    "github.com/bluestatedigital/centralbooking/interfaces"
    
    vaultapi "github.com/hashicorp/vault/api"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    
    "github.com/stretchr/testify/mock"
)

var _ = Describe("CentralBooking v1", func() {
    var registrar *instance.Registrar

    var mockVaultClient interfaces.MockVaultClient
    var mockVaultClientTemp interfaces.MockVaultClient
    
    BeforeEach(func() {
        mockVaultClient = interfaces.MockVaultClient{}
        mockVaultClientTemp = interfaces.MockVaultClient{}

        registrar = instance.NewRegistrar(
            &mockVaultClient,
        )
    })
    
    Describe("instance registration", func() {
        It("should fail if policies not provided", func() {
            req := &instance.RegisterRequest{
                Env:        "dev",
                Provider:   "aws",
                Account:    "gen",
                Region:     "us-east-1",
                InstanceID: "i-04c9c4c4",
                Role:       "cluster-server",
                Policies:   []string{},
            }
            _, err := registrar.Register(req)
            Expect(err).To(MatchError("no policies specified"))
        })
        
        It("should fail if root policy requested", func() {
            req := &instance.RegisterRequest{
                Env:        "dev",
                Provider:   "aws",
                Account:    "gen",
                Region:     "us-east-1",
                InstanceID: "i-04c9c4c4",
                Role:       "cluster-server",
                Policies:   []string{ "root" },
            }
            _, err := registrar.Register(req)
            Expect(err).To(MatchError("illegal policy"))
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

                // start the test (*phew!*)
                req := &instance.RegisterRequest{
                    Env:        "dev",
                    Provider:   "aws",
                    Account:    "gen",
                    Region:     "us-east-1",
                    InstanceID: "i-04c9c4c4",
                    Role:       "cluster-server",
                    Policies:   []string{ "instance-management" },
                }
                resp, err := registrar.Register(req)
                Expect(err).To(BeNil())

                mockVaultClient.AssertExpectations(GinkgoT())
                mockVaultClientTemp.AssertExpectations(GinkgoT())

                // returns payload with temp token, consul server addresses, vault endpoint

                Expect(resp.TempToken).To(Equal("generated-temp-token"), "temp token")
            })
        })
    })
})
