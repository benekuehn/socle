package cmdutils

// FindIndexInStack finds the index of a branch in a stack
func FindIndexInStack(branch string, stack []string) int {
	for i, name := range stack {
		if name == branch {
			return i
		}
	}
	return -1 // Not found
}
