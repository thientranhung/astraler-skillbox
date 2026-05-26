package providers

// Registry maps provider keys to their adapters.
type Registry struct {
	adapters map[string]ProviderAdapter
}

// NewDefaultRegistry returns a registry pre-populated with all production adapters.
// Use this in main and in registry-vs-seed tests so both always stay in sync.
func NewDefaultRegistry() *Registry {
	return NewRegistry(
		NewGenericAgentsAdapter(),
		NewClaudeAdapter(),
		NewCodexAdapter(),
		NewGeminiAdapter(),
		NewAntigravityCLIAdapter(),
	)
}

// NewRegistry creates a registry pre-populated with the given adapters.
func NewRegistry(adapters ...ProviderAdapter) *Registry {
	r := &Registry{adapters: make(map[string]ProviderAdapter, len(adapters))}
	for _, a := range adapters {
		r.adapters[a.Key()] = a
	}
	return r
}

// Get returns the adapter for key, or (nil, false) if not registered.
func (r *Registry) Get(key string) (ProviderAdapter, bool) {
	a, ok := r.adapters[key]
	return a, ok
}

// All returns all registered adapters in unspecified order.
func (r *Registry) All() []ProviderAdapter {
	result := make([]ProviderAdapter, 0, len(r.adapters))
	for _, a := range r.adapters {
		result = append(result, a)
	}
	return result
}
