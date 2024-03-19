// Code generated by MockGen. DO NOT EDIT.
// Source: datasource/datasource.go

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	types "github.com/machinefi/sprout/types"
)

// MockDatasource is a mock of Datasource interface.
type MockDatasource struct {
	ctrl     *gomock.Controller
	recorder *MockDatasourceMockRecorder
}

// MockDatasourceMockRecorder is the mock recorder for MockDatasource.
type MockDatasourceMockRecorder struct {
	mock *MockDatasource
}

// NewMockDatasource creates a new mock instance.
func NewMockDatasource(ctrl *gomock.Controller) *MockDatasource {
	mock := &MockDatasource{ctrl: ctrl}
	mock.recorder = &MockDatasourceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDatasource) EXPECT() *MockDatasourceMockRecorder {
	return m.recorder
}

// Retrieve mocks base method.
func (m *MockDatasource) Retrieve(nextTaskID uint64) (*types.Task, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Retrieve", nextTaskID)
	ret0, _ := ret[0].(*types.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Retrieve indicates an expected call of Retrieve.
func (mr *MockDatasourceMockRecorder) Retrieve(nextTaskID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Retrieve", reflect.TypeOf((*MockDatasource)(nil).Retrieve), nextTaskID)
}