package migrations_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshuaa "github.com/cloudfoundry/bosh-cli/uaa"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type Config struct {
	UAAURL string `json:"uaa_url"`

	DirectorClient       string `json:"director_client"`
	DirectorClientSecret string `json:"director_client_secret"`
	DirectorCAPath       string `json:"director_ca_path"`
	DirectorURL          string `json:"director_url"`

	DeploymentName string `json:"deployment_name"`

	SingleNodeKuboManifestPath string `json:"single_node_kubo_manifest_path"`
	SingleNodeCFCRManifestPath string `json:"single_node_cfcr_manifest_path"`
}

var _ = Describe("Migrate From Kubo ETCD", func() {
	var (
		cfg Config

		expectedKey   string
		expectedValue string
	)

	BeforeEach(func() {
		configPath := os.Getenv("CONFIG_FILE")
		Expect(configPath).NotTo(BeEmpty(), "CONFIG_FILE must be set to run the acceptance test suite.")

		configFile, err := os.Open(configPath)
		Expect(err).NotTo(HaveOccurred())

		cfg = Config{}
		decoder := json.NewDecoder(configFile)
		Expect(decoder.Decode(&cfg)).To(Succeed())

		expectedKey = uuid.NewV4().String()
		expectedValue = uuid.NewV4().String()
	})

	It("successfully migrates from a single node kubo etcd to a new single node deployment", func() {
		directorCA, err := ioutil.ReadFile(cfg.DirectorCAPath)
		Expect(err).NotTo(HaveOccurred())

		logger := boshlog.NewLogger(boshlog.LevelError)
		uaaFactory := boshuaa.NewFactory(logger)

		uaaCfg, err := boshuaa.NewConfigFromURL(cfg.UAAURL)
		Expect(err).NotTo(HaveOccurred())

		uaaCfg.Client = cfg.DirectorClient
		uaaCfg.ClientSecret = cfg.DirectorClientSecret
		uaaCfg.CACert = string(directorCA)

		uaa, err := uaaFactory.New(uaaCfg)
		Expect(err).NotTo(HaveOccurred())

		directorFactory := boshdir.NewFactory(logger)

		directorCfg, err := boshdir.NewConfigFromURL(cfg.DirectorURL)
		Expect(err).NotTo(HaveOccurred())

		directorCfg.CACert = string(directorCA)
		directorCfg.TokenFunc = boshuaa.NewClientTokenSession(uaa).TokenFunc

		boshDirector, err := directorFactory.New(directorCfg, boshdir.NewNoopTaskReporter(), boshdir.NewNoopFileReporter())
		Expect(err).NotTo(HaveOccurred())
		Expect(boshDirector).NotTo(BeNil())

		deployment, err := boshDirector.FindDeployment(cfg.DeploymentName)
		Expect(err).NotTo(HaveOccurred())

		singleNodeKuboManifest, err := ioutil.ReadFile(cfg.SingleNodeKuboManifestPath)
		Expect(err).NotTo(HaveOccurred())

		Expect(deployment.Update(singleNodeKuboManifest, boshdir.UpdateOpts{})).To(Succeed())

		host, username, privateKey := getSSHCreds(cfg.DeploymentName, "etcd", "0", boshDirector)
		result := runSSHCommand(host, 22, username, privateKey, fmt.Sprintf("ETCDCTL_API=3 /var/vcap/jobs/etcd/bin/etcdctl put %s %s", expectedKey, expectedValue))
		Expect(result).To(ContainSubstring("OK"))
		result = runSSHCommand(host, 22, username, privateKey, fmt.Sprintf("ETCDCTL_API=3 /var/vcap/jobs/etcd/bin/etcdctl get %s", expectedKey))
		Expect(result).To(ContainSubstring(expectedValue))
		cleanupSSHCreds(cfg.DeploymentName, "etcd", "0", boshDirector)

		singleNodeCFCRManifest, err := ioutil.ReadFile(cfg.SingleNodeCFCRManifestPath)
		Expect(err).NotTo(HaveOccurred())

		Expect(deployment.Update(singleNodeCFCRManifest, boshdir.UpdateOpts{})).To(Succeed())

		host, username, privateKey = getSSHCreds(cfg.DeploymentName, "etcd", "0", boshDirector)
		result = runSSHCommand(host, 22, username, privateKey, fmt.Sprintf("ETCDCTL_API=3 /var/vcap/jobs/etcd/bin/etcdctl get %s", expectedKey))
		Expect(result).To(ContainSubstring(expectedValue))
		cleanupSSHCreds(cfg.DeploymentName, "etcd", "0", boshDirector)
	})
})
