package input

import (
	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/types"
)

// nodeContainsLocal tests whether (lx, ly) falls inside a node's hit region.
// Uses HitShape if set; otherwise derives AABB from node dimensions.
// Containers with no HitShape are not hit-testable.
func nodeContainsLocal(n *node.Node, lx, ly float64) bool {
	if n.HitShape != nil {
		return n.HitShape.Contains(lx, ly)
	}
	if NodeDimensionsFn == nil {
		return false
	}
	w, h := NodeDimensionsFn(n)
	if w == 0 && h == 0 {
		return false
	}
	return lx >= 0 && lx <= w && ly >= 0 && ly <= h
}

// collectInteractable walks the tree in painter order (DFS, ZIndex-sorted),
// appending interactable leaf nodes to buf. Skips Visible=false or
// Interactable=false subtrees.
func collectInteractable(n *node.Node, buf []*node.Node) []*node.Node {
	if !n.Visible || !n.Interactable {
		return buf
	}

	if n.HitShape != nil || n.Type != types.NodeTypeContainer {
		buf = append(buf, n)
	}

	if len(n.Children_) == 0 {
		return buf
	}

	children := n.Children_
	if !n.ChildrenSorted {
		if RebuildSortedChildrenFn != nil {
			RebuildSortedChildrenFn(n)
		}
	}
	if n.SortedChildren != nil {
		children = n.SortedChildren
	}
	for _, child := range children {
		buf = collectInteractable(child, buf)
	}
	return buf
}

// hitTest finds the topmost interactable node at (worldX, worldY).
// Returns nil if nothing is hit.
func (m *Manager) hitTest(root *node.Node, worldX, worldY float64) *node.Node {
	m.HitBuf = collectInteractable(root, m.HitBuf[:0])

	for i := len(m.HitBuf) - 1; i >= 0; i-- {
		n := m.HitBuf[i]
		lx, ly := n.WorldToLocal(worldX, worldY)
		if nodeContainsLocal(n, lx, ly) {
			return n
		}
	}
	return nil
}
