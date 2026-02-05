// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"
	"io"
	"time"

	pb "github.com/dapr/dapr/pkg/proto/runtime/v1"
	"github.com/dapr/go-sdk/actor"
	"github.com/dapr/go-sdk/actor/config"
	"github.com/dapr/go-sdk/client"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// DaprClient is an interface that matches the subset of client.Client we use in testing.
// This is necessary because the Dapr SDK uses unexported types in some method signatures
// (e.g., conversationRequest in ConverseAlpha1) which prevents direct interface embedding.
type DaprClient interface {
	InvokeBinding(ctx context.Context, in *client.InvokeBindingRequest) (*client.BindingEvent, error)
	InvokeOutputBinding(ctx context.Context, in *client.InvokeBindingRequest) error
	InvokeMethod(ctx context.Context, appID, methodName, verb string) ([]byte, error)
	InvokeMethodWithContent(ctx context.Context, appID, methodName, verb string, content *client.DataContent) ([]byte, error)
	InvokeMethodWithCustomContent(ctx context.Context, appID, methodName, verb string, contentType string, content interface{}) ([]byte, error)
	GetMetadata(ctx context.Context) (*client.GetMetadataResponse, error)
	SetMetadata(ctx context.Context, key, value string) error
	PublishEvent(ctx context.Context, pubsubName, topicName string, data interface{}, opts ...client.PublishEventOption) error
	PublishEventfromCustomContent(ctx context.Context, pubsubName, topicName string, data interface{}) error
	PublishEvents(ctx context.Context, pubsubName, topicName string, events []interface{}, opts ...client.PublishEventsOption) client.PublishEventsResponse
	GetSecret(ctx context.Context, storeName, key string, meta map[string]string) (map[string]string, error)
	GetBulkSecret(ctx context.Context, storeName string, meta map[string]string) (map[string]map[string]string, error)
	SaveState(ctx context.Context, storeName, key string, data []byte, meta map[string]string, so ...client.StateOption) error
	SaveStateWithETag(ctx context.Context, storeName, key string, data []byte, etag string, meta map[string]string, so ...client.StateOption) error
	SaveBulkState(ctx context.Context, storeName string, items ...*client.SetStateItem) error
	GetState(ctx context.Context, storeName, key string, meta map[string]string) (*client.StateItem, error)
	GetStateWithConsistency(ctx context.Context, storeName, key string, meta map[string]string, sc client.StateConsistency) (*client.StateItem, error)
	GetBulkState(ctx context.Context, storeName string, keys []string, meta map[string]string, parallelism int32) ([]*client.BulkStateItem, error)
	QueryStateAlpha1(ctx context.Context, storeName, query string, meta map[string]string) (*client.QueryResponse, error)
	DeleteState(ctx context.Context, storeName, key string, meta map[string]string) error
	DeleteStateWithETag(ctx context.Context, storeName, key string, etag *client.ETag, meta map[string]string, opts *client.StateOptions) error
	ExecuteStateTransaction(ctx context.Context, storeName string, meta map[string]string, ops []*client.StateOperation) error
	GetConfigurationItem(ctx context.Context, storeName, key string, opts ...client.ConfigurationOpt) (*client.ConfigurationItem, error)
	GetConfigurationItems(ctx context.Context, storeName string, keys []string, opts ...client.ConfigurationOpt) (map[string]*client.ConfigurationItem, error)
	SubscribeConfigurationItems(ctx context.Context, storeName string, keys []string, handler client.ConfigurationHandleFunction, opts ...client.ConfigurationOpt) (string, error)
	UnsubscribeConfigurationItems(ctx context.Context, storeName string, id string, opts ...client.ConfigurationOpt) error
	Subscribe(ctx context.Context, opts client.SubscriptionOptions) (*client.Subscription, error)
	SubscribeWithHandler(ctx context.Context, opts client.SubscriptionOptions, handler client.SubscriptionHandleFunction) (func() error, error)
	DeleteBulkState(ctx context.Context, storeName string, keys []string, meta map[string]string) error
	DeleteBulkStateItems(ctx context.Context, storeName string, items []*client.DeleteStateItem) error
	TryLockAlpha1(ctx context.Context, storeName string, req *client.LockRequest) (*client.LockResponse, error)
	UnlockAlpha1(ctx context.Context, storeName string, req *client.UnlockRequest) (*client.UnlockResponse, error)
	Encrypt(ctx context.Context, data io.Reader, opts client.EncryptOptions) (io.Reader, error)
	Decrypt(ctx context.Context, data io.Reader, opts client.DecryptOptions) (io.Reader, error)
	Shutdown(ctx context.Context) error
	Wait(ctx context.Context, timeout time.Duration) error
	WithTraceID(ctx context.Context, id string) context.Context
	WithAuthToken(token string)
	Close()
	RegisterActorTimer(ctx context.Context, req *client.RegisterActorTimerRequest) error
	UnregisterActorTimer(ctx context.Context, req *client.UnregisterActorTimerRequest) error
	RegisterActorReminder(ctx context.Context, req *client.RegisterActorReminderRequest) error
	UnregisterActorReminder(ctx context.Context, req *client.UnregisterActorReminderRequest) error
	InvokeActor(ctx context.Context, req *client.InvokeActorRequest) (*client.InvokeActorResponse, error)
	GetActorState(ctx context.Context, req *client.GetActorStateRequest) (*client.GetActorStateResponse, error)
	SaveStateTransactionally(ctx context.Context, actorType, actorID string, ops []*client.ActorStateOperation) error
	ImplActorClientStub(actorStub actor.Client, opt ...config.Option)
	ScheduleJobAlpha1(ctx context.Context, req *client.Job) error
	GetJobAlpha1(ctx context.Context, name string) (*client.Job, error)
	DeleteJobAlpha1(ctx context.Context, name string) error
	ConverseAlpha2(ctx context.Context, req client.ConversationRequestAlpha2) (*client.ConversationResponseAlpha2, error)
	GrpcClient() pb.DaprClient
	GrpcClientConn() *grpc.ClientConn
}

