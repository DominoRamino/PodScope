package hub

import (
	"os"
	"testing"
)

func TestNewFlowRingBuffer_DefaultCapacity(t *testing.T) {
	// When capacity is 0, should use default of 10000
	os.Unsetenv("MAX_FLOWS") // Ensure env var doesn't interfere

	rb := NewFlowRingBuffer(0)

	if rb.Capacity() != 10000 {
		t.Errorf("expected default capacity 10000, got %d", rb.Capacity())
	}
	if rb.Size() != 0 {
		t.Errorf("expected initial size 0, got %d", rb.Size())
	}
	if len(rb.flows) != 10000 {
		t.Errorf("expected flows slice length 10000, got %d", len(rb.flows))
	}
	if rb.index == nil {
		t.Error("expected index map to be initialized, got nil")
	}
	if rb.head != 0 {
		t.Errorf("expected head 0, got %d", rb.head)
	}
}

func TestNewFlowRingBuffer_CustomCapacity(t *testing.T) {
	// Custom capacity should be respected
	customCap := 500
	rb := NewFlowRingBuffer(customCap)

	if rb.Capacity() != customCap {
		t.Errorf("expected capacity %d, got %d", customCap, rb.Capacity())
	}
	if len(rb.flows) != customCap {
		t.Errorf("expected flows slice length %d, got %d", customCap, len(rb.flows))
	}
}

func TestNewFlowRingBuffer_NegativeCapacity(t *testing.T) {
	// Negative capacity should fall back to default
	os.Unsetenv("MAX_FLOWS")

	rb := NewFlowRingBuffer(-10)

	if rb.Capacity() != 10000 {
		t.Errorf("expected default capacity 10000 for negative input, got %d", rb.Capacity())
	}
}

func TestNewFlowRingBuffer_EnvOverride(t *testing.T) {
	// MAX_FLOWS environment variable should override default when capacity is 0
	os.Setenv("MAX_FLOWS", "5000")
	defer os.Unsetenv("MAX_FLOWS")

	rb := NewFlowRingBuffer(0)

	if rb.Capacity() != 5000 {
		t.Errorf("expected capacity 5000 from env, got %d", rb.Capacity())
	}
}

func TestNewFlowRingBuffer_EnvOverrideInvalidValue(t *testing.T) {
	// Invalid MAX_FLOWS should fall back to default
	os.Setenv("MAX_FLOWS", "not_a_number")
	defer os.Unsetenv("MAX_FLOWS")

	rb := NewFlowRingBuffer(0)

	if rb.Capacity() != 10000 {
		t.Errorf("expected default capacity 10000 for invalid env, got %d", rb.Capacity())
	}
}

func TestNewFlowRingBuffer_ExplicitCapacityIgnoresEnv(t *testing.T) {
	// Explicit positive capacity should ignore environment variable
	os.Setenv("MAX_FLOWS", "5000")
	defer os.Unsetenv("MAX_FLOWS")

	customCap := 2000
	rb := NewFlowRingBuffer(customCap)

	if rb.Capacity() != customCap {
		t.Errorf("expected explicit capacity %d to override env, got %d", customCap, rb.Capacity())
	}
}
