package fixtures

import (
	"context"
	"fmt"

	"google.golang.org/grpc/metadata"

	. "github.com/onsi/gomega"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/grpc/pb"
	"google.golang.org/grpc"
)

var DefaultBucket = "personal"
var MirrorBucket = "personal_mirror"

func (a *RunAppCtx) Client() pb.SpaceApiClient {
	if a.client != nil {
		return a.client
	}

	conn, err := DialGrpcClient(fmt.Sprintf(":%d", a.cfg.GetInt(config.SpaceServerPort, 9999)), a)
	Expect(err).NotTo(HaveOccurred())
	a.client = pb.NewSpaceApiClient(conn)

	return a.client
}

func DialGrpcClient(targetAddr string, a *RunAppCtx) (*grpc.ClientConn, error) {
	return grpc.Dial(
		targetAddr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithUnaryInterceptor(func(
			ctx context.Context,
			method string,
			req, reply interface{},
			cc *grpc.ClientConn,
			invoker grpc.UnaryInvoker,
			opts ...grpc.CallOption,
		) error {
			if a.ClientAppToken != "" {
				md := metadata.New(map[string]string{"authorization": "AppToken " + a.ClientAppToken})
				ctx = metadata.NewOutgoingContext(ctx, md)
			}

			return invoker(ctx, method, req, reply, cc, opts...)
		}),
		grpc.WithStreamInterceptor(func(
			ctx context.Context,
			desc *grpc.StreamDesc,
			cc *grpc.ClientConn,
			method string,
			streamer grpc.Streamer,
			opts ...grpc.CallOption,
		) (grpc.ClientStream, error) {
			if a.ClientAppToken != "" {
				md := metadata.New(map[string]string{"authorization": "AppToken " + a.ClientAppToken})
				ctx = metadata.NewOutgoingContext(ctx, md)
			}

			return streamer(ctx, desc, cc, method, opts...)
		}),
	)
}
