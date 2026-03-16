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

// isSorted returns true if commands are already in sort order.
func isSorted(commands []RenderCommand) bool {
	for i := 1; i < len(commands); i++ {
		if !CommandLessOrEqual(commands[i-1], commands[i]) {
			return false
		}
	}
	return true
}

// MergeSort sorts commands in-place using sortBuf as scratch space.
// Bottom-up merge sort: zero allocations after the sort buffer reaches high-water mark.
//
// This is the active sort used by Pipeline.Sort(). Benchmarked against RadixSort
// (2026-03-16) on 10k commands: merge sort matched or beat radix sort because the
// final scatter of large RenderCommand structs (~300B each) negates the advantage
// of sorting smaller (key, index) pairs. The O(n) already-sorted fast path also
// makes merge sort near-free for static scenes.
func MergeSort(commands []RenderCommand, sortBuf *[]RenderCommand) {
	n := len(commands)
	if n <= 1 {
		return
	}

	// Quick O(n) check: if commands are already sorted, skip the full sort.
	if isSorted(commands) {
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

// --- Radix sort (alternative, kept for future evaluation) ---
//
// Benchmarked 2026-03-16 on Apple M3 Max, 10k commands:
//   MergeSort:  ~875 µs    RadixSort:  ~920 µs
// Radix did not outperform merge sort because the final scatter of ~300B
// RenderCommand structs dominates. May become worthwhile if RenderCommand
// shrinks or command counts grow significantly (100k+).

// sortEntry pairs a packed sort key with the original command index.
type sortEntry struct {
	key   uint64
	index uint32
}

// packSortKey packs (RenderLayer, GlobalOrder, TreeOrder) into a uint64.
// Layout: [63:56] RenderLayer | [55:28] GlobalOrder (biased) | [27:0] TreeOrder (biased)
// Bias converts signed ints to unsigned for correct unsigned ordering.
// GlobalOrder and TreeOrder are each limited to 28 bits (±134,217,727).
func packSortKey(cmd *RenderCommand) uint64 {
	return uint64(cmd.RenderLayer)<<56 |
		uint64(uint32(cmd.GlobalOrder)+0x8000000)<<28 |
		uint64(uint32(cmd.TreeOrder)+0x8000000)
}

// RadixSort sorts commands using LSD radix sort on packed 64-bit keys.
// Operates on small (key, index) pairs during sorting, then scatters the
// full commands once at the end. Zero allocations after buffers reach
// high-water mark.
//
// Not currently used — see benchmark note above. To switch:
//
//	func (p *Pipeline) Sort() {
//	    RadixSort(p.Commands, &p.SortBuf, &p.SortKeys, &p.SortKeysBuf)
//	}
func RadixSort(commands []RenderCommand, sortBuf *[]RenderCommand, entries, entriesBuf *[]sortEntry) {
	n := len(commands)
	if n <= 1 {
		return
	}

	// Quick O(n) check: if commands are already sorted, skip everything.
	if isSorted(commands) {
		return
	}

	// Grow buffers to fit.
	if cap(*sortBuf) < n {
		*sortBuf = make([]RenderCommand, n)
	}
	*sortBuf = (*sortBuf)[:n]
	if cap(*entries) < n {
		*entries = make([]sortEntry, n)
	}
	*entries = (*entries)[:n]
	if cap(*entriesBuf) < n {
		*entriesBuf = make([]sortEntry, n)
	}
	*entriesBuf = (*entriesBuf)[:n]

	// Pack keys.
	src := *entries
	for i := range commands {
		src[i] = sortEntry{key: packSortKey(&commands[i]), index: uint32(i)}
	}
	dst := *entriesBuf

	// 8-pass LSD radix sort, 256 buckets per pass (one byte at a time).
	var counts [256]int
	for pass := uint(0); pass < 8; pass++ {
		shift := pass * 8

		// Count occurrences of each byte value.
		counts = [256]int{}
		for i := 0; i < n; i++ {
			digit := (src[i].key >> shift) & 0xFF
			counts[digit]++
		}

		// Skip pass if all entries have the same byte value.
		allSame := true
		for _, c := range &counts {
			if c != 0 && c != n {
				allSame = false
				break
			}
		}
		if allSame {
			continue
		}

		// Prefix sum.
		total := 0
		for i := range &counts {
			c := counts[i]
			counts[i] = total
			total += c
		}

		// Scatter.
		for i := 0; i < n; i++ {
			digit := (src[i].key >> shift) & 0xFF
			dst[counts[digit]] = src[i]
			counts[digit]++
		}

		src, dst = dst, src
	}

	// src now holds the sorted entries. Scatter commands into sortBuf, then copy back.
	buf := *sortBuf
	for i := 0; i < n; i++ {
		buf[i] = commands[src[i].index]
	}
	copy(commands, buf)
}
