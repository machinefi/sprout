// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/libp2p/go-libp2p/core/host (interfaces: Host)

// Package mock_host is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	connmgr "github.com/libp2p/go-libp2p/core/connmgr"
	event "github.com/libp2p/go-libp2p/core/event"
	network "github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"
	peerstore "github.com/libp2p/go-libp2p/core/peerstore"
	protocol "github.com/libp2p/go-libp2p/core/protocol"
	multiaddr "github.com/multiformats/go-multiaddr"
)

// MockHost is a mock of Host interface.
type MockHost struct {
	ctrl     *gomock.Controller
	recorder *MockHostMockRecorder
}

// MockHostMockRecorder is the mock recorder for MockHost.
type MockHostMockRecorder struct {
	mock *MockHost
}

// NewMockHost creates a new mock instance.
func NewMockHost(ctrl *gomock.Controller) *MockHost {
	mock := &MockHost{ctrl: ctrl}
	mock.recorder = &MockHostMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockHost) EXPECT() *MockHostMockRecorder {
	return m.recorder
}

// Addrs mocks base method.
func (m *MockHost) Addrs() []multiaddr.Multiaddr {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Addrs")
	ret0, _ := ret[0].([]multiaddr.Multiaddr)
	return ret0
}

// Addrs indicates an expected call of Addrs.
func (mr *MockHostMockRecorder) Addrs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Addrs", reflect.TypeOf((*MockHost)(nil).Addrs))
}

// Close mocks base method.
func (m *MockHost) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockHostMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockHost)(nil).Close))
}

// ConnManager mocks base method.
func (m *MockHost) ConnManager() connmgr.ConnManager {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnManager")
	ret0, _ := ret[0].(connmgr.ConnManager)
	return ret0
}

// ConnManager indicates an expected call of ConnManager.
func (mr *MockHostMockRecorder) ConnManager() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnManager", reflect.TypeOf((*MockHost)(nil).ConnManager))
}

// Connect mocks base method.
func (m *MockHost) Connect(arg0 context.Context, arg1 peer.AddrInfo) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Connect", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Connect indicates an expected call of Connect.
func (mr *MockHostMockRecorder) Connect(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Connect", reflect.TypeOf((*MockHost)(nil).Connect), arg0, arg1)
}

// EventBus mocks base method.
func (m *MockHost) EventBus() event.Bus {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EventBus")
	ret0, _ := ret[0].(event.Bus)
	return ret0
}

// EventBus indicates an expected call of EventBus.
func (mr *MockHostMockRecorder) EventBus() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EventBus", reflect.TypeOf((*MockHost)(nil).EventBus))
}

// ID mocks base method.
func (m *MockHost) ID() peer.ID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(peer.ID)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *MockHostMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockHost)(nil).ID))
}

// Mux mocks base method.
func (m *MockHost) Mux() protocol.Switch {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Mux")
	ret0, _ := ret[0].(protocol.Switch)
	return ret0
}

// Mux indicates an expected call of Mux.
func (mr *MockHostMockRecorder) Mux() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Mux", reflect.TypeOf((*MockHost)(nil).Mux))
}

// Network mocks base method.
func (m *MockHost) Network() network.Network {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Network")
	ret0, _ := ret[0].(network.Network)
	return ret0
}

// Network indicates an expected call of Network.
func (mr *MockHostMockRecorder) Network() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Network", reflect.TypeOf((*MockHost)(nil).Network))
}

// NewStream mocks base method.
func (m *MockHost) NewStream(arg0 context.Context, arg1 peer.ID, arg2 ...protocol.ID) (network.Stream, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "NewStream", varargs...)
	ret0, _ := ret[0].(network.Stream)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewStream indicates an expected call of NewStream.
func (mr *MockHostMockRecorder) NewStream(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewStream", reflect.TypeOf((*MockHost)(nil).NewStream), varargs...)
}

// Peerstore mocks base method.
func (m *MockHost) Peerstore() peerstore.Peerstore {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Peerstore")
	ret0, _ := ret[0].(peerstore.Peerstore)
	return ret0
}

// Peerstore indicates an expected call of Peerstore.
func (mr *MockHostMockRecorder) Peerstore() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Peerstore", reflect.TypeOf((*MockHost)(nil).Peerstore))
}

// RemoveStreamHandler mocks base method.
func (m *MockHost) RemoveStreamHandler(arg0 protocol.ID) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RemoveStreamHandler", arg0)
}

// RemoveStreamHandler indicates an expected call of RemoveStreamHandler.
func (mr *MockHostMockRecorder) RemoveStreamHandler(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveStreamHandler", reflect.TypeOf((*MockHost)(nil).RemoveStreamHandler), arg0)
}

// SetStreamHandler mocks base method.
func (m *MockHost) SetStreamHandler(arg0 protocol.ID, arg1 network.StreamHandler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetStreamHandler", arg0, arg1)
}

// SetStreamHandler indicates an expected call of SetStreamHandler.
func (mr *MockHostMockRecorder) SetStreamHandler(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetStreamHandler", reflect.TypeOf((*MockHost)(nil).SetStreamHandler), arg0, arg1)
}

// SetStreamHandlerMatch mocks base method.
func (m *MockHost) SetStreamHandlerMatch(arg0 protocol.ID, arg1 func(protocol.ID) bool, arg2 network.StreamHandler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetStreamHandlerMatch", arg0, arg1, arg2)
}

// SetStreamHandlerMatch indicates an expected call of SetStreamHandlerMatch.
func (mr *MockHostMockRecorder) SetStreamHandlerMatch(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetStreamHandlerMatch", reflect.TypeOf((*MockHost)(nil).SetStreamHandlerMatch), arg0, arg1, arg2)
}
