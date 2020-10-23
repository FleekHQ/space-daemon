package fixtures

import (
	"os"
	"path"
	"strings"

	"github.com/FleekHQ/space-daemon/log"

	"github.com/FleekHQ/space-daemon/core/keychain"

	"github.com/99designs/keyring"

	"github.com/FleekHQ/space-daemon/app"
	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/grpc/pb"
	. "github.com/onsi/gomega"
)

type RunAppCtx struct {
	App            *app.App
	cfg            config.Config
	client         pb.SpaceApiClient
	ClientAppToken string
	ClientMnemonic string
}

func RunApp() *RunAppCtx {
	_, cfg, env := GetTestConfig()
	spaceApp := app.New(cfg, env)
	err := spaceApp.Start()

	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "space app failed to start")
	ExpectWithOffset(1, spaceApp.IsRunning).To(Equal(true), "spaceApp.IsRunning should be true")

	return &RunAppCtx{
		App:    spaceApp,
		cfg:    cfg,
		client: nil,
	}
}

// RunAppWithClientAppToken creates an instance of RunAppCtx for test but with the
// ClientAppToken already set
func RunAppWithClientAppToken(appToken string) *RunAppCtx {
	newApp := RunApp()
	newApp.ClientAppToken = appToken
	return newApp
}

func (a *RunAppCtx) Shutdown() {
	if a.App != nil {
		// shutdown app
		err := a.App.Shutdown()
		if err != nil {
			log.Error("Failed to shutdown app in test", err)
		}

		spaceStorePath := a.cfg.GetString(config.SpaceStorePath, "")
		buckdPath := a.cfg.GetString(config.BuckdPath, "")

		// delete app dir
		_ = os.RemoveAll(spaceStorePath)
		_ = os.RemoveAll(buckdPath)
	}
}

func (a *RunAppCtx) ClearMasterAppToken() {
	spaceStorePath := a.cfg.GetString(config.SpaceStorePath, "")

	// clear master token from keystore
	ucd, _ := os.UserConfigDir()
	ring, err := keyring.Open(keyring.Config{
		ServiceName:                    "space",
		KeychainTrustApplication:       true,
		KeychainAccessibleWhenUnlocked: true,
		KWalletAppID:                   "space",
		KWalletFolder:                  "space",
		WinCredPrefix:                  "space",
		LibSecretCollectionName:        "space",
		PassPrefix:                     "space",
		PassDir:                        spaceStorePath + "/kcpw",
		FileDir:                        path.Join(ucd, "space", "keyring"),
	})
	if err == nil {
		_ = ring.Remove(keychain.AppTokenStoreKey + "_" + keychain.MasterAppTokenStoreKey)
		if a.ClientAppToken != "" {
			parts := strings.Split(a.ClientAppToken, ".")
			_ = ring.Remove(keychain.AppTokenStoreKey + "_" + parts[0])
		}
	}
}
