package migrations_test

import (
	"bytes"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

func TestMigrations(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Migrations Suite")
}

func runSSHCommand(server string, port int, username string, privateKey string, command string) string {
	parsedPrivateKey, err := ssh.ParsePrivateKey([]byte(privateKey))
	Expect(err).NotTo(HaveOccurred())

	config := &ssh.ClientConfig{
		User:            username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(parsedPrivateKey),
		},
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", server, port), config)
	Expect(err).NotTo(HaveOccurred())
	defer conn.Close()

	session, err := conn.NewSession()
	Expect(err).NotTo(HaveOccurred())
	defer session.Close()

	var output bytes.Buffer

	session.Stdout = &output

	err = session.Run(command)
	Expect(err).NotTo(HaveOccurred())

	return output.String()
}

func getSSHCreds(deploymentName, instanceGroupName, index string, director boshdir.Director) (string, string, string) {
	deployment, err := director.FindDeployment(deploymentName)
	Expect(err).NotTo(HaveOccurred())

	sshOpts, privateKey, err := boshdir.NewSSHOpts(boshuuid.NewGenerator())
	Expect(err).NotTo(HaveOccurred())

	slug := boshdir.NewAllOrInstanceGroupOrInstanceSlug(instanceGroupName, index)
	sshResult, err := deployment.SetUpSSH(slug, sshOpts)
	Expect(err).NotTo(HaveOccurred())

	return sshResult.Hosts[0].Host, sshOpts.Username, privateKey
}

func cleanupSSHCreds(deploymentName, instanceGroupName, index string, director boshdir.Director) {
	deployment, err := director.FindDeployment(deploymentName)
	Expect(err).NotTo(HaveOccurred())

	sshOpts, _, err := boshdir.NewSSHOpts(boshuuid.NewGenerator())
	Expect(err).NotTo(HaveOccurred())

	slug := boshdir.NewAllOrInstanceGroupOrInstanceSlug(instanceGroupName, index)
	Expect(deployment.CleanUpSSH(slug, sshOpts)).To(Succeed())
}
