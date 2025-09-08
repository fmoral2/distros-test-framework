package qainfra

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/rancher/distros-test-framework/internal/provisioning/driver"
	"github.com/rancher/distros-test-framework/internal/resources"
)

type ProvisioningStep func(*driver.InfraConfig) error

var (
	qaCfg                  *driver.InfraConfig
	qaOnce                 sync.Once
	defaultContainerKeyDir = "/go/src/github.com/rancher/distros-test-framework"
)

func addQAInfraEnv(infraCfg *driver.InfraConfig) *driver.InfraConfig {
	qaOnce.Do(func() {
		qaCfg = loadQAInfra(infraCfg)
		if qaCfg == nil {
			resources.LogLevel("error", "error loading qainfra cluster config")
			os.Exit(1)
		}
	})

	return qaCfg
}

// loadQAInfra creates a configuration for qainfra driver.
func loadQAInfra(i *driver.InfraConfig) *driver.InfraConfig {
	workspace := fmt.Sprintf("dsf-%s", time.Now().Format("20060102150405"))
	uniqueID := time.Now().Format("01021504")

	// Determine the root directory to proceed with necessary file operations.
	var defaultKeyDir string
	_, callerFilePath, _, _ := runtime.Caller(0)
	defaultKeyDir = filepath.Join(filepath.Dir(callerFilePath), "..", "..")

	var nodeSource, tempDir string
	isContainer := false
	nodeSource = filepath.Join("/tmp", fmt.Sprintf("qainfra-tofu-%s", workspace))
	tempDir = "/tmp/qainfra-ansible"

	if resources.IsRunningInContainer() {
		isContainer = true
		defaultKeyDir = defaultContainerKeyDir
		nodeSource = filepath.Join("/tmp", fmt.Sprintf("qainfra-tofu-%s", workspace))
		tempDir = "/tmp/qainfra-ansible"
	}

	var ansiblePath string
	switch strings.ToLower(i.Product) {
	case "k3s":
		ansiblePath = "ansible/k3s/default"
	case "rke2":
		ansiblePath = "ansible/rke2/default"
	default:
		resources.LogLevel("error", "unsupported product: %s", i.Product)
	}

	ansibleDir := filepath.Join(tempDir, ansiblePath)
	sshUser := i.Cluster.SSH.User
	sshPrivKeyPath := i.Cluster.SSH.PrivKeyPath

	// Adjust SSH key path for container environment.
	if isContainer && sshPrivKeyPath != "" {
		// In container, SSH keys are typically mounted to a standard location
		sshPrivKeyPath = defaultContainerKeyDir + "/config/.ssh/aws_key.pem"
		resources.LogLevel("info", "Container detected: Using container SSH key path: %s", sshPrivKeyPath)
	}

	tmpSSHPubPath, keyErr := prepareSSHKeys(sshPrivKeyPath)
	if keyErr != nil {
		resources.LogLevel("error", "error preparing SSH keys: %v", keyErr)
		return nil
	}

	i = &driver.InfraConfig{
		Provisioner:    i.Provisioner,
		ResourceName:   i.ResourceName,
		Product:        strings.ToLower(i.Product),
		Module:         i.Module,
		InstallVersion: i.InstallVersion,
		QAInfraModule:  i.QAInfraModule,
		NodeOS:         i.NodeOS,
		CNI:            i.CNI,

		Cluster: &driver.Cluster{
			SSH: driver.SSHConfig{
				PrivKeyPath: sshPrivKeyPath,
				User:        sshUser,
				PubKeyPath:  tmpSSHPubPath,
			},
			Config: driver.Config{
				Arch:        i.Cluster.Config.Arch,
				ServerFlags: i.Cluster.Config.ServerFlags,
				WorkerFlags: i.Cluster.Config.WorkerFlags,
				Product:     strings.ToLower(i.Product),
				Version:     i.InstallVersion,
			},
		},
		InfraProvisioner: &driver.InfraProvisionerConfig{
			Workspace:      workspace,
			UniqueID:       uniqueID,
			QAInfraModule:  i.QAInfraModule,
			IsContainer:    isContainer,
			RootDir:        defaultKeyDir,
			TFNodeSource:   nodeSource,
			TempDir:        tempDir,
			KubeconfigPath: filepath.Join(ansibleDir, "kubeconfig.yaml"),

			Inventory: driver.Inventory{
				Path: filepath.Join(ansibleDir, "inventory.yml"),
			},
			Ansible: driver.Ansible{
				Dir:  ansibleDir,
				Path: ansiblePath,
			},
			Terraform: driver.Terraform{
				TFVarsPath: filepath.Join(nodeSource, "vars.tfvars"),
				MainTfPath: filepath.Join(nodeSource, "main.tf"),
			},

			OpenTofuOutputs: driver.OpenTofuOutputs{},
			AirgapSetup:     false,
			ProxySetup:      false,
		},
	}

	resources.LogLevel("debug", "Created QA infra configuration:\n%+v", i)

	return i
}

