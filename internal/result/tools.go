package result

// CountToolTransitions counts context switches between consecutive tools.
// The first tool is not a transition. One tool throughout is 0.
func CountToolTransitions(tools []string) int {
	n := 0
	prev := ""
	for _, tool := range tools {
		if tool == "" {
			continue
		}
		if prev != "" && tool != prev {
			n++
		}
		prev = tool
	}
	return n
}
