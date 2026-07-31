package trace

import (
	"sync"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestResourcesSupportConcurrentAppendAndSnapshot(t *testing.T) {
	resourceAttributes.Lock()
	resourceAttributes.values = nil
	resourceAttributes.Unlock()

	var wait sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			AddResources(attribute.Int("worker", index))
			_ = resourceSnapshot()
		}(worker)
	}
	wait.Wait()
	if got := len(resourceSnapshot()); got != 16 {
		t.Fatalf("resource count = %d, want 16", got)
	}
}
