package render

// CommandLessOrEqual returns true if a should sort before or at the same position as b.
// Using <= for TreeOrder ensures stability.
func CommandLessOrEqual(a, b RenderCommand) bool {
	if a.RenderLayer != b.RenderLayer {
		return a.RenderLayer < b.RenderLayer
	}
	if a.GlobalOrder != b.GlobalOrder {
		return a.GlobalOrder < b.GlobalOrder
	}
	return a.TreeOrder <= b.TreeOrder
}

// MergeSort sorts commands in-place using sortBuf as scratch space.
// Bottom-up merge sort: zero allocations after the sort buffer reaches high-water mark.
func MergeSort(commands []RenderCommand, sortBuf *[]RenderCommand) {
	n := len(commands)
	if n <= 1 {
		return
	}

	// Quick O(n) check: if commands are already sorted, skip the full sort.
	sorted := true
	for i := 1; i < n; i++ {
		if !CommandLessOrEqual(commands[i-1], commands[i]) {
			sorted = false
			break
		}
	}
	if sorted {
		return
	}

	if cap(*sortBuf) < n {
		*sortBuf = make([]RenderCommand, n)
	}
	*sortBuf = (*sortBuf)[:n]

	a := commands
	b := *sortBuf
	swapped := false

	for width := 1; width < n; width *= 2 {
		for i := 0; i < n; i += 2 * width {
			lo := i
			mid := lo + width
			if mid > n {
				mid = n
			}
			hi := lo + 2*width
			if hi > n {
				hi = n
			}
			mergeRun(a, b, lo, mid, hi)
		}
		a, b = b, a
		swapped = !swapped
	}

	if swapped {
		copy(commands, *sortBuf)
	}
}

// mergeRun merges two sorted runs [lo, mid) and [mid, hi) from src into dst.
func mergeRun(src, dst []RenderCommand, lo, mid, hi int) {
	i, j, k := lo, mid, lo
	for i < mid && j < hi {
		if CommandLessOrEqual(src[i], src[j]) {
			dst[k] = src[i]
			i++
		} else {
			dst[k] = src[j]
			j++
		}
		k++
	}
	for i < mid {
		dst[k] = src[i]
		i++
		k++
	}
	for j < hi {
		dst[k] = src[j]
		j++
		k++
	}
}
