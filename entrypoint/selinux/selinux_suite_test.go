package selinux

import (
	"os"
	"testing"

	"github.com/rancher/distros-test-framework/factory"
	"github.com/rancher/distros-test-framework/pkg/customflag"
	"github.com/rancher/distros-test-framework/shared"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMain(m *testing.M) {
	customflag.AddFlags("channel", "installVersionOrCommit", "destroy")

	_, err := shared.EnvConfig()
	if err != nil {
		return
	}

	customflag.ValidateFlags()

	os.Exit(m.Run())
}

func TestSelinuxSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Selinux Test Suite")
}

var _ = AfterSuite(func() {
	g := GinkgoT()
	if customflag.ServiceFlag.ClusterConfig.Destroy {
		status, err := factory.DestroyCluster(g)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal("cluster destroyed"))
	}
})