// MockDaprClient is a mock implementation of the DaprClient interface.
type MockDaprClient struct {
	mock.Mock
}

// Ensure MockDaprClient implements DaprClient
var _ DaprClient = (*MockDaprClient)(nil)

// InvokeBinding mocks the InvokeBinding method
func (m *MockDaprClient) InvokeBinding(ctx context.Context, in *client.InvokeBindingRequest) (*client.BindingEvent, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.BindingEvent), args.Error(1)
}

// InvokeOutputBinding mocks the InvokeOutputBinding method
func (m *MockDaprClient) InvokeOutputBinding(ctx context.Context, in *client.InvokeBindingRequest) error {
	args := m.Called(ctx, in)
	return args.Error(0)
}

// InvokeMethod mocks the InvokeMethod method
func (m *MockDaprClient) InvokeMethod(ctx context.Context, appID, methodName, verb string) ([]byte, error) {
	args := m.Called(ctx, appID, methodName, verb)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// InvokeMethodWithContent mocks the InvokeMethodWithContent method
func (m *MockDaprClient) InvokeMethodWithContent(ctx context.Context, appID, methodName, verb string, content *client.DataContent) ([]byte, error) {
	args := m.Called(ctx, appID, methodName, verb, content)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// InvokeMethodWithCustomContent mocks the InvokeMethodWithCustomContent method
func (m *MockDaprClient) InvokeMethodWithCustomContent(ctx context.Context, appID, methodName, verb string, contentType string, content interface{}) ([]byte, error) {
	args := m.Called(ctx, appID, methodName, verb, contentType, content)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// GetMetadata mocks the GetMetadata method
func (m *MockDaprClient) GetMetadata(ctx context.Context) (*client.GetMetadataResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.GetMetadataResponse), args.Error(1)
}

// SetMetadata mocks the SetMetadata method
func (m *MockDaprClient) SetMetadata(ctx context.Context, key, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

// PublishEvent mocks the PublishEvent method
func (m *MockDaprClient) PublishEvent(ctx context.Context, pubsubName, topicName string, data interface{}, opts ...client.PublishEventOption) error {
	args := m.Called(ctx, pubsubName, topicName, data, opts)
	return args.Error(0)
}

// PublishEventfromCustomContent mocks the deprecated method
func (m *MockDaprClient) PublishEventfromCustomContent(ctx context.Context, pubsubName, topicName string, data interface{}) error {
	args := m.Called(ctx, pubsubName, topicName, data)
	return args.Error(0)
}

// PublishEvents mocks the PublishEvents method
func (m *MockDaprClient) PublishEvents(ctx context.Context, pubsubName, topicName string, events []interface{}, opts ...client.PublishEventsOption) client.PublishEventsResponse {
	args := m.Called(ctx, pubsubName, topicName, events, opts)
	return args.Get(0).(client.PublishEventsResponse)
}

// GetSecret mocks the GetSecret method
func (m *MockDaprClient) GetSecret(ctx context.Context, storeName, key string, meta map[string]string) (map[string]string, error) {
	args := m.Called(ctx, storeName, key, meta)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

// GetBulkSecret mocks the GetBulkSecret method
func (m *MockDaprClient) GetBulkSecret(ctx context.Context, storeName string, meta map[string]string) (map[string]map[string]string, error) {
	args := m.Called(ctx, storeName, meta)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]map[string]string), args.Error(1)
}

// SaveState mocks the SaveState method
func (m *MockDaprClient) SaveState(ctx context.Context, storeName, key string, data []byte, meta map[string]string, so ...client.StateOption) error {
	args := m.Called(ctx, storeName, key, data, meta, so)
	return args.Error(0)
}

// SaveStateWithETag mocks the SaveStateWithETag method
func (m *MockDaprClient) SaveStateWithETag(ctx context.Context, storeName, key string, data []byte, etag string, meta map[string]string, so ...client.StateOption) error {
	args := m.Called(ctx, storeName, key, data, etag, meta, so)
	return args.Error(0)
}

// SaveBulkState mocks the SaveBulkState method
func (m *MockDaprClient) SaveBulkState(ctx context.Context, storeName string, items ...*client.SetStateItem) error {
	args := m.Called(ctx, storeName, items)
	return args.Error(0)
}

// GetState mocks the GetState method
func (m *MockDaprClient) GetState(ctx context.Context, storeName, key string, meta map[string]string) (*client.StateItem, error) {
	args := m.Called(ctx, storeName, key, meta)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.StateItem), args.Error(1)
}

// GetStateWithConsistency mocks the GetStateWithConsistency method
func (m *MockDaprClient) GetStateWithConsistency(ctx context.Context, storeName, key string, meta map[string]string, sc client.StateConsistency) (*client.StateItem, error) {
	args := m.Called(ctx, storeName, key, meta, sc)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.StateItem), args.Error(1)
}

