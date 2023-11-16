package aws

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/rancher/distros-test-framework/config"
	"github.com/rancher/distros-test-framework/shared"
)

type Client struct {
	c   *config.Infra
	aws *ec2.EC2
}

type response struct {
	nodeId     string
	externalIp string
}

func AddAwsNode() (*Client, error) {
	env, err := shared.EnvConfig("pkg")
	if err != nil {
		return nil, shared.ReturnLogError(fmt.Sprintf("error getting env config: %v\n", err))
	}
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(env.Region)})
	if err != nil {
		return nil, shared.ReturnLogError("fatal", fmt.Sprintf("error creating AWS session: %v\n", err))
	}

	return &Client{
		c: &config.Infra{
			Ami:              env.Ami,
			Region:           env.Region,
			VolumeSize:       env.VolumeSize,
			InstanceClass:    env.InstanceClass,
			Subnets:          env.Subnets,
			AvailabilityZone: env.AvailabilityZone,
			SgId:             env.SgId,
			KeyName:          env.KeyName,
		},
		aws: ec2.New(sess),
	}, nil
}

func (a Client) CreateInstances(names ...string) (ids, ips []string, err error) {
	if len(names) == 0 {
		return nil, nil, shared.ReturnLogError("must sent a name: %s\n", names)
	}

	errChan := make(chan error, len(names))
	resChan := make(chan response, len(names))
	var wg sync.WaitGroup

	for _, n := range names {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()

			res, err := a.create(n)
			if err != nil {
				errChan <- shared.ReturnLogError("error creating instance: %w\n", err)
				return
			}

			nodeID, err := extractID(res)
			if err != nil {
				errChan <- shared.ReturnLogError("error extracting instance id: %w\n", err)
				return
			}

			externalIp, err := a.fetchIP(nodeID)
			if err != nil {
				errChan <- shared.ReturnLogError("error fetching ip: %w\n", err)
				return
			}

			resChan <- response{nodeId: nodeID, externalIp: externalIp}
		}(n)
	}
	go func() {
		wg.Wait()
		close(resChan)
		close(errChan)
	}()

	for e := range errChan {
		if e != nil {
			return nil, nil, shared.ReturnLogError("error from errChan: %w\n", e)
		}
	}

	var nodeIps, nodeIds []string
	for i := range resChan {
		nodeIds = append(nodeIds, i.nodeId)
		nodeIps = append(nodeIps, i.externalIp)
	}

	return nodeIps, nodeIds, nil
}

func (a Client) DeleteInstance(ip string) error {
	if ip == "" {
		return shared.ReturnLogError("must sent a ip: %s\n", ip)
	}

	data := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("ip-address"),
				Values: aws.StringSlice([]string{ip}),
			},
		},
	}

	res, err := a.aws.DescribeInstances(data)
	if err != nil {
		return shared.ReturnLogError("error describing instances: %w\n", err)
	}

	found := false
	for _, r := range res.Reservations {
		for _, node := range r.Instances {
			if *node.State.Name == "running" {
				found = true
				terminateInput := &ec2.TerminateInstancesInput{
					InstanceIds: aws.StringSlice([]string{*node.InstanceId}),
				}

				_, err := a.aws.TerminateInstances(terminateInput)
				if err != nil {
					return fmt.Errorf("error terminating instance: %w", err)
				}
				instanceName := "Unknown"
				if len(node.Tags) > 0 {
					instanceName = *node.Tags[0].Value
				}
				shared.LogLevel("info", fmt.Sprintf("\nTerminated instance: %s (ID: %s)", instanceName, *node.InstanceId))
			}
		}
	}
	if !found {
		return shared.ReturnLogError("no running instances found for ip: %s\n", ip)
	}

	return nil
}

func (a Client) WaitForInstanceRunning(instanceId string) error {
	input := &ec2.DescribeInstanceStatusInput{
		InstanceIds:         aws.StringSlice([]string{instanceId}),
		IncludeAllInstances: aws.Bool(true),
	}

	ticker := time.NewTicker(15 * time.Second)
	timeout := time.After(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timed out waiting for instance to be in running state and pass status checks")
		case <-ticker.C:
			statusRes, err := a.aws.DescribeInstanceStatus(input)
			if err != nil {
				return fmt.Errorf("error describing instance status: %w", err)
			}

			if len(statusRes.InstanceStatuses) == 0 {
				continue
			}

			status := statusRes.InstanceStatuses[0]
			if *status.InstanceStatus.Status == "ok" && *status.SystemStatus.Status == "ok" {
				shared.LogLevel("info", fmt.Sprintf("\nInstance %s is running and passed status checks", instanceId))
				return nil
			}
		}
	}
}

func (a Client) create(name string) (*ec2.Reservation, error) {
	volume, err := strconv.ParseInt(a.c.VolumeSize, 10, 64)
	if err != nil {
		return nil, shared.ReturnLogError("error converting volume size to int64: %w\n", err)
	}

	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(a.c.Ami),
		InstanceType: aws.String(a.c.InstanceClass),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		KeyName:      aws.String(a.c.KeyName),
		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
			{
				AssociatePublicIpAddress: aws.Bool(true),
				DeviceIndex:              aws.Int64(0),
				SubnetId:                 aws.String(a.c.Subnets),
				Groups:                   aws.StringSlice([]string{a.c.SgId}),
			},
		},
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &ec2.EbsBlockDevice{
					VolumeSize: aws.Int64(volume),
					VolumeType: aws.String("gp2"),
				},
			},
		},
		Placement: &ec2.Placement{
			AvailabilityZone: aws.String(a.c.AvailabilityZone),
		},
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(name),
					},
				},
			},
		},
	}

	shared.LogLevel("info", fmt.Sprintf("\nCreating instance: %s\n", name))

	return a.aws.RunInstances(input)
}

func (a Client) fetchIP(nodeID string) (string, error) {
	waitErr := a.WaitForInstanceRunning(nodeID)
	if waitErr != nil {
		return "", shared.ReturnLogError("error waiting for instance to be in running state: %w\n", waitErr)
	}

	id := &ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{nodeID}),
	}
	result, err := a.aws.DescribeInstances(id)
	if err != nil {
		return "", shared.ReturnLogError("error describing instances: %w\n", err)
	}

	for _, r := range result.Reservations {
		for _, i := range r.Instances {
			if i.PublicIpAddress != nil {
				return *i.PublicIpAddress, nil
			}
		}
	}

	return "", shared.ReturnLogError("no public ip found for instance: %s\n", nodeID)
}

func extractID(reservation *ec2.Reservation) (string, error) {
	if len(reservation.Instances) == 0 || reservation.Instances[0].InstanceId == nil {
		return "", fmt.Errorf("no instance ID found")
	}

	return *reservation.Instances[0].InstanceId, nil
}
