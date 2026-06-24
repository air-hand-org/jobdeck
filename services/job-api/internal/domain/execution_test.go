package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutionStatus_Valid(t *testing.T) {
	assert.True(t, EXECUTION_STATUS_QUEUED.Valid())
	assert.False(t, ExecutionStatus("").Valid())
}

func TestExecutionTrigger_Valid(t *testing.T) {
	assert.True(t, EXECUTION_TRIGGER_MANUAL.Valid())
	assert.False(t, ExecutionTrigger("").Valid())
}
