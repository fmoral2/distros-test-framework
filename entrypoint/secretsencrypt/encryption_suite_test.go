package secretsencrypt

import (
	"flag"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rancher/distros-test-framework/config"
	"github.com/rancher/distros-test-framework/internal/pkg/customflag"
	"github.com/rancher/distros-test-framework/internal/pkg/qase"
	"github.com/rancher/distros-test-framework/internal/resources"
)

var (
	qaseReport    = os.Getenv("REPORT_TO_QASE")
	kubeconfig    string
	cluster       *resources.Cluster
	flags         *customflag.FlagConfig
	cfg           *config.Env
	reportSummary string
	reportErr     error
	err           error
)

func TestMain(m *testing.M) {
	flags = &customflag.ServiceFlag
	flag.Var(&flags.Destroy, "destroy", "Destroy cluster after test")
	flag.StringVar(&flags.SecretsEncrypt.Method, "secretsEncryptMethod", "both", "method to perform secrets encryption")
	flag.Parse()

	cfg, err = config.AddEnv()
	if err != nil {
		resources.LogLevel("error", "error adding env vars: %w\n", err)
		os.Exit(1)
	}

	validateSecretsEncryptFlag()

	kubeconfig = os.Getenv("KUBE_CONFIG")
	if kubeconfig == "" {
		// gets a cluster from terraform.
		cluster = resources.ClusterConfig(cfg.Product, cfg.Module)
	} else {
		// gets a cluster from kubeconfig.
		cluster = resources.KubeConfigCluster(kubeconfig)
	}

	os.Exit(m.Run())
}

func TestSecretsEncryptionSuite(t *testing.T) {
	RegisterFailHandler(FailWithReport)
	RunSpecs(t, "Secrets Encryption Test Suite")
}

func validateSecretsEncryptFlag() {
	if cfg.Product == "k3s" {
		if !strings.Contains(os.Getenv("server_flags"), "secrets-encryption:") {
			resources.LogLevel("error", "Add secrets-encryption:true to server_flags for this test")
			os.Exit(1)
		}
	}

	if strings.Contains(os.Getenv("server_flags"), "secretbox") &&
		flags.SecretsEncrypt.Method != "rotate-keys" {
		resources.LogLevel("info", "secretbox provider is supported only with rotate-keys operation")
		flags.SecretsEncrypt.Method = "rotate-keys"
	}
}

var _ = ReportAfterSuite("Secrets Encryption Test Suite", func(report Report) {
	// Add Qase reporting capabilities.
	if strings.ToLower(qaseReport) == "true" {
		qaseClient, err := qase.AddQase()
		Expect(err).ToNot(HaveOccurred(), "error adding qase")

		qaseClient.SpecReportTestResults(qaseClient.Ctx, cluster, &report, reportSummary)
	} else {
		resources.LogLevel("info", "Qase reporting is not enabled")
	}
})

var _ = AfterSuite(func() {
	reportSummary, reportErr = resources.SummaryReportData(cluster, flags)
	if reportErr != nil {
		resources.LogLevel("error", "error getting report summary data: %v\n", reportErr)
	}

	if customflag.ServiceFlag.Destroy {
		status, err := resources.DestroyInfrastructure(cfg.Product, cfg.Module)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal("cluster destroyed"))
	}
})

func FailWithReport(message string, callerSkip ...int) {
	Fail(message, callerSkip[0]+1)
}
