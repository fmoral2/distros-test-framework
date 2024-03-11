package dualstack

import (
	"os"
	"testing"

	"github.com/rancher/distros-test-framework/entrypoint"
	"github.com/rancher/distros-test-framework/factory"
	"github.com/rancher/distros-test-framework/pkg/customflag"
	"github.com/rancher/distros-test-framework/shared"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMain(m *testing.M) {
	entrypoint.AddFlags("destroy")

	_, err := shared.EnvConfig()
	if err != nil {
		return
	}

	os.Exit(m.Run())
}

func TestDualStackSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create Dual-Stack Cluster Test Suite")
}

var _ = AfterSuite(func() {
	g := GinkgoT()
	if customflag.ServiceFlag.ClusterConfig.Destroy {
		status, err := factory.DestroyCluster(g)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal("cluster destroyed"))
	}
})
