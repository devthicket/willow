package core

import "github.com/devthicket/willow/internal/node"

// PropagateScene recursively sets the scene back-pointer on n and all
// descendants, firing node lifecycle hooks on the scene-membership transition:
// OnEnter/OnReady when entering the live tree (top-down), OnExit when leaving
// (bottom-up).
func PropagateScene(n *node.Node, s any) {
	old := n.Scene_
	if old == s {
		return
	}
	n.Scene_ = s
	entering := old == nil && s != nil
	exiting := old != nil && s == nil
	// Enter is top-down: a parent enters before its children.
	if entering {
		node.FireEnter(n)
	}
	for _, child := range n.Children_ {
		PropagateScene(child, s)
	}
	// Exit is bottom-up: children leave before their parent.
	if exiting {
		node.FireExit(n)
	}
}
