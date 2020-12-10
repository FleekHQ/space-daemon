package helpers

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
	"github.com/FleekHQ/space-daemon/integration_tests/fixtures"
	. "github.com/onsi/gomega"
)

func InitializeApp(app *fixtures.RunAppCtx) {
	if app.ClientAppToken == "" {
		res, err := app.Client().InitializeMasterAppToken(
			context.Background(),
			&pb.InitializeMasterAppTokenRequest{},
		)

		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, res.AppToken).NotTo(BeEmpty())
		app.ClientAppToken = res.AppToken
	}

	if app.ClientMnemonic == "" {
		kpRes, err := app.Client().GenerateKeyPair(context.Background(), &pb.GenerateKeyPairRequest{})
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		app.ClientMnemonic = kpRes.Mnemonic
	}
}
