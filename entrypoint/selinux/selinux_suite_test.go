package selinux

import (
	"flag"
	"os"
	"testing"

	"github.com/rancher/distros-test-framework/config"
	"github.com/rancher/distros-test-framework/internal/pkg/customflag"
	"github.com/rancher/distros-test-framework/internal/pkg/k8s"
	"github.com/rancher/distros-test-framework/internal/pkg/testcase"
	"github.com/rancher/distros-test-framework/internal/resources"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	kubeconfig string
	cluster    *resources.Cluster
	k8sClient  *k8s.Client
	cfg        *config.Env
	err        error
)

func TestMain(m *testing.M) {
	flag.Var(&customflag.ServiceFlag.Destroy, "destroy", "Destroy cluster after test")
	flag.Var(&customflag.ServiceFlag.InstallMode, "installVersionOrCommit", "Install upgrade customflag for version bump")
	flag.Var(&customflag.ServiceFlag.Channel, "channel", "channel to use on install or upgrade")
	flag.Parse()

	cfg, err = config.AddEnv()
	if err != nil {
		resources.LogLevel("error", "error adding env vars: %w\n", err)
		os.Exit(1)
	}

	kubeconfig = os.Getenv("KUBE_CONFIG")
	if kubeconfig == "" {
		// gets a cluster from terraform.
		cluster = resources.ClusterConfig(cfg.Product, cfg.Module)
	} else {
		// gets a cluster from kubeconfig.
		cluster = resources.KubeConfigCluster(kubeconfig)
	}

	k8sClient, err = k8s.AddClient()
	if err != nil {
		resources.LogLevel("error", "error adding k8s: %w\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestSelinuxSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Selinux Test Suite")
}

var _ = AfterSuite(func() {
	if customflag.ServiceFlag.Destroy {
		testcase.TestUninstallPolicy(cluster, true)
		status, err := resources.DestroyInfrastructure(cfg.Product, cfg.Module)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal("cluster destroyed"))
	}
})
