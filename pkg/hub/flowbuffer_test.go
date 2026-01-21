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

// Tests for US-003: FlowRingBuffer retrieval methods

func TestGetAll_EmptyBuffer(t *testing.T) {
	rb := NewFlowRingBuffer(10)

	flows := rb.GetAll()

	if flows == nil {
		t.Error("expected GetAll() to return non-nil slice, got nil")
	}
	if len(flows) != 0 {
		t.Errorf("expected empty slice for empty buffer, got %d flows", len(flows))
	}
}

func TestGetAll_ChronologicalOrder(t *testing.T) {
	rb := NewFlowRingBuffer(10)

	// Add flows in order
	flow1 := createTestFlow("flow-1")
	flow2 := createTestFlow("flow-2")
	flow3 := createTestFlow("flow-3")

	rb.Add(flow1) // Oldest
	rb.Add(flow2)
	rb.Add(flow3) // Newest

	flows := rb.GetAll()

	if len(flows) != 3 {
		t.Fatalf("expected 3 flows, got %d", len(flows))
	}

	// Oldest first
	if flows[0].ID != "flow-1" {
		t.Errorf("expected first flow to be flow-1 (oldest), got %s", flows[0].ID)
	}
	if flows[1].ID != "flow-2" {
		t.Errorf("expected second flow to be flow-2, got %s", flows[1].ID)
	}
	if flows[2].ID != "flow-3" {
		t.Errorf("expected third flow to be flow-3 (newest), got %s", flows[2].ID)
	}
}

func TestGetAll_OrderAfterWrapAround(t *testing.T) {
	// Use small capacity to force wrap-around
	capacity := 3
	rb := NewFlowRingBuffer(capacity)

	// Add 5 flows to a capacity-3 buffer, causing wrap-around
	rb.Add(createTestFlow("flow-1")) // Evicted
	rb.Add(createTestFlow("flow-2")) // Evicted
	rb.Add(createTestFlow("flow-3")) // Oldest remaining
	rb.Add(createTestFlow("flow-4"))
	rb.Add(createTestFlow("flow-5")) // Newest

	flows := rb.GetAll()

	if len(flows) != 3 {
		t.Fatalf("expected 3 flows after eviction, got %d", len(flows))
	}

	// Should be in chronological order: flow-3 (oldest), flow-4, flow-5 (newest)
	expectedOrder := []string{"flow-3", "flow-4", "flow-5"}
	for i, expected := range expectedOrder {
		if flows[i].ID != expected {
			t.Errorf("position %d: expected %s, got %s", i, expected, flows[i].ID)
		}
	}
}

func TestGetAll_OrderAfterMultipleWrapArounds(t *testing.T) {
	capacity := 3
	rb := NewFlowRingBuffer(capacity)

	// Add 10 flows to force multiple complete wrap-arounds
	for i := 1; i <= 10; i++ {
		rb.Add(createTestFlow(string(rune('0'+i)) + "-flow"))
	}

	flows := rb.GetAll()

	if len(flows) != 3 {
		t.Fatalf("expected 3 flows, got %d", len(flows))
	}

	// Last 3 flows added were 8, 9, 10 - should be in chronological order
	expectedOrder := []string{"8-flow", "9-flow", ":-flow"} // ':' is rune 58 = '0'+10
	for i, expected := range expectedOrder {
		if flows[i].ID != expected {
			t.Errorf("position %d: expected %s, got %s", i, expected, flows[i].ID)
		}
	}
}

func TestGetRecent_NewestFirstOrdering(t *testing.T) {
	rb := NewFlowRingBuffer(10)

	rb.Add(createTestFlow("flow-1")) // Oldest
	rb.Add(createTestFlow("flow-2"))
	rb.Add(createTestFlow("flow-3")) // Newest

	// Get 2 most recent
	flows := rb.GetRecent(2)

	if len(flows) != 2 {
		t.Fatalf("expected 2 flows, got %d", len(flows))
	}

	// Newest first
	if flows[0].ID != "flow-3" {
		t.Errorf("expected first to be flow-3 (newest), got %s", flows[0].ID)
	}
	if flows[1].ID != "flow-2" {
		t.Errorf("expected second to be flow-2, got %s", flows[1].ID)
	}
}