// GetBulkState mocks the GetBulkState method
func (m *MockDaprClient) GetBulkState(ctx context.Context, storeName string, keys []string, meta map[string]string, parallelism int32) ([]*client.BulkStateItem, error) {
	args := m.Called(ctx, storeName, keys, meta, parallelism)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*client.BulkStateItem), args.Error(1)
}

// QueryStateAlpha1 mocks the QueryStateAlpha1 method
func (m *MockDaprClient) QueryStateAlpha1(ctx context.Context, storeName, query string, meta map[string]string) (*client.QueryResponse, error) {
	args := m.Called(ctx, storeName, query, meta)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.QueryResponse), args.Error(1)
}

// DeleteState mocks the DeleteState method
func (m *MockDaprClient) DeleteState(ctx context.Context, storeName, key string, meta map[string]string) error {
	args := m.Called(ctx, storeName, key, meta)
	return args.Error(0)
}

// DeleteStateWithETag mocks the DeleteStateWithETag method
func (m *MockDaprClient) DeleteStateWithETag(ctx context.Context, storeName, key string, etag *client.ETag, meta map[string]string, opts *client.StateOptions) error {
	args := m.Called(ctx, storeName, key, etag, meta, opts)
	return args.Error(0)
}

// ExecuteStateTransaction mocks the ExecuteStateTransaction method
func (m *MockDaprClient) ExecuteStateTransaction(ctx context.Context, storeName string, meta map[string]string, ops []*client.StateOperation) error {
	args := m.Called(ctx, storeName, meta, ops)
	return args.Error(0)
}

