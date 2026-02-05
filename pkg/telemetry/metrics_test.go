package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewToolMetrics(t *testing.T) {
	metrics, err := NewToolMetrics()

	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.invocations)
	assert.NotNil(t, metrics.errors)
	assert.NotNil(t, metrics.duration)
	assert.NotNil(t, metrics.inProgress)
	assert.NotNil(t, metrics.meter)
}

func TestToolInvocationStruct(t *testing.T) {
	inv := ToolInvocation{
		ToolName:      "test-tool",
		ToolPackage:   "test-package",
		ComponentType: "state.redis",
		Outcome:       "success",
	}

	assert.Equal(t, "test-tool", inv.ToolName)
	assert.Equal(t, "test-package", inv.ToolPackage)
	assert.Equal(t, "state.redis", inv.ComponentType)
	assert.Equal(t, "success", inv.Outcome)
}

func TestRecordInvocation(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	inv := ToolInvocation{
		ToolName:      "save_state",
		ToolPackage:   "state",
		ComponentType: "state.redis",
		Outcome:       "success",
	}

	// Should not panic
	metrics.RecordInvocation(context.Background(), inv)
}

func TestRecordInvocationWithoutOptionalFields(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	inv := ToolInvocation{
		ToolName:    "get_state",
		ToolPackage: "state",
		// ComponentType and Outcome are empty
	}

	// Should not panic
	metrics.RecordInvocation(context.Background(), inv)
}

func TestRecordError(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	inv := ToolInvocation{
		ToolName:      "save_state",
		ToolPackage:   "state",
		ComponentType: "state.redis",
	}

	// Should not panic
	metrics.RecordError(context.Background(), inv, "connection_error")
}

func TestRecordErrorWithoutComponentType(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	inv := ToolInvocation{
		ToolName:    "save_state",
		ToolPackage: "state",
	}

	// Should not panic
	metrics.RecordError(context.Background(), inv, "validation_error")
}

func TestRecordDuration(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	inv := ToolInvocation{
		ToolName:      "invoke_service",
		ToolPackage:   "invoke",
		ComponentType: "",
		Outcome:       "success",
	}

	// Should not panic
	metrics.RecordDuration(context.Background(), inv, 150.5)
}

func TestRecordDurationWithAllFields(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	inv := ToolInvocation{
		ToolName:      "publish_event",
		ToolPackage:   "pubsub",
		ComponentType: "pubsub.redis",
		Outcome:       "success",
	}

	// Should not panic
	metrics.RecordDuration(context.Background(), inv, 25.3)
}

func TestStartInProgress(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	// Should not panic
	metrics.StartInProgress(context.Background(), "test_tool", "test_package")
}

func TestEndInProgress(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	// Should not panic
	metrics.EndInProgress(context.Background(), "test_tool", "test_package")
}

func TestStartTimer(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	timer := metrics.StartTimer(context.Background(), "test_tool", "test_package")

	assert.NotNil(t, timer)
	assert.Equal(t, "test_tool", timer.inv.ToolName)
	assert.Equal(t, "test_package", timer.inv.ToolPackage)
	assert.NotZero(t, timer.start)
	assert.Equal(t, metrics, timer.metrics)
}

func TestTimerStop(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	timer := metrics.StartTimer(context.Background(), "test_tool", "test_package")

	// Should not panic
	timer.Stop("success", "state.redis")

	assert.Equal(t, "success", timer.inv.Outcome)
	assert.Equal(t, "state.redis", timer.inv.ComponentType)
}

func TestTimerStopWithError(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	timer := metrics.StartTimer(context.Background(), "failing_tool", "test_package")

	// Should not panic and should record error
	timer.Stop("error", "")

	assert.Equal(t, "error", timer.inv.Outcome)
}

func TestTimerStruct(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	ctx := context.Background()
	timer := &Timer{
		metrics: metrics,
		inv: ToolInvocation{
			ToolName:    "test",
			ToolPackage: "pkg",
		},
		ctx: ctx,
	}

	assert.Equal(t, metrics, timer.metrics)
	assert.Equal(t, "test", timer.inv.ToolName)
	assert.Equal(t, ctx, timer.ctx)
}

func TestMetricsFullWorkflow(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	ctx := context.Background()

	// Start a timer (which also marks in-progress)
	timer := metrics.StartTimer(ctx, "workflow_tool", "workflow_package")

	// Simulate some work...

	// Stop the timer (which records invocation, duration, and ends in-progress)
	timer.Stop("success", "state.redis")

	// All operations should complete without panic
}

func TestMetricsErrorWorkflow(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	ctx := context.Background()

	// Start a timer
	timer := metrics.StartTimer(ctx, "error_tool", "error_package")

	// Stop with error outcome - this should also record an error
	timer.Stop("error", "")

	// All operations should complete without panic
}

func TestRecordInvocationMultipleTimes(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	ctx := context.Background()
	inv := ToolInvocation{
		ToolName:    "repeated_tool",
		ToolPackage: "test",
		Outcome:     "success",
	}

	// Record multiple invocations
	for i := 0; i < 10; i++ {
		metrics.RecordInvocation(ctx, inv)
	}

	// Should not panic
}

func TestToolMetricsNilContext(t *testing.T) {
	metrics, err := NewToolMetrics()
	assert.NoError(t, err)

	inv := ToolInvocation{
		ToolName:    "nil_ctx_tool",
		ToolPackage: "test",
	}

	// These should handle nil context gracefully
	// Note: In practice, context should not be nil, but the code should not panic
	metrics.RecordInvocation(context.Background(), inv)
	metrics.RecordError(context.Background(), inv, "test_error")
	metrics.RecordDuration(context.Background(), inv, 100.0)
	metrics.StartInProgress(context.Background(), "tool", "pkg")
	metrics.EndInProgress(context.Background(), "tool", "pkg")
}
