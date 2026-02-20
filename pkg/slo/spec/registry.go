package spec

import "fmt"

// Registry stores SLI specifications.
// Registry는 SLI 명세를 저장합니다.
type Registry struct {
	items map[string]SLISpec
}

// NewRegistry creates a new Registry.
// NewRegistry는 새로운 Registry를 생성합니다.
func NewRegistry() *Registry {
	return &Registry{items: map[string]SLISpec{}}
}

// Register adds an SLI spec to the registry.
// Register는 SLI 명세를 레지스트리에 추가합니다.
func (r *Registry) Register(s SLISpec) error {
	if s.ID == "" {
		return fmt.Errorf("sli spec id is required")
	}
	if _, exists := r.items[s.ID]; exists {
		return fmt.Errorf("sli spec already registered: %s", s.ID)
	}
	r.items[s.ID] = s
	return nil
}

// MustRegister adds an SLI spec or panics if it fails.
// MustRegister는 SLI 명세를 추가하며, 실패 시 패닉을 발생시킵니다.
func (r *Registry) MustRegister(s SLISpec) {
	if err := r.Register(s); err != nil {
		panic(err)
	}
}

// Get retrieves an SLI spec by ID.
// Get은 ID로 SLI 명세를 조회합니다.
func (r *Registry) Get(id string) (SLISpec, bool) {
	s, ok := r.items[id]
	return s, ok
}

// List returns all registered SLI specs.
// List는 등록된 모든 SLI 명세를 반환합니다.
func (r *Registry) List() []SLISpec {
	out := make([]SLISpec, 0, len(r.items))
	for _, s := range r.items {
		out = append(out, s)
	}
	return out
}
