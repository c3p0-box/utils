package set

type Void struct{}

// Set is a generic set type that can hold any comparable type
type Set[T comparable] map[T]Void

// New creates a new empty set
func New[T comparable]() Set[T] {
	return make(Set[T])
}

// Add adds an item to the set
func (s Set[T]) Add(item T) {
	s[item] = Void{}
}

func (s Set[T]) AddList(items []T) {
	for i := range items {
		s[items[i]] = Void{}
	}
}

// Contains checks if an item exists in the set
func (s Set[T]) Contains(item T) bool {
	_, exists := s[item]
	return exists
}

// Remove removes an item from the set
func (s Set[T]) Remove(item T) {
	delete(s, item)
}

// Size returns the number of items in the set
func (s Set[T]) Size() int {
	return len(s)
}

// IsEmpty returns true if the set is empty
func (s Set[T]) IsEmpty() bool {
	return len(s) == 0
}

// Clear removes all items from the set
func (s Set[T]) Clear() {
	for k := range s {
		delete(s, k)
	}
}

// ToSlice returns all items in the set as a slice
func (s Set[T]) ToSlice() []T {
	items := make([]T, 0, len(s))
	for item := range s {
		items = append(items, item)
	}
	return items
}

func Add[T comparable](set Set[T], item T) {
	set[item] = Void{}
}

func AddList[T comparable](set Set[T], items []T) {
	for i := range items {
		set[items[i]] = Void{}
	}
}

func Contains[T comparable](set Set[T], item T) bool {
	_, exists := set[item]
	return exists
}

func Remove[T comparable](set Set[T], item T) {
	delete(set, item)
}
