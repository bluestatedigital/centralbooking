# Central registration authority for dynamic instances

## summary

Provides a [Vault](https://vaultproject.io) token and a list of [Consul](https://consul.io) WAN addresses to allow a newly-launched instance to join an existing network.

## description

This service is designed around the [Cubbyhole Authentication Principles](https://hashicorp.com/blog/vault-cubbyhole-principles.html) post on the Hashicorp blog.  The `temp_token` in the response to a `POST` to `/v1/register/instance` is exchanged for a "perm" token from Vault.  That is in turn used to retrieve other credentials from Vault necessary for bootstrapping the instance.  These may include a Consul ACL token, the gossip encryption key, a TLS certificate for Consul, and other credentials or tokens needed by applications.  This workflow allows an instance access to sensitive credentials from Vault while still functioning in a fully auto-scaled environment.

When an instance registers with centralbooking, a number of factors are used to verify its identity. (@todo!)

## registering an instance

    curl -s -X POST \
        -d '{
            "environment": "dev",
            "provider":    "aws",
            "account":     "gen",
            "region":      "us-east-1",
            "instance_id": "i-04c9c4c4",
            "role":        "cluster-server",
            "policies":    ["instance-management"]
        }' \
         "http://centralbooking/v1/register/instance"

response:

    {
        "temp_token":     "0b54bd3c-d649-48af-b44f-d16d738ae07c",
        "vault_endpoint": "https://vault.example.com",
        "consul_servers": [
            "10.0.1.1:8302",
            "10.0.1.2:8302",
            "10.0.1.3:8302"
        ]
    }

## retrieving the perm token

    VAULT_TOKEN="<temp_token from above>" vault read cubbyhole/perm

# @todos

* include the Consul ACL datacenter
* build list of Consul servers dynamically
* validate the instance against the cloud provider
* record instance metadata in Consul