// GetConfigurationItem mocks the GetConfigurationItem method
func (m *MockDaprClient) GetConfigurationItem(ctx context.Context, storeName, key string, opts ...client.ConfigurationOpt) (*client.ConfigurationItem, error) {
	args := m.Called(ctx, storeName, key, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.ConfigurationItem), args.Error(1)
}

// GetConfigurationItems mocks the GetConfigurationItems method
func (m *MockDaprClient) GetConfigurationItems(ctx context.Context, storeName string, keys []string, opts ...client.ConfigurationOpt) (map[string]*client.ConfigurationItem, error) {
	args := m.Called(ctx, storeName, keys, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*client.ConfigurationItem), args.Error(1)
}

// SubscribeConfigurationItems mocks the SubscribeConfigurationItems method
func (m *MockDaprClient) SubscribeConfigurationItems(ctx context.Context, storeName string, keys []string, handler client.ConfigurationHandleFunction, opts ...client.ConfigurationOpt) (string, error) {
	args := m.Called(ctx, storeName, keys, handler, opts)
	return args.String(0), args.Error(1)
}

// UnsubscribeConfigurationItems mocks the UnsubscribeConfigurationItems method
func (m *MockDaprClient) UnsubscribeConfigurationItems(ctx context.Context, storeName string, id string, opts ...client.ConfigurationOpt) error {
	args := m.Called(ctx, storeName, id, opts)
	return args.Error(0)
}

// Subscribe mocks the Subscribe method
func (m *MockDaprClient) Subscribe(ctx context.Context, opts client.SubscriptionOptions) (*client.Subscription, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Subscription), args.Error(1)
}

// SubscribeWithHandler mocks the SubscribeWithHandler method
func (m *MockDaprClient) SubscribeWithHandler(ctx context.Context, opts client.SubscriptionOptions, handler client.SubscriptionHandleFunction) (func() error, error) {
	args := m.Called(ctx, opts, handler)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(func() error), args.Error(1)
}

// DeleteBulkState mocks the DeleteBulkState method
func (m *MockDaprClient) DeleteBulkState(ctx context.Context, storeName string, keys []string, meta map[string]string) error {
	args := m.Called(ctx, storeName, keys, meta)
	return args.Error(0)
}

// DeleteBulkStateItems mocks the DeleteBulkStateItems method
func (m *MockDaprClient) DeleteBulkStateItems(ctx context.Context, storeName string, items []*client.DeleteStateItem) error {
	args := m.Called(ctx, storeName, items)
	return args.Error(0)
}

// TryLockAlpha1 mocks the TryLockAlpha1 method
func (m *MockDaprClient) TryLockAlpha1(ctx context.Context, storeName string, req *client.LockRequest) (*client.LockResponse, error) {
	args := m.Called(ctx, storeName, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.LockResponse), args.Error(1)
}

// UnlockAlpha1 mocks the UnlockAlpha1 method
func (m *MockDaprClient) UnlockAlpha1(ctx context.Context, storeName string, req *client.UnlockRequest) (*client.UnlockResponse, error) {
	args := m.Called(ctx, storeName, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.UnlockResponse), args.Error(1)
}

// Encrypt mocks the Encrypt method
func (m *MockDaprClient) Encrypt(ctx context.Context, data io.Reader, opts client.EncryptOptions) (io.Reader, error) {
	args := m.Called(ctx, data, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.Reader), args.Error(1)
}

