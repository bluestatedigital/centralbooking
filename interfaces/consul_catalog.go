package interfaces

import (
    "github.com/hashicorp/consul/api"
)

type ConsulCatalog interface {
    Service(service, tag string, q *api.QueryOptions) ([]*api.CatalogService, *api.QueryMeta, error)
}
