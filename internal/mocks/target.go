package mocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

// MockTarget implements target.Target interface for testing.
type MockTarget struct {
	mock.Mock
}

func (m *MockTarget) Scan() (*types.ScanResult, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ScanResult), args.Error(1)
}

func (m *MockTarget) Clean(items []types.CleanableItem) (*types.CleanResult, error) {
	args := m.Called(items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.CleanResult), args.Error(1)
}

func (m *MockTarget) Category() types.Category {
	args := m.Called()
	return args.Get(0).(types.Category)
}

func (m *MockTarget) IsAvailable() bool {
	args := m.Called()
	return args.Bool(0)
}
