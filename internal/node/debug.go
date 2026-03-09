package node

import (
	"fmt"
	"os"
)

const debugMaxTreeDepth = 32
const debugMaxChildCount = 1000

// DefaultDebugCheckDisposed panics if a disposed node is used in a tree operation.
func DefaultDebugCheckDisposed(n *Node, op string) {
	if n.Disposed {
		panic(fmt.Sprintf("willow debug: %s on disposed node %q (ID was %d)", op, n.Name, n.ID))
	}
}

// DefaultDebugCheckTreeDepth warns on stderr if tree depth exceeds threshold.
func DefaultDebugCheckTreeDepth(n *Node) {
	depth := 0
	for p := n; p != nil; p = p.Parent {
		depth++
	}
	if depth > debugMaxTreeDepth {
		_, _ = fmt.Fprintf(os.Stderr, "[willow] warning: tree depth %d exceeds %d (node %q)\n",
			depth, debugMaxTreeDepth, n.Name)
	}
}

// DefaultDebugCheckChildCount warns on stderr if a node has more than 1000 children.
func DefaultDebugCheckChildCount(n *Node) {
	if len(n.Children_) > debugMaxChildCount {
		_, _ = fmt.Fprintf(os.Stderr, "[willow] warning: node %q has %d children (threshold %d)\n",
			n.Name, len(n.Children_), debugMaxChildCount)
	}
}

func init() {
	DebugCheckDisposed = DefaultDebugCheckDisposed
	DebugCheckTreeDepth = DefaultDebugCheckTreeDepth
	DebugCheckChildCount = DefaultDebugCheckChildCount
}
