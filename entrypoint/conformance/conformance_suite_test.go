package sonobuoyconformance

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/rancher/distros-test-framework/config"
	"github.com/rancher/distros-test-framework/internal/pkg/customflag"
	"github.com/rancher/distros-test-framework/internal/pkg/qase"
	"github.com/rancher/distros-test-framework/internal/resources"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	flag.StringVar(&customflag.ServiceFlag.External.SonobuoyVersion, "sonobuoyVersion", "0.57.3", "Sonobuoy binary version")
	flag.Var(&customflag.ServiceFlag.Destroy, "destroy", "Destroy cluster after test")
	flag.Parse()

	cfg, err = config.AddEnv()
	if err != nil {
		resources.LogLevel("error", "error adding env vars: %w\n", err)
		os.Exit(1)
	}

	verifyClusterNodes()

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

func TestConformance(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Run Conformance Suite")
}

var _ = ReportAfterSuite("Conformance Suite", func(report Report) {
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

func verifyClusterNodes() {
	resources.LogLevel("info", "verying cluster configuration matches minimum requirements for conformance tests")
	s, serverErr := strconv.Atoi(os.Getenv("no_of_server_nodes"))
	w, workerErr := strconv.Atoi(os.Getenv("no_of_worker_nodes"))

	if serverErr != nil || workerErr != nil {
		resources.LogLevel("error", "Failed to convert node counts to integers: %v, %v", serverErr, workerErr)
		os.Exit(1)
	}

	if s < 1 && w < 1 {
		resources.LogLevel("error", "%s", "cluster must at least consist of 1 server and 1 agent")
		os.Exit(1)
	}
}
