package services

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/space/domain"
)

// Return session token for central services authenticated access
func (s *Space) GetAPISessionTokens(ctx context.Context) (domain.APISessionTokens, error) {
	return domain.APISessionTokens{
		HubToken:      "",
		ServicesToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwdWJrZXkiOiJhZTRiMmFiNjU4ZmJiNzcyMjE0MDRkNjU3YzZiNzQyZDJlZjdjNTI2YjZhNWE5YzIwMGNjZjkzZmNhMWRjZTYzIiwidXVpZCI6ImM5MDdlN2VmLTdiMzYtNGFiMS04YTU2LWY3ODhkNzUyNmEyYyIsImlhdCI6MTU5ODI4NTA0MSwiZXhwIjoxNjAwODc3MDQxfQ.dgp8UhWCLjsU0SjxXwSb3g0jEurt2jAKPaY3B_eO-qE",
	}, nil
}
