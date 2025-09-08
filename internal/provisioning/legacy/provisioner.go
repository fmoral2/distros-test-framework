package legacy

import (
	"github.com/rancher/distros-test-framework/internal/provisioning/driver"
)

// Provider implements the driver.Provisioner interface for legacy infrastructure
type Provider struct{}

func New() *Provider { return &Provider{} }

func (p *Provider) Provision(cfg *driver.InfraConfig) (*driver.Cluster, error) {
	return Provision(cfg.Product, cfg.Module)
}

func (p *Provider) Destroy(product, module string) (string, error) {
	return Destroy(product, module)
}
