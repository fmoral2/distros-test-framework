package qainfra

import (
	"github.com/rancher/distros-test-framework/internal/provisioning/driver"
)

// Provider implements the driver.Provisioner interface for qainfra infrastructure.
type Provider struct{}

func New() *Provider { return &Provider{} }

func (p *Provider) Provision(cfg *driver.InfraConfig) (*driver.Cluster, error) {
	return p.provisionInfrastructure(cfg)
}

func (p *Provider) Destroy(product, module string) (string, error) {
	return p.destroyInfrastructure(product, module)
}
