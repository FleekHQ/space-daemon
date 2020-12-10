package integration_tests

import (
	"context"
	"os"
	"testing"

	"github.com/FleekHQ/space-daemon/config"

	ipfs "github.com/FleekHQ/space-daemon/core/ipfs/node"

	. "github.com/FleekHQ/space-daemon/integration_tests/helpers"

	"github.com/FleekHQ/space-daemon/integration_tests/fixtures"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	app      *fixtures.RunAppCtx
	ipfsNode *ipfs.IpfsNode
	ipfsCfg  config.Config
)

// TestIntegrationTests registers the integration test suite with ginkgo.
func TestIntegrationTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IntegrationTests Suite")
}

var _ = BeforeSuite(func() {
	// start ipfs node
	_, ipfsCfg, _ = fixtures.GetTestConfig()
	ipfsNode = ipfs.NewIpsNode(ipfsCfg)
	ipfsErrChan := make(chan error)
	go func() {
		ipfsErrChan <- ipfsNode.Start(context.TODO())
	}()

	select {
	case err := <-ipfsErrChan:
		Expect(err).NotTo(HaveOccurred(), "Error starting ipfs node for integration tests")
	case <-ipfsNode.WaitForReady():
		// ipfs node ready
	}

	app = fixtures.RunApp()
	InitializeApp(app)
})

var _ = AfterSuite(func() {
	app.Shutdown()
	app.ClearMasterAppToken()

	// shutdown ipfs
	_ = ipfsNode.Shutdown()
	_ = os.RemoveAll(ipfsCfg.GetString(config.Ipfsnodepath, ""))
})
