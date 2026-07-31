package trace

import (
	"sync"

	"go.opentelemetry.io/otel/attribute"
)

var resourceAttributes = struct {
	sync.RWMutex
	values []attribute.KeyValue
}{}

// AddResources add more resources in addition to configured trace name.
func AddResources(attrs ...attribute.KeyValue) {
	resourceAttributes.Lock()
	resourceAttributes.values = append(resourceAttributes.values, attrs...)
	resourceAttributes.Unlock()
}

func resourceSnapshot() []attribute.KeyValue {
	resourceAttributes.RLock()
	defer resourceAttributes.RUnlock()
	return append([]attribute.KeyValue(nil), resourceAttributes.values...)
}