// Decrypt mocks the Decrypt method
func (m *MockDaprClient) Decrypt(ctx context.Context, data io.Reader, opts client.DecryptOptions) (io.Reader, error) {
	args := m.Called(ctx, data, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.Reader), args.Error(1)
}

// Shutdown mocks the Shutdown method
func (m *MockDaprClient) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Wait mocks the Wait method
func (m *MockDaprClient) Wait(ctx context.Context, timeout time.Duration) error {
	args := m.Called(ctx, timeout)
	return args.Error(0)
}

// WithTraceID mocks the WithTraceID method
func (m *MockDaprClient) WithTraceID(ctx context.Context, id string) context.Context {
	args := m.Called(ctx, id)
	return args.Get(0).(context.Context)
}

// WithAuthToken mocks the WithAuthToken method
func (m *MockDaprClient) WithAuthToken(token string) {
	m.Called(token)
}

// Close mocks the Close method
func (m *MockDaprClient) Close() {
	m.Called()
}

// RegisterActorTimer mocks the RegisterActorTimer method
func (m *MockDaprClient) RegisterActorTimer(ctx context.Context, req *client.RegisterActorTimerRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

// UnregisterActorTimer mocks the UnregisterActorTimer method
func (m *MockDaprClient) UnregisterActorTimer(ctx context.Context, req *client.UnregisterActorTimerRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

// RegisterActorReminder mocks the RegisterActorReminder method
func (m *MockDaprClient) RegisterActorReminder(ctx context.Context, req *client.RegisterActorReminderRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

// UnregisterActorReminder mocks the UnregisterActorReminder method
func (m *MockDaprClient) UnregisterActorReminder(ctx context.Context, req *client.UnregisterActorReminderRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

// InvokeActor mocks the InvokeActor method
func (m *MockDaprClient) InvokeActor(ctx context.Context, req *client.InvokeActorRequest) (*client.InvokeActorResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.InvokeActorResponse), args.Error(1)
}

// GetActorState mocks the GetActorState method
func (m *MockDaprClient) GetActorState(ctx context.Context, req *client.GetActorStateRequest) (*client.GetActorStateResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.GetActorStateResponse), args.Error(1)
}

// SaveStateTransactionally mocks the SaveStateTransactionally method
func (m *MockDaprClient) SaveStateTransactionally(ctx context.Context, actorType, actorID string, ops []*client.ActorStateOperation) error {
	args := m.Called(ctx, actorType, actorID, ops)
	return args.Error(0)
}

// ImplActorClientStub mocks the ImplActorClientStub method
func (m *MockDaprClient) ImplActorClientStub(actorStub actor.Client, opt ...config.Option) {
	m.Called(actorStub, opt)
}

// ScheduleJobAlpha1 mocks the ScheduleJobAlpha1 method
func (m *MockDaprClient) ScheduleJobAlpha1(ctx context.Context, req *client.Job) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

// GetJobAlpha1 mocks the GetJobAlpha1 method
func (m *MockDaprClient) GetJobAlpha1(ctx context.Context, name string) (*client.Job, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.Job), args.Error(1)
}

// DeleteJobAlpha1 mocks the DeleteJobAlpha1 method
func (m *MockDaprClient) DeleteJobAlpha1(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

// ConverseAlpha2 mocks the ConverseAlpha2 method for conversation API v2
func (m *MockDaprClient) ConverseAlpha2(ctx context.Context, req client.ConversationRequestAlpha2) (*client.ConversationResponseAlpha2, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.ConversationResponseAlpha2), args.Error(1)
}

// GrpcClient mocks the GrpcClient method
func (m *MockDaprClient) GrpcClient() pb.DaprClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(pb.DaprClient)
}

// GrpcClientConn mocks the GrpcClientConn method
func (m *MockDaprClient) GrpcClientConn() *grpc.ClientConn {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*grpc.ClientConn)
}
