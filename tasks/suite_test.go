package tasks_test

import (
	"testing"

	"github.com/iter8-tools/handler/tasks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var log *logrus.Logger
var testEnv *envtest.Environment
var k8sClient client.Client

func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}

var _ = BeforeSuite(func(done Done) {
	log = tasks.GetLogger()
	log.SetOutput(GinkgoWriter)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{}
	var err error
	var restConf *rest.Config
	// create a "fake" k8s cluster and get client config in restConf
	restConf, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(restConf).ToNot(BeNil())
	// Install CRDs into the cluster
	crdPath := tasks.CompletePath("../", "testdata/crd/bases")
	_, err = envtest.InstallCRDs(restConf, envtest.CRDInstallOptions{Paths: []string{crdPath}})
	Expect(err).ToNot(HaveOccurred())

	By("initializing k8sclient")
	tasks.GetConfig = func() (*rest.Config, error) {
		return restConf, err
	}
	k8sClient, err = tasks.GetClient()
	Expect(k8sClient).ToNot(BeNil())
	Expect(err).ToNot(HaveOccurred())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
