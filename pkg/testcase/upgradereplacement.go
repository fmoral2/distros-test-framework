package testcase

//
// import (
// 	"bytes"
// 	"fmt"
//
// 	"github.com/aws/aws-sdk-go/aws"
// 	"github.com/aws/aws-sdk-go/aws/session"
// 	"github.com/aws/aws-sdk-go/service/ec2"
// 	"github.com/rancher/distros-test-framework/shared"
// )
//
// // TODO://  cluster set
// // TestUpgradeReplacement after cluster is set and ok
// func TestUpgradeReplacement(deployWorkload bool) {
// 	ips := shared.FetchNodeExternalIP()
// 	fmt.Println(ips)
//
// 	configProd, _ := shared.GetProduct()
//
// 	t := token()
// 	err := deleteServer(ips[0], "us-west-2")
// 	if err != nil {
// 		fmt.Println(err)
// 	}
//
// 	err = joinServerInstallProd(ips[0], t, "version", flag.Upgrade, configProd)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
//
// 	fmt.Println("done")
//
// }
//
// func token() string {
//
// 	return ""
// }
//
// func deleteServer(instanceID, region string) error {
// 	s, err := session.NewSession(&aws.Config{
// 		Region: aws.String(region),
// 	})
// 	if err != nil {
// 		return err
// 	}
//
// 	svc := ec2.New(s)
//
// 	input := &ec2.TerminateInstancesInput{
// 		InstanceIds: []*string{
// 			aws.String(instanceID),
// 		},
// 	}
// 	i, err := svc.TerminateInstances(input)
// 	if err != nil {
// 		return err
// 	}
//
// 	for _, instance := range i.TerminatingInstances {
//
// 		fmt.Println("Deleting instance", *instance.CurrentState.Name)
// 	}
//
// 	return nil
// }
//
// func joinServerInstallProd(serverIP, token, installType, version, prod string) error {
//
// 	c, err := shared.ConfigureSSH(serverIP)
// 	if err != nil {
// 		return err
// 	}
//
// 	var command string
//
// 	switch prod {
// 	case "rke2":
// 		if installType == "v" {
// 			command = fmt.Sprintf("curl -sfL https://get.rke2.io | sh -s -- --version %s", version)
// 		} else {
// 			command = fmt.Sprintf("curl -sfL https://get.rke2.io | sh -s -- --version latest")
// 		}
// 	case "k3s":
//
// 	}
//
// 	s, err := c.NewSession()
// 	if err != nil {
// 		return err
// 	}
// 	defer s.Close()
//
// 	var b bytes.Buffer
// 	s.Stdout = &b
// 	if err := s.Run(command); err != nil {
// 		return err
// 	}
// 	fmt.Println(b.String())
//
// 	return nil
// }
//
// func deleteAgent() {
//
// }
//
// func joinAgent() {
//
// }
//
// // delete 1 server
//
// // join one new with latest version
//
// // delete next server
//
// // join one new with latest version
//
// // delete agent if exists
//
// // join one new with latest version
//
// // check nodes in correct version
//
// // check pods
//
// // validations
//
// func main() {
// 	c, _ := shared.ConfigureSSH("ip")
//
// 	fmt.Println(c)
// }
