package stack

type (
	// Stack represents the stack data struct
	// Also exposes the underlying data array for easy persistence of the queues data
	//
	// Note that the Stack is not thread safe so accurate synchronization should
	// be done when using acrosss go routines/threads.
	Stack struct {
		items []interface{}
	}
)

// Create a new stack
func New() *Stack {
	return &Stack{}
}

// Return the number of items in the stack
func (s *Stack) Len() int {
	return len(s.items)
}

// View the top item on the stack
func (s *Stack) Peek() interface{} {
	if s.Len() == 0 {
		return nil
	}
	return s.items[s.Len()-1]
}

// Pop the top item of the stack and return it
func (s *Stack) Pop() interface{} {
	if s.Len() == 0 {
		return nil
	}

	top := s.Peek()

	n := s.Len() - 1
	s.items[n] = nil // clear internal reference to the data
	s.items = s.items[:n]

	return top
}

// Push a value onto the top of the stack
func (s *Stack) Push(value interface{}) {
	s.items = append(s.items, value)
}
