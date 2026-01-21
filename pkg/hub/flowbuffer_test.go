package hub

import (
	"os"
	"testing"

	"github.com/podscope/podscope/pkg/protocol"
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

// Helper to create a test flow with a given ID
func createTestFlow(id string) *protocol.Flow {
	return &protocol.Flow{
		ID:       id,
		SrcIP:    "10.0.0.1",
		SrcPort:  12345,
		DstIP:    "10.0.0.2",
		DstPort:  80,
		Protocol: protocol.ProtocolTCP,
		Status:   protocol.StatusOpen,
	}
}

func TestAdd_NewFlowReturnsTrue(t *testing.T) {
	rb := NewFlowRingBuffer(10)
	flow := createTestFlow("flow-1")

	isNew := rb.Add(flow)

	if !isNew {
		t.Error("expected Add() to return true for new flow")
	}
	if rb.Size() != 1 {
		t.Errorf("expected size 1 after adding one flow, got %d", rb.Size())
	}
}

func TestAdd_UpdateExistingFlowReturnsFalse(t *testing.T) {
	rb := NewFlowRingBuffer(10)

	// Add original flow
	original := createTestFlow("flow-1")
	original.BytesSent = 100
	rb.Add(original)

	// Update with same ID
	updated := createTestFlow("flow-1")
	updated.BytesSent = 200

	isNew := rb.Add(updated)

	if isNew {
		t.Error("expected Add() to return false for existing flow update")
	}
	if rb.Size() != 1 {
		t.Errorf("expected size to remain 1 after update, got %d", rb.Size())
	}

	// Verify the flow was actually updated
	retrieved := rb.Get("flow-1")
	if retrieved == nil {
		t.Fatal("expected to retrieve flow, got nil")
	}
	if retrieved.BytesSent != 200 {
		t.Errorf("expected updated BytesSent 200, got %d", retrieved.BytesSent)
	}
}

func TestAdd_EvictsOldestWhenAtCapacity(t *testing.T) {
	capacity := 3
	rb := NewFlowRingBuffer(capacity)

	// Fill buffer to capacity
	flow1 := createTestFlow("flow-1")
	flow2 := createTestFlow("flow-2")
	flow3 := createTestFlow("flow-3")

	rb.Add(flow1) // Oldest
	rb.Add(flow2)
	rb.Add(flow3)

	if rb.Size() != 3 {
		t.Errorf("expected size 3, got %d", rb.Size())
	}

	// Add one more, should evict flow-1
	flow4 := createTestFlow("flow-4")
	isNew := rb.Add(flow4)

	if !isNew {
		t.Error("expected Add() to return true for new flow after eviction")
	}
	if rb.Size() != 3 {
		t.Errorf("expected size to remain 3 after eviction, got %d", rb.Size())
	}

	// flow-1 should be evicted
	if rb.Get("flow-1") != nil {
		t.Error("expected flow-1 to be evicted, but it's still in buffer")
	}

	// Other flows should still exist
	if rb.Get("flow-2") == nil {
		t.Error("expected flow-2 to still exist")
	}
	if rb.Get("flow-3") == nil {
		t.Error("expected flow-3 to still exist")
	}
	if rb.Get("flow-4") == nil {
		t.Error("expected flow-4 to exist")
	}
}

func TestAdd_IndexMapUpdatedCorrectlyOnEviction(t *testing.T) {
	capacity := 2
	rb := NewFlowRingBuffer(capacity)

	// Fill buffer
	flow1 := createTestFlow("flow-1")
	flow2 := createTestFlow("flow-2")
	rb.Add(flow1)
	rb.Add(flow2)

	// Verify initial index state
	if rb.Get("flow-1") == nil || rb.Get("flow-2") == nil {
		t.Fatal("expected both flows to be accessible initially")
	}

	// Evict flow-1 by adding flow-3
	flow3 := createTestFlow("flow-3")
	rb.Add(flow3)

	// Index should no longer contain flow-1
	if rb.Get("flow-1") != nil {
		t.Error("expected flow-1 to be removed from index after eviction")
	}

	// Index should contain flow-2 and flow-3
	if rb.Get("flow-2") == nil {
		t.Error("expected flow-2 to remain in index")
	}
	if rb.Get("flow-3") == nil {
		t.Error("expected flow-3 to be in index")
	}

	// Evict flow-2 by adding flow-4
	flow4 := createTestFlow("flow-4")
	rb.Add(flow4)

	// Now flow-1 and flow-2 should be gone
	if rb.Get("flow-1") != nil {
		t.Error("expected flow-1 to remain evicted")
	}
	if rb.Get("flow-2") != nil {
		t.Error("expected flow-2 to be evicted after adding flow-4")
	}

	// flow-3 and flow-4 should exist
	if rb.Get("flow-3") == nil {
		t.Error("expected flow-3 to remain in index")
	}
	if rb.Get("flow-4") == nil {
		t.Error("expected flow-4 to be in index")
	}
}

func TestAdd_MultipleEvictions(t *testing.T) {
	capacity := 3
	rb := NewFlowRingBuffer(capacity)

	// Add many flows, causing multiple evictions
	for i := 1; i <= 10; i++ {
		flow := createTestFlow(string(rune('a'-1+i)) + "-flow")
		rb.Add(flow)
	}

	if rb.Size() != capacity {
		t.Errorf("expected size %d, got %d", capacity, rb.Size())
	}

	// Only the last 3 flows should exist
	flows := rb.GetAll()
	if len(flows) != 3 {
		t.Errorf("expected 3 flows from GetAll(), got %d", len(flows))
	}

	// Verify the oldest flows were evicted (flows a through g should be gone)
	// Only h, i, j should remain
	expectedIDs := map[string]bool{"h-flow": true, "i-flow": true, "j-flow": true}
	for _, f := range flows {
		if !expectedIDs[f.ID] {
			t.Errorf("unexpected flow ID %s in buffer", f.ID)
		}
	}
}
