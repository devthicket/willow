package node

// AddChild appends child to this node's children.
// If child already has a parent, it is removed from that parent first.
func (n *Node) AddChild(child *Node) {
	if child == nil {
		panic("node: cannot add nil child")
	}
	if Debug {
		if DebugCheckDisposed != nil {
			DebugCheckDisposed(n, "AddChild (parent)")
			DebugCheckDisposed(child, "AddChild (child)")
		}
	}
	if IsAncestor(child, n) {
		panic("node: adding child would create a cycle")
	}
	if child.Parent != nil {
		child.Parent.removeChildByPtr(child)
	}
	child.Parent = n
	n.Children_ = append(n.Children_, child)
	n.ChildrenSorted = false
	if PropagateSceneFn != nil {
		PropagateSceneFn(child, n.Scene_)
	}
	MarkSubtreeDirty(child)
	invalidateAncestorCache(n)
	if Debug {
		if DebugCheckTreeDepth != nil {
			DebugCheckTreeDepth(child)
		}
		if DebugCheckChildCount != nil {
			DebugCheckChildCount(n)
		}
	}
}

// AddChildAt inserts child at the given index.
func (n *Node) AddChildAt(child *Node, index int) {
	if child == nil {
		panic("node: cannot add nil child")
	}
	if Debug {
		if DebugCheckDisposed != nil {
			DebugCheckDisposed(n, "AddChildAt (parent)")
			DebugCheckDisposed(child, "AddChildAt (child)")
		}
	}
	if IsAncestor(child, n) {
		panic("node: adding child would create a cycle")
	}
	if index < 0 || index > len(n.Children_) {
		panic("node: child index out of range")
	}
	if child.Parent != nil {
		child.Parent.removeChildByPtr(child)
	}
	child.Parent = n
	n.Children_ = append(n.Children_, nil)
	copy(n.Children_[index+1:], n.Children_[index:])
	n.Children_[index] = child
	n.ChildrenSorted = false
	if PropagateSceneFn != nil {
		PropagateSceneFn(child, n.Scene_)
	}
	MarkSubtreeDirty(child)
	invalidateAncestorCache(n)
	if Debug {
		if DebugCheckTreeDepth != nil {
			DebugCheckTreeDepth(child)
		}
		if DebugCheckChildCount != nil {
			DebugCheckChildCount(n)
		}
	}
}

// RemoveChild detaches child from this node.
func (n *Node) RemoveChild(child *Node) {
	if Debug && DebugCheckDisposed != nil {
		DebugCheckDisposed(n, "RemoveChild (parent)")
		DebugCheckDisposed(child, "RemoveChild (child)")
	}
	if child.Parent != n {
		panic("node: child's parent is not this node")
	}
	n.removeChildByPtr(child)
	child.Parent = nil
	if PropagateSceneFn != nil {
		PropagateSceneFn(child, nil)
	}
	n.ChildrenSorted = false
	MarkSubtreeDirty(child)
	invalidateAncestorCache(n)
}

// RemoveChildAt removes and returns the child at the given index.
func (n *Node) RemoveChildAt(index int) *Node {
	if Debug && DebugCheckDisposed != nil {
		DebugCheckDisposed(n, "RemoveChildAt")
	}
	if index < 0 || index >= len(n.Children_) {
		panic("node: child index out of range")
	}
	child := n.Children_[index]
	copy(n.Children_[index:], n.Children_[index+1:])
	n.Children_[len(n.Children_)-1] = nil
	n.Children_ = n.Children_[:len(n.Children_)-1]
	child.Parent = nil
	if PropagateSceneFn != nil {
		PropagateSceneFn(child, nil)
	}
	n.ChildrenSorted = false
	MarkSubtreeDirty(child)
	invalidateAncestorCache(n)
	return child
}

// RemoveFromParent detaches this node from its parent.
func (n *Node) RemoveFromParent() {
	if n.Parent == nil {
		return
	}
	n.Parent.RemoveChild(n)
}

// RemoveChildren detaches all children from this node.
func (n *Node) RemoveChildren() {
	for i, child := range n.Children_ {
		child.Parent = nil
		if PropagateSceneFn != nil {
			PropagateSceneFn(child, nil)
		}
		MarkSubtreeDirty(child)
		n.Children_[i] = nil
	}
	n.Children_ = n.Children_[:0]
	n.ChildrenSorted = true
	invalidateAncestorCache(n)
}

// Children returns the child list.
func (n *Node) Children() []*Node {
	return n.Children_
}

// NumChildren returns the number of children.
func (n *Node) NumChildren() int {
	return len(n.Children_)
}

// ChildAt returns the child at the given index.
func (n *Node) ChildAt(index int) *Node {
	return n.Children_[index]
}

// FindChild returns the first direct child matching the name pattern, or nil.
func (n *Node) FindChild(pattern string) *Node {
	inner, mode := ParsePattern(pattern)
	for _, c := range n.Children_ {
		if MatchPattern(c.Name, inner, mode) {
			return c
		}
	}
	return nil
}

// FindDescendant returns the first descendant (depth-first) matching the name pattern.
func (n *Node) FindDescendant(pattern string) *Node {
	inner, mode := ParsePattern(pattern)
	return n.findDescendant(inner, mode)
}

func (n *Node) findDescendant(inner string, mode WildMode) *Node {
	for _, c := range n.Children_ {
		if MatchPattern(c.Name, inner, mode) {
			return c
		}
		if found := c.findDescendant(inner, mode); found != nil {
			return found
		}
	}
	return nil
}

// SetChildIndex moves child to a new index among its siblings.
func (n *Node) SetChildIndex(child *Node, index int) {
	if child.Parent != n {
		panic("node: child's parent is not this node")
	}
	nc := len(n.Children_)
	if index < 0 || index >= nc {
		panic("node: child index out of range")
	}
	oldIndex := -1
	for i, c := range n.Children_ {
		if c == child {
			oldIndex = i
			break
		}
	}
	if oldIndex == index {
		return
	}
	if oldIndex < index {
		copy(n.Children_[oldIndex:], n.Children_[oldIndex+1:index+1])
	} else {
		copy(n.Children_[index+1:], n.Children_[index:oldIndex])
	}
	n.Children_[index] = child
	n.ChildrenSorted = false
}

// removeChildByPtr removes child from n.Children_ without clearing child.Parent.
func (n *Node) removeChildByPtr(child *Node) {
	for i, c := range n.Children_ {
		if c == child {
			copy(n.Children_[i:], n.Children_[i+1:])
			n.Children_[len(n.Children_)-1] = nil
			n.Children_ = n.Children_[:len(n.Children_)-1]
			return
		}
	}
}
