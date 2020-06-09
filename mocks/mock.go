package mocks

import "github.com/stretchr/testify/mock"

// Common utils and helpers can go here

// NOTE: mocks generated using https://github.com/vektra/mockery


type Path struct {
	mock.Mock
}

func (m Path) String() string {
	args := m.Called()
	return args.String(0)
}

func (m Path) Namespace() string {
	panic("implement me")
}

func (m Path) Mutable() bool {
	panic("implement me")
}

func (m Path) IsValid() error {
	panic("implement me")
}


