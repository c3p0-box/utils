// Package set provides a generic set data structure for Go using Go generics.
// It supports any comparable type and offers both method-based and standalone
// function APIs for convenience.
//
// The Set type is implemented as a map[T]struct{} for memory efficiency.
// All operations are O(1) average case for add, remove, and contains.
//
// Example:
//
//	s := set.New[string]()
//	s.Add("apple")
//	s.Add("banana")
//	if s.Contains("apple") {
//		fmt.Println("Found apple")
//	}
//	fmt.Println(s.Size()) // 2
package set

// Void is an empty struct used as map values for memory-efficient sets.
type Void struct{}

// Set is a generic set type that can hold any comparable type.
type Set[T comparable] map[T]Void

// New creates a new empty set.
func New[T comparable]() Set[T] {
	return make(Set[T])
}

// Add adds an item to the set.
func (s Set[T]) Add(item T) {
	s[item] = Void{}
}

// AddList adds multiple items to the set.
func (s Set[T]) AddList(items []T) {
	for i := range items {
		s[items[i]] = Void{}
	}
}

// Contains checks if an item exists in the set.
func (s Set[T]) Contains(item T) bool {
	_, exists := s[item]
	return exists
}

// Remove removes an item from the set.
func (s Set[T]) Remove(item T) {
	delete(s, item)
}

// Size returns the number of items in the set.
func (s Set[T]) Size() int {
	return len(s)
}

// IsEmpty returns true if the set is empty.
func (s Set[T]) IsEmpty() bool {
	return len(s) == 0
}

// Clear removes all items from the set.
func (s Set[T]) Clear() {
	for k := range s {
		delete(s, k)
	}
}

// ToSlice returns all items in the set as a slice.
func (s Set[T]) ToSlice() []T {
	items := make([]T, 0, len(s))
	for item := range s {
		items = append(items, item)
	}
	return items
}

// Add adds an item to the set using the standalone function API.
func Add[T comparable](set Set[T], item T) {
	set[item] = Void{}
}

// AddList adds multiple items to the set using the standalone function API.
func AddList[T comparable](set Set[T], items []T) {
	for i := range items {
		set[items[i]] = Void{}
	}
}

// Contains checks if an item exists in the set using the standalone function API.
func Contains[T comparable](set Set[T], item T) bool {
	_, exists := set[item]
	return exists
}

// Remove removes an item from the set using the standalone function API.
func Remove[T comparable](set Set[T], item T) {
	delete(set, item)
}
