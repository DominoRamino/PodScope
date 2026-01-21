package hub

import (
	"os"
	"strconv"
	"sync"

	"github.com/podscope/podscope/pkg/protocol"
)

// FlowRingBuffer is a fixed-size circular buffer for storing flows.
// It provides O(1) insertion with automatic eviction of oldest flows when full.
type FlowRingBuffer struct {
	flows    []*protocol.Flow
	capacity int
	head     int // Next write position
	size     int // Current number of elements
	mutex    sync.RWMutex
	index    map[string]int // flow.ID -> position for O(1) updates
}

// NewFlowRingBuffer creates a new ring buffer with the specified capacity.
// If capacity is 0, it reads from MAX_FLOWS environment variable (default 10000).
func NewFlowRingBuffer(capacity int) *FlowRingBuffer {
	if capacity <= 0 {
		capacity = getEnvInt("MAX_FLOWS", 10000)
	}

	return &FlowRingBuffer{
		flows:    make([]*protocol.Flow, capacity),
		capacity: capacity,
		index:    make(map[string]int, capacity),
	}
}

// Add inserts a flow into the buffer. If the flow ID already exists,
// it updates the existing flow in place. If the buffer is full,
// the oldest flow is evicted. Returns true if this was a new flow.
func (r *FlowRingBuffer) Add(flow *protocol.Flow) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if this is an update to an existing flow
	if pos, exists := r.index[flow.ID]; exists {
		r.flows[pos] = flow
		return false // Update, not new
	}

	// Evict oldest if at capacity
	if r.size == r.capacity {
		evicted := r.flows[r.head]
		if evicted != nil {
			delete(r.index, evicted.ID)
		}
	}

	// Insert new flow
	r.flows[r.head] = flow
	r.index[flow.ID] = r.head
	r.head = (r.head + 1) % r.capacity
	if r.size < r.capacity {
		r.size++
	}

	return true // New flow
}

// GetAll returns all flows in chronological order (oldest first).
func (r *FlowRingBuffer) GetAll() []*protocol.Flow {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.size == 0 {
		return []*protocol.Flow{}
	}

	result := make([]*protocol.Flow, 0, r.size)
	// Start from oldest: (head - size) wrapped around
	start := (r.head - r.size + r.capacity) % r.capacity
	for i := 0; i < r.size; i++ {
		pos := (start + i) % r.capacity
		if r.flows[pos] != nil {
			result = append(result, r.flows[pos])
		}
	}
	return result
}

// GetRecent returns the most recent n flows (newest first).
// If n > size, returns all available flows.
func (r *FlowRingBuffer) GetRecent(n int) []*protocol.Flow {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.size == 0 {
		return []*protocol.Flow{}
	}

	if n > r.size {
		n = r.size
	}

	result := make([]*protocol.Flow, 0, n)
	// Start from newest: (head - 1) and go backwards
	for i := 0; i < n; i++ {
		pos := (r.head - 1 - i + r.capacity) % r.capacity
		if r.flows[pos] != nil {
			result = append(result, r.flows[pos])
		}
	}
	return result
}

// Size returns the current number of flows in the buffer.
func (r *FlowRingBuffer) Size() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.size
}

// Capacity returns the maximum capacity of the buffer.
func (r *FlowRingBuffer) Capacity() int {
	return r.capacity
}

// Get retrieves a flow by ID. Returns nil if not found.
func (r *FlowRingBuffer) Get(id string) *protocol.Flow {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if pos, exists := r.index[id]; exists {
		return r.flows[pos]
	}
	return nil
}

// Clear removes all flows from the buffer.
func (r *FlowRingBuffer) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.flows = make([]*protocol.Flow, r.capacity)
	r.index = make(map[string]int, r.capacity)
	r.head = 0
	r.size = 0
}

// getEnvInt reads an integer from environment variable with a default value.
func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}
