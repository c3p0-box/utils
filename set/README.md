# set

A generic set data structure for Go, implemented using Go generics for type safety.

## Features

- **Generic Type Support**: Works with any comparable type using Go generics
- **Simple API**: Familiar set operations (Add, Remove, Contains, etc.)
- **String Utilities**: Helper functions for string-to-set conversions
- **Thread-Safe**: Safe for concurrent use when properly synchronized
- **Minimal Overhead**: Simple map-based implementation
- **No Dependencies**: Uses only Go standard library

## Installation

```bash
go get github.com/c3p0-box/utils/set
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/c3p0-box/utils/set"
)

func main() {
    // Create a new set of strings
    s := set.New[string]()
    
    // Add elements
    s.Add("apple")
    s.Add("banana")
    s.Add("cherry")
    
    // Check if element exists
    if s.Contains("apple") {
        fmt.Println("Set contains apple")
    }
    
    // Get size
    fmt.Printf("Set size: %d\n", s.Size())
    
    // Remove element
    s.Remove("banana")
    
    // Convert to slice
    items := s.ToSlice()
    fmt.Printf("Items: %v\n", items)
}
```

## API Reference

### Set Type

```go
type Set[T comparable] map[T]Void
```

`Set` is a generic set type that can hold any comparable type. It is implemented as a map with empty struct values for memory efficiency.

### Creating Sets

```go
// Create empty set
s := set.New[string]()
s := set.New[int]()

// Create from string
s := set.FromStr("a,b,c", ",")
```

### Set Methods

```go
// Add a single item
func (s Set[T]) Add(item T)

// Add multiple items
func (s Set[T]) AddList(items []T)

// Check if item exists
func (s Set[T]) Contains(item T) bool

// Remove an item
func (s Set[T]) Remove(item T)

// Get number of items
func (s Set[T]) Size() int

// Check if set is empty
func (s Set[T]) IsEmpty() bool

// Remove all items
func (s Set[T]) Clear()

// Convert to slice
func (s Set[T]) ToSlice() []T
```

### Standalone Functions

```go
// Add item to set
func Add[T comparable](set Set[T], item T)

// Add multiple items
func AddList[T comparable](set Set[T], items []T)

// Check if item exists
func Contains[T comparable](set Set[T], item T) bool

// Remove item
func Remove[T comparable](set Set[T], item T)
```

### String Utilities

```go
// FromStr converts a delimited string into a Set of strings.
// Trims whitespace and punctuation characters from each item.
func FromStr(str, sep string) Set[string]

// ToStr joins the items of a Set into a single string.
func ToStr(set Set[string], sep string) string
```

## Examples

### Basic Set Operations

```go
s := set.New[int]()

// Add elements
s.Add(1)
s.Add(2)
s.Add(3)
s.AddList([]int{4, 5, 6})

// Check membership
fmt.Println(s.Contains(2))  // true
fmt.Println(s.Contains(10)) // false

// Get size
fmt.Println(s.Size()) // 6

// Remove
s.Remove(2)
fmt.Println(s.Contains(2)) // false

// Iterate
for _, item := range s.ToSlice() {
    fmt.Println(item)
}
```

### Set Intersection/Difference (using Contains)

```go
set1 := set.New[string]()
set1.Add("a")
set1.Add("b")
set1.Add("c")

set2 := set.New[string]()
set2.Add("b")
set2.Add("c")
set2.Add("d")

// Intersection
intersection := set.New[string]()
for _, item := range set1.ToSlice() {
    if set2.Contains(item) {
        intersection.Add(item)
    }
}
fmt.Println(intersection.ToSlice()) // [b c]

// Difference
diff := set.New[string]()
for _, item := range set1.ToSlice() {
    if !set2.Contains(item) {
        diff.Add(item)
    }
}
fmt.Println(diff.ToSlice()) // [a]
```

### Working with Strings

```go
// Parse comma-separated values
csv := "apple, banana, cherry, apple"
fruits := set.FromStr(csv, ",")
fmt.Println(fruits.Size()) // 3 (duplicates removed)

// Convert back to string
result := set.ToStr(fruits, " | ")
fmt.Println(result) // e.g., "apple | banana | cherry"

// Parse with custom separator
paths := set.FromStr("/home;/var;/tmp;/home", ";")
fmt.Println(paths.Size()) // 3
```

### Using Standalone Functions

```go
s := set.New[string]()

// Using standalone functions
set.Add(s, "item1")
set.AddList(s, []string{"item2", "item3"})

if set.Contains(s, "item1") {
    fmt.Println("Found item1")
}

set.Remove(s, "item2")
```

## Performance

- **Add**: O(1) average case
- **Remove**: O(1) average case
- **Contains**: O(1) average case
- **Size**: O(1)
- **ToSlice**: O(n)

Memory overhead is minimal as the underlying map uses empty struct values.

## When to Use

Use this set implementation when you need:

- Fast membership testing
- Deduplication of elements
- Set operations (intersection, union, difference)
- Type-safe collections with Go generics

## Thread Safety

The `Set` type itself is not thread-safe. If you need concurrent access, protect the set with appropriate synchronization:

```go
type SafeSet struct {
    mu sync.RWMutex
    set.Set[string]
}

func (s *SafeSet) Add(item string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.Set.Add(item)
}

func (s *SafeSet) Contains(item string) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.Set.Contains(item)
}
```

## License

This package is part of the c3p0-box/utils collection and follows the same license terms.
