package progress

import (
	"sync"
)

type slotBuffer struct {
	mu     sync.Mutex
	buffer map[EventType][]*SlotRecord
}

func newSlotBuffer() *slotBuffer {
	return &slotBuffer{
		buffer: make(map[EventType][]*SlotRecord),
	}
}

func (b *slotBuffer) Add(eventType EventType, record *SlotRecord) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buffer[eventType] = append(b.buffer[eventType], record)
}

func (b *slotBuffer) Flush() map[EventType][]*SlotRecord {
	b.mu.Lock()
	defer b.mu.Unlock()

	flushed := make(map[EventType][]*SlotRecord, len(b.buffer))
	for et, list := range b.buffer {
		flushed[et] = list
	}
	b.buffer = make(map[EventType][]*SlotRecord) // reset
	return flushed
}

func (b *slotBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	total := 0
	for _, list := range b.buffer {
		total += len(list)
	}
	return total
}