func loadQAInfraTFVars(clusterConfig *driver.Cluster, airgapSetup, proxySetup bool, nodeSource string) error {
	varsFile := filepath.Join(nodeSource, "vars.tfvars")

	if err := loadVarsFromFile(
		clusterConfig,
		airgapSetup,
		proxySetup,
		varsFile,
	); err != nil {
		return fmt.Errorf("failed to load qainfra vars: %w", err)
	}

	clusterConfig.Aws.AccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	clusterConfig.Aws.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")

	resources.LogLevel("info", "Cluster configuration loaded from vars.tfvars\n%+v", clusterConfig)

	return nil
}

// loadVarsFromFile loads variables from a vars.tfvars file into cluster config.
func loadVarsFromFile(clusterConfig *driver.Cluster, airgapSetup, proxySetup bool, filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read vars file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

			switch key {
			case "aws_region":
				clusterConfig.Aws.Region = value
			case "aws_subnet", "subnets":
				clusterConfig.Aws.Subnets = value
			case "availability_zone":
				clusterConfig.Aws.AvailabilityZone = value
			case "sg_id":
				clusterConfig.Aws.SgId = value
			case "vpc_id":
				clusterConfig.Aws.VPCID = value
			case "public_ssh_key":
				// this value came from runtime not from the vars.tfvars file.
			case "aws_ssh_user":
				clusterConfig.SSH.User = value
			case "aws_ami":
				clusterConfig.Aws.EC2.Ami = value
			case "volume_size":
				clusterConfig.Aws.EC2.VolumeSize = value
			case "ec2_instance_class", "instance_type":
				clusterConfig.Aws.EC2.InstanceClass = value
			case "aws_volume_type":
				clusterConfig.Aws.EC2.VolumeType = value
			case "aws_volume_size":
				clusterConfig.Aws.EC2.VolumeSize = value
			case "airgap_setup":
				// set as false for now, TODO: implement airgap support
				airgapSetup = false
			case "proxy_setup":
				// set as false for now, TODO: implement proxy support
				proxySetup = false
			}
		}
	}

	return nil
}

func setupDirectories(config *driver.InfraConfig) error {
	directories := []string{
		config.InfraProvisioner.TFNodeSource,
		config.InfraProvisioner.TempDir,
		config.InfraProvisioner.TempDir,
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// buildClusterConfig todo: adjust to add missing data for all tests run.
func buildClusterConfig(config *driver.InfraConfig) error {
	if config.Cluster.Config.DataStore == "" {
		config.Cluster.Config.DataStore = "etcd"
	}

	nodes, err := extractNodesFromState(config)
	if err != nil {
		return fmt.Errorf("failed to extract nodes from state: %w", err)
	}

	var serverIPs, agentIPs []string
	for _, node := range nodes {
		if isServerRole(node.role) {
			serverIPs = append(serverIPs, node.publicIP)
		} else {
			agentIPs = append(agentIPs, node.publicIP)
		}
	}

	config.Cluster.ServerIPs = serverIPs
	config.Cluster.AgentIPs = agentIPs
	config.Cluster.NumServers = len(serverIPs)
	config.Cluster.NumAgents = len(agentIPs)
	config.Cluster.Status = "cluster created"

	resources.LogLevel("info", "Built cluster config: %d servers, %d agents", len(serverIPs), len(agentIPs))
	resources.LogLevel("debug", "Server IPs: %v", serverIPs)
	resources.LogLevel("debug", "Agent IPs: %v", agentIPs)

	return nil
}

func isServerRole(role string) bool {
	return strings.Contains(role, "etcd") || strings.Contains(role, "cp")
}

// prepareSSHKeys copies the mounted PEM to /tmp and writes /tmp/key.pub
func prepareSSHKeys(envKeyPath string) (pubPath string, err error) {
	tmpPvtKeyPath := "/tmp/aws_key.pem"
	tmpPubPath := "/tmp/key.pub"

	if copyErr := resources.CopyFileContents(envKeyPath, tmpPvtKeyPath, 0o600); copyErr != nil {
		return "", fmt.Errorf("copy private key: %w", copyErr)
	}

	pubLine, err := authorizedKeyFromPrivateFile(tmpPvtKeyPath)
	if err != nil {
		return "", fmt.Errorf("derive public key: %w", err)
	}

	if writeErr := os.WriteFile(tmpPubPath, []byte(pubLine+"\n"), 0o644); writeErr != nil {
		return "", fmt.Errorf("write pub file: %w", writeErr)
	}

	return tmpPubPath, nil
}

// authorizedKeyFromPrivateFile parses RSA/ED25519/ECDSA PEM (unencrypted).
func authorizedKeyFromPrivateFile(privateKeyPath string) (string, error) {
	key, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("read private key %s: %w", privateKeyPath, err)
	}

	block, _ := pem.Decode(key)
	if block == nil {
		return "", fmt.Errorf("no PEM block found in %s", privateKeyPath)
	}

	signer, signErr := ssh.ParsePrivateKey(key)
	if signErr != nil {
		return "", fmt.Errorf("parse private key %s: %w", privateKeyPath, signErr)
	}

	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey()))), nil
}

// runCmdWithTimeout executes a command with a specific timeout
func runCmdWithTimeout(dir string, timeout time.Duration, name string, args ...string) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	resources.LogLevel("info", "Running with %v timeout: %s %v (in %s)", timeout, name, args, dir)
	err := cmd.Run()

	if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("command timed out after %v: %s %v", timeout, name, args)
	}

	return err
}
