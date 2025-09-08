# QA Infrastructure Automation Integration

High level overview of the integration of the `qainfra-automation` repository into the `distros-test-framework` for standardized infrastructure provisioning and product deployment.

## Overview

The distros test framework now supports two infrastructure providers:

- **Legacy**: Original Terraform modules (now in `infrastructure/legacy/` which will be deprecated)
- **QA-Infra**: New standardized approach using `qainfra-automation` with OpenTofu and Ansible (in `infrastructure/qainfra/`) giving a unified configuration for both K3s and RKE2,
allowing easy switching and adding new providers/provisioners implementations.

## Architecture

### Flow Diagram

## QA Infrastructure Pipeline Flow

The qainfra provisioning pipeline follows these sequential steps:

### Pipeline Steps

1. **Environment Setup**            - Configure variables and container environment
2. **setupDirectories**             - Create temporary directories for Tofu and Ansible
3. **prepareTerraformFiles**        - Copy and configure OpenTofu files
4. **executeOpenTofuOperations**    - Provision infrastructure resources
5. **buildClusterConfig**           - Extract node information from Tofu state
6. **setupAnsibleEnvironment**      - Clone remote playbooks and setup inventory
7. **executeAnsiblePlaybook**       - Deploy K3s/RKE2 cluster
8. **addTofuOutputsToConfig**       - Add final outputs to cluster config

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Environment   │ -> │   OpenTofu       │ -> │   Node Config   │ -> │   Ansible       │
│     Setup       │    │  Infrastructure  │    │   Extraction    │    │   Deployment    │
│  (entrypoint.sh)│    │   Provisioning   │    │ (buildCluster)  │    │  (K3s/RKE2)     │
└─────────────────┘    └──────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │                       │
         v                       v                       v                       v
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ • .env vars     │    │ Resources:       │    │ • Extract IPs   │    │ • Dynamic       │
│ • SSH keys      │    │ - Instances/VMs  │    │ • Infer roles   │    │   Inventory     │
│ • vars.tfvars   │    │ - DNS records    │    │ • Populate      │    │ • Remote        │
└─────────────────┘    │ - Security Groups│    │   ServerIPs     │    │   Playbooks     │
                       │ - Load Balancer  │    │ • Build cluster │    │ • K8s Cluster   │
                       └──────────────────┘    │   config        │    │   Ready         │
                                               └─────────────────┘    └─────────────────┘
```

### Key Components

- **Remote Repository**: `qa-infra-automation` - Contains Ansible playbooks and scripts
- **Dynamic Inventory**: Uses `cloud.terraform.terraform_provider` plugin
- **Template Substitution**: `envsubst` for environment variable injection
- **Node Roles**: Supports split roles (etcd, cp, worker) and combined roles

## Configuration

### Environment Variables (.env)

It needs some new environment variables from `config/.env`:

```bash
# FRAMEWORK VARS
PROVISIONER=qainfra                                 # Provider: qainfra or legacy
QA_INFRA_MODULE=aws                                 # qainfra module to use: aws, vsphere, harvester, etc. ( for now only aws is supported )

RESOURCE_NAME={your-local-resource-name}            # Unique resource name prefix for AWS resources (e.g., "mytest123")

# SSH Configuration
SSH_USER=ec2-user                                   # SSH user to connect to instances/vms.
SSH_LOCAL_KEY_PATH="/path/to/private/aws_key.pem"   # Host path to private key

# Product Configuration
NODE_OS=sles15                                      # Node OS to use: sles15, rhel8, rhel9, etc.
CNI=calico                                          # RKE2 only for now CNI plugin: calico, canal, flannel, etc.

```

### Infrastructure Configuration

#### Unified Configuration Files

- `infrastructure/qainfra/vars.tfvars`          - OpenTofu variables (used for both K3s and RKE2)
- `infrastructure/qainfra/ansible/vars.yaml`    - Ansible variables  (used for both K3s and RKE2) ( for easier management inside distros for now it is being handled in code )

#### Key Variables in vars.tfvars

```hcl
user_id             = "distros-qa-"
aws_hostname_prefix  = "distros-qa-"

aws_region          = "us-east-2"
aws_ami             = "ami-01de4781572fa1285"
instance_type       = "t3.xlarge"
aws_ssh_user        = "ec2-user"

public_ssh_key      = "{leave-empty-to-auto-generate}"
aws_access_key      = "{leave-empty-to-use-env-var}"
aws_secret_key      = "{leave-empty-to-use-env-var}"


# Node configuration
nodes = [
  {
    count = 1 
    role  = ["etcd", "cp", "worker"]  # All-in-one master node
  },
  {
    count = 1 
    role  = ["worker"]                # Dedicated worker node
  }
]
```

## How It Works

### 1. Infrastructure Provisioning (OpenTofu)

When `INFRA_PROVIDER=qainfra`, the framework:

- After having env vars or files done ( .env, vars.tfvars )
- After your reach the entrypoint ( test_suites ) in any form ( docker,local,Jenkins... )

1. **Initializes OpenTofu** in `infrastructure/qainfra/`
2. **Creates workspace** with timestamp (e.g., `dsf-20250830221057`)
3. **Provisions resources** using remote modules from `qainfra-automation`:

### 2. Product Deployment (Ansible)

The framework then:

1. **Clones Ansible playbooks**
2. **Creates inventory file** with provisioned instance IPs:

   ```ini
   [all]
   master ansible_host=3.128.18.246
   worker-0 ansible_host=3.144.221.160

   [all:vars]
   ansible_ssh_private_key_file=/path/to/aws_key.pem
   ansible_user=ec2-user
   ansible_ssh_common_args=-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null
   ```

3. **Runs ansible-playbook** with:
   - Product-specific playbook (k3s-playbook.yml or rke2-playbook.yml)
   - Version from `INSTALL_VERSION` environment variable
   - Unified configuration from `ansible/vars.yaml`

## Troubleshooting

### Common debug and verification steps that might help

### Check Infrastructure State

```bash
# Inside container or locally with OpenTofu installed
cd infrastructure/qainfra  cd /tmp/qainfra-tofu-dsf-YYYYMMDDHHMMSS
tofu workspace list
tofu workspace select dsf-YYYYMMDDHHMMSS
tofu show

 cd /tmp/qainfra-tofu-dsf-YYYYMMDDHHMMSS
 tofu show
```

### Verify Ansible Inventory

-- todo: add commands examples .!!!!
The framework creates debug output showing:

- Node IPs discovered from OpenTofu state
- Inventory file content
- Ansible command being executed

### Common Issues

1. **SSH Key Permissions**:

   ```bash
   chmod 600 /path/to/your/private/aws_key.pem
   ```

2. **AWS Credentials**:
   - Verify AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
   - Check AWS CLI: `aws sts get-caller-identity`

3. **Docker Mount Issues**:
   - Ensure SSH_LOCAL_KEY_PATH points to a file, not directory
   - Verify the .pub file exists alongside the private key
