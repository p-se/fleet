// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/rancher/wrangler/pkg/generated/controllers/core/v1 (interfaces: SecretController)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	generic "github.com/rancher/wrangler/pkg/generic"
	v10 "k8s.io/api/core/v1"
	v11 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// MockSecretController is a mock of SecretController interface.
type MockSecretController struct {
	ctrl     *gomock.Controller
	recorder *MockSecretControllerMockRecorder
}

// MockSecretControllerMockRecorder is the mock recorder for MockSecretController.
type MockSecretControllerMockRecorder struct {
	mock *MockSecretController
}

// NewMockSecretController creates a new mock instance.
func NewMockSecretController(ctrl *gomock.Controller) *MockSecretController {
	mock := &MockSecretController{ctrl: ctrl}
	mock.recorder = &MockSecretControllerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSecretController) EXPECT() *MockSecretControllerMockRecorder {
	return m.recorder
}

// AddGenericHandler mocks base method.
func (m *MockSecretController) AddGenericHandler(arg0 context.Context, arg1 string, arg2 generic.Handler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddGenericHandler", arg0, arg1, arg2)
}

// AddGenericHandler indicates an expected call of AddGenericHandler.
func (mr *MockSecretControllerMockRecorder) AddGenericHandler(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddGenericHandler", reflect.TypeOf((*MockSecretController)(nil).AddGenericHandler), arg0, arg1, arg2)
}

// AddGenericRemoveHandler mocks base method.
func (m *MockSecretController) AddGenericRemoveHandler(arg0 context.Context, arg1 string, arg2 generic.Handler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddGenericRemoveHandler", arg0, arg1, arg2)
}

// AddGenericRemoveHandler indicates an expected call of AddGenericRemoveHandler.
func (mr *MockSecretControllerMockRecorder) AddGenericRemoveHandler(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddGenericRemoveHandler", reflect.TypeOf((*MockSecretController)(nil).AddGenericRemoveHandler), arg0, arg1, arg2)
}

// Cache mocks base method.
func (m *MockSecretController) Cache() v1.SecretCache {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cache")
	ret0, _ := ret[0].(v1.SecretCache)
	return ret0
}

// Cache indicates an expected call of Cache.
func (mr *MockSecretControllerMockRecorder) Cache() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cache", reflect.TypeOf((*MockSecretController)(nil).Cache))
}

// Create mocks base method.
func (m *MockSecretController) Create(arg0 *v10.Secret) (*v10.Secret, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0)
	ret0, _ := ret[0].(*v10.Secret)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockSecretControllerMockRecorder) Create(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockSecretController)(nil).Create), arg0)
}

// Delete mocks base method.
func (m *MockSecretController) Delete(arg0, arg1 string, arg2 *v11.DeleteOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockSecretControllerMockRecorder) Delete(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockSecretController)(nil).Delete), arg0, arg1, arg2)
}

// Enqueue mocks base method.
func (m *MockSecretController) Enqueue(arg0, arg1 string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Enqueue", arg0, arg1)
}

// Enqueue indicates an expected call of Enqueue.
func (mr *MockSecretControllerMockRecorder) Enqueue(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enqueue", reflect.TypeOf((*MockSecretController)(nil).Enqueue), arg0, arg1)
}

// EnqueueAfter mocks base method.
func (m *MockSecretController) EnqueueAfter(arg0, arg1 string, arg2 time.Duration) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "EnqueueAfter", arg0, arg1, arg2)
}

// EnqueueAfter indicates an expected call of EnqueueAfter.
func (mr *MockSecretControllerMockRecorder) EnqueueAfter(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnqueueAfter", reflect.TypeOf((*MockSecretController)(nil).EnqueueAfter), arg0, arg1, arg2)
}

// Get mocks base method.
func (m *MockSecretController) Get(arg0, arg1 string, arg2 v11.GetOptions) (*v10.Secret, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1, arg2)
	ret0, _ := ret[0].(*v10.Secret)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockSecretControllerMockRecorder) Get(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockSecretController)(nil).Get), arg0, arg1, arg2)
}

// GroupVersionKind mocks base method.
func (m *MockSecretController) GroupVersionKind() schema.GroupVersionKind {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GroupVersionKind")
	ret0, _ := ret[0].(schema.GroupVersionKind)
	return ret0
}

// GroupVersionKind indicates an expected call of GroupVersionKind.
func (mr *MockSecretControllerMockRecorder) GroupVersionKind() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GroupVersionKind", reflect.TypeOf((*MockSecretController)(nil).GroupVersionKind))
}

// Informer mocks base method.
func (m *MockSecretController) Informer() cache.SharedIndexInformer {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Informer")
	ret0, _ := ret[0].(cache.SharedIndexInformer)
	return ret0
}

// Informer indicates an expected call of Informer.
func (mr *MockSecretControllerMockRecorder) Informer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Informer", reflect.TypeOf((*MockSecretController)(nil).Informer))
}

// List mocks base method.
func (m *MockSecretController) List(arg0 string, arg1 v11.ListOptions) (*v10.SecretList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0, arg1)
	ret0, _ := ret[0].(*v10.SecretList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockSecretControllerMockRecorder) List(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockSecretController)(nil).List), arg0, arg1)
}

// OnChange mocks base method.
func (m *MockSecretController) OnChange(arg0 context.Context, arg1 string, arg2 v1.SecretHandler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnChange", arg0, arg1, arg2)
}

// OnChange indicates an expected call of OnChange.
func (mr *MockSecretControllerMockRecorder) OnChange(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnChange", reflect.TypeOf((*MockSecretController)(nil).OnChange), arg0, arg1, arg2)
}

// OnRemove mocks base method.
func (m *MockSecretController) OnRemove(arg0 context.Context, arg1 string, arg2 v1.SecretHandler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnRemove", arg0, arg1, arg2)
}

// OnRemove indicates an expected call of OnRemove.
func (mr *MockSecretControllerMockRecorder) OnRemove(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnRemove", reflect.TypeOf((*MockSecretController)(nil).OnRemove), arg0, arg1, arg2)
}

// Patch mocks base method.
func (m *MockSecretController) Patch(arg0, arg1 string, arg2 types.PatchType, arg3 []byte, arg4 ...string) (*v10.Secret, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1, arg2, arg3}
	for _, a := range arg4 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Patch", varargs...)
	ret0, _ := ret[0].(*v10.Secret)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Patch indicates an expected call of Patch.
func (mr *MockSecretControllerMockRecorder) Patch(arg0, arg1, arg2, arg3 interface{}, arg4 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1, arg2, arg3}, arg4...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Patch", reflect.TypeOf((*MockSecretController)(nil).Patch), varargs...)
}

// Update mocks base method.
func (m *MockSecretController) Update(arg0 *v10.Secret) (*v10.Secret, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0)
	ret0, _ := ret[0].(*v10.Secret)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockSecretControllerMockRecorder) Update(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockSecretController)(nil).Update), arg0)
}

// Updater mocks base method.
func (m *MockSecretController) Updater() generic.Updater {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Updater")
	ret0, _ := ret[0].(generic.Updater)
	return ret0
}

// Updater indicates an expected call of Updater.
func (mr *MockSecretControllerMockRecorder) Updater() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Updater", reflect.TypeOf((*MockSecretController)(nil).Updater))
}

// Watch mocks base method.
func (m *MockSecretController) Watch(arg0 string, arg1 v11.ListOptions) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Watch", arg0, arg1)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Watch indicates an expected call of Watch.
func (mr *MockSecretControllerMockRecorder) Watch(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockSecretController)(nil).Watch), arg0, arg1)
}
