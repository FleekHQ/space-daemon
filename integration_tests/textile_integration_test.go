package integration_tests

/// Commenting this out because it does not have proper setup and shutdown implemented
/// And moreover, the TestAddAndRetreiveFile case below has been handled in the app_test
// TODO: Delete this file
//
//import (
//	"bytes"
//	"context"
//	"os"
//	"testing"
//
//	"github.com/FleekHQ/space-daemon/core/store"
//	"github.com/FleekHQ/space-daemon/core/textile"
//	"github.com/stretchr/testify/assert"
//)
//
//var (
//	textileClient textile.Client
//	ctx           context.Context
//)
//
//func TestMain(m *testing.M) {
//	ctx = context.Background()
//
//	_, cfg, _ := GetTestConfig()
//	st := store.New(
//		store.WithPath("/tmp"),
//	)
//	if err := st.Open(); err != nil {
//		return
//	}
//
//	// setup local buckets
//	buckd := textile.NewBuckd(cfg)
//	err := buckd.Start(ctx)
//	if err != nil {
//		return
//	}
//
//	// setup textile client
//	textileClient = textile.NewClient(st)
//	go func() {
//		textileClient.StartAndBootstrap(ctx, cfg)
//	}()
//	<-textileClient.WaitForReady()
//
//	// run
//	exitCode := m.Run()
//
//	if err = textileClient.Shutdown(); err != nil {
//	}
//
//	os.Exit(exitCode)
//}
//
//func TestAddAndRetreiveFile(t *testing.T) {
//	b, _ := textileClient.CreateBucket(ctx, "testbucket1")
//
//	assert.NotNil(t, b)
//
//	f, _ := os.Open("./test_files/file1")
//	p1, p2, _ := b.UploadFile(ctx, "file1", f)
//	assert.NotNil(t, p1)
//	assert.NotNil(t, p2)
//
//	var bb bytes.Buffer
//	b.GetFile(ctx, "file1", &bb)
//	assert.Equal(t, bb.String(), "test data")
//}