func TestGetRecent_EmptyBuffer(t *testing.T) {
	rb := NewFlowRingBuffer(10)

	flows := rb.GetRecent(5)

	if flows == nil {
		t.Error("expected GetRecent() to return non-nil slice, got nil")
	}
	if len(flows) != 0 {
		t.Errorf("expected empty slice for empty buffer, got %d flows", len(flows))
	}
}

func TestGetRecent_RequestMoreThanSize(t *testing.T) {
	rb := NewFlowRingBuffer(10)

	rb.Add(createTestFlow("flow-1"))
	rb.Add(createTestFlow("flow-2"))

	// Request 100 when only 2 exist
	flows := rb.GetRecent(100)

	if len(flows) != 2 {
		t.Errorf("expected 2 flows (all available), got %d", len(flows))
	}

	// Should still be newest first
	if flows[0].ID != "flow-2" {
		t.Errorf("expected first to be flow-2 (newest), got %s", flows[0].ID)
	}
	if flows[1].ID != "flow-1" {
		t.Errorf("expected second to be flow-1 (oldest), got %s", flows[1].ID)
	}
}

func TestGetRecent_AfterWrapAround(t *testing.T) {
	capacity := 3
	rb := NewFlowRingBuffer(capacity)

	// Add 5 flows to force wrap-around
	rb.Add(createTestFlow("flow-1")) // Evicted
	rb.Add(createTestFlow("flow-2")) // Evicted
	rb.Add(createTestFlow("flow-3"))
	rb.Add(createTestFlow("flow-4"))
	rb.Add(createTestFlow("flow-5")) // Newest

	flows := rb.GetRecent(2)

	if len(flows) != 2 {
		t.Fatalf("expected 2 flows, got %d", len(flows))
	}

	// Newest first: flow-5, then flow-4
	if flows[0].ID != "flow-5" {
		t.Errorf("expected first to be flow-5 (newest), got %s", flows[0].ID)
	}
	if flows[1].ID != "flow-4" {
		t.Errorf("expected second to be flow-4, got %s", flows[1].ID)
	}
}

func TestGet_ExistingFlow(t *testing.T) {
	rb := NewFlowRingBuffer(10)

	original := createTestFlow("target-flow")
	original.BytesSent = 999
	original.SrcIP = "192.168.1.100"

	rb.Add(createTestFlow("other-1"))
	rb.Add(original)
	rb.Add(createTestFlow("other-2"))

	retrieved := rb.Get("target-flow")

	if retrieved == nil {
		t.Fatal("expected to retrieve flow, got nil")
	}
	if retrieved.ID != "target-flow" {
		t.Errorf("expected ID 'target-flow', got %s", retrieved.ID)
	}
	if retrieved.BytesSent != 999 {
		t.Errorf("expected BytesSent 999, got %d", retrieved.BytesSent)
	}
	if retrieved.SrcIP != "192.168.1.100" {
		t.Errorf("expected SrcIP '192.168.1.100', got %s", retrieved.SrcIP)
	}
}

func TestGet_NonExistingFlow(t *testing.T) {
	rb := NewFlowRingBuffer(10)

	rb.Add(createTestFlow("flow-1"))
	rb.Add(createTestFlow("flow-2"))

	retrieved := rb.Get("nonexistent-flow")

	if retrieved != nil {
		t.Errorf("expected nil for non-existent flow, got %+v", retrieved)
	}
}

func TestGet_EmptyBuffer(t *testing.T) {
	rb := NewFlowRingBuffer(10)

	retrieved := rb.Get("any-id")

	if retrieved != nil {
		t.Errorf("expected nil for empty buffer, got %+v", retrieved)
	}
}

func TestGet_AfterEviction(t *testing.T) {
	capacity := 2
	rb := NewFlowRingBuffer(capacity)

	rb.Add(createTestFlow("flow-1")) // Will be evicted
	rb.Add(createTestFlow("flow-2"))
	rb.Add(createTestFlow("flow-3")) // Evicts flow-1

	// flow-1 should no longer be retrievable
	if rb.Get("flow-1") != nil {
		t.Error("expected evicted flow to return nil")
	}

	// flow-2 and flow-3 should still be retrievable
	if rb.Get("flow-2") == nil {
		t.Error("expected flow-2 to be retrievable")
	}
	if rb.Get("flow-3") == nil {
		t.Error("expected flow-3 to be retrievable")
	}
}
