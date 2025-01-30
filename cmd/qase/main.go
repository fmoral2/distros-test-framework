package main

import (
	"flag"
	"os"

	"github.com/rancher/distros-test-framework/config"
	"github.com/rancher/distros-test-framework/pkg/qase"
	"github.com/rancher/distros-test-framework/shared"
)

var fileName string

func main() {
	flag.StringVar(&fileName, "f", "", "path to rke2/k3s e2e tests log file")
	flag.Parse()

	if fileName == "" {
		shared.LogLevel("error", "--file flag is required")
		os.Exit(1)
	}

	cfg, err := config.AddEnv()
	if err != nil {
		shared.LogLevel("error", "error adding env vars: %w\n", err)
		os.Exit(1)
	}

	qaseClient, err := qase.AddQase()
	if err != nil {
		shared.LogLevel("error", "error adding qase: %w\n", err)
		os.Exit(1)
	}

	qaseClient.ReportTestResults(qaseClient.Ctx, nil, cfg.InstallVersion)
}
