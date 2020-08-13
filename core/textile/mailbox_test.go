package textile_test

import (
	"context"
	"testing"

	tc "github.com/FleekHQ/space-daemon/core/textile"
	"github.com/FleekHQ/space-daemon/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	uc "github.com/textileio/textile/api/users/client"
)

var (
	cfg    *mocks.Config
	st     *mocks.Store
	mockUc *mocks.UsersClient
)

type TearDown func()

func initTestMailbox(t *testing.T) (tc.Client, TearDown) {
	st = new(mocks.Store)
	client := tc.NewClient(st)
	mockUc = new(mocks.UsersClient)
	client.SetUc(mockUc)

	tearDown := func() {
		t.Log("tearDown called")
	}

	return client, tearDown
}

func TestSendMessage(t *testing.T) {
	tc, tearDown := initTestMailbox(t)
	defer tearDown()

	assert.NotNil(t, tc)

	msg := &uc.Message{
		ID: "testid",
	}

	t.Log("set mock")
	mockUc.On("SendMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(msg, nil)
	t.Log("returned from setting mock")
	rmsg, _ := tc.SendMessage(context.Background(), "recipientpubkey", "body")
	assert.NotNil(t, rmsg)
}
