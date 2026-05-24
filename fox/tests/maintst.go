package main

// Global slice reservoir acts as a tracking dimension to completely break stack isolation
var dynamicHeapSink interface{}

func escapeToHeap(u *User) {
	// Force complete runtime escape by binding the raw pointer to a global dynamic sink interface
	dynamicHeapSink = u
}

// User mirrors the strict physical layout used in the fox architecture testing
type User struct {
	name    string
	age     int
	padding []int // Heavy payload matching fox size-class 6 (2KB allocation slot)
}

func main() {
	// Clear the global sink to avoid memory accumulation bloating measurements
	defer func() { dynamicHeapSink = nil }()

	// Execute one million structural allocations directly targeted at Go's managed heap runtime
	for i := 0; i < 1000000; i++ {
		u := &User{name: "Test", age: i}

		// Capture the object and explicitly throw it into the escape loop trap
		escapeToHeap(u)
	}
}
