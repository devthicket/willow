package willow

// SetMask sets a mask node for this node. The mask node's alpha channel
// determines which parts of this node are visible. The mask node is NOT
// part of the scene tree  -  its transforms are relative to the masked node.
func (n *Node) SetMask(maskNode *Node) {
	n.mask = maskNode
	invalidateAncestorCache(n)
}

// ClearMask removes the mask from this node.
func (n *Node) ClearMask() {
	n.mask = nil
	invalidateAncestorCache(n)
}

// GetMask returns the current mask node, or nil if no mask is set.
func (n *Node) GetMask() *Node {
	return n.mask
}
