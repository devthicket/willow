package willow

import "strings"

// NodeIndex is an opt-in registry for looking up nodes by name or tag.
// Data lives here, not on the Node struct, so there is zero overhead
// for nodes that are never indexed.
type NodeIndex struct {
	nameMap map[string][]*Node        // node.Name → nodes
	tagMap  map[string][]*Node        // tag → nodes
	nodeSet map[*Node]map[string]bool // node → its tags (for efficient Remove)
}

// NewNodeIndex creates an empty NodeIndex.
func NewNodeIndex() *NodeIndex {
	return &NodeIndex{
		nameMap: make(map[string][]*Node),
		tagMap:  make(map[string][]*Node),
		nodeSet: make(map[*Node]map[string]bool),
	}
}

// Add registers a node in the index. The node is indexed by its Name
// for FindByName lookups. Optional tags are applied immediately.
// If the node is already registered, additional tags are appended.
func (idx *NodeIndex) Add(node *Node, tags ...string) {
	if _, exists := idx.nodeSet[node]; !exists {
		idx.nodeSet[node] = make(map[string]bool)
		idx.nameMap[node.Name] = append(idx.nameMap[node.Name], node)
	}
	for _, tag := range tags {
		if idx.nodeSet[node][tag] {
			continue
		}
		idx.nodeSet[node][tag] = true
		idx.tagMap[tag] = append(idx.tagMap[tag], node)
	}
}

// Remove removes tags from a node. If no tags are provided, the node
// is unregistered from the index entirely.
func (idx *NodeIndex) Remove(node *Node, tags ...string) {
	nodeTags, exists := idx.nodeSet[node]
	if !exists {
		return
	}
	if len(tags) == 0 {
		for tag := range nodeTags {
			idx.removeFromSlice(idx.tagMap, tag, node)
		}
		idx.removeFromSlice(idx.nameMap, node.Name, node)
		delete(idx.nodeSet, node)
		return
	}
	for _, tag := range tags {
		if !nodeTags[tag] {
			continue
		}
		delete(nodeTags, tag)
		idx.removeFromSlice(idx.tagMap, tag, node)
	}
}

// FindByTag returns all nodes with the given tag pattern.
// Supports % wildcards: "enemy" (exact), "enemy%" (starts with),
// "%boss" (ends with), "%ene%" (contains). Exact matches use O(1)
// map lookup; wildcards scan all tags.
//
// The returned slice MUST NOT be mutated by the caller. If you need to
// modify the result, copy it first.
func (idx *NodeIndex) FindByTag(pattern string) []*Node {
	inner, mode := parsePattern(pattern)
	if mode == wildNone {
		return idx.tagMap[inner]
	}
	seen := make(map[*Node]bool)
	var result []*Node
	for tag, nodes := range idx.tagMap {
		if matchPattern(tag, inner, mode) {
			for _, n := range nodes {
				if !seen[n] {
					seen[n] = true
					result = append(result, n)
				}
			}
		}
	}
	return result
}

// FindByName returns the first registered node matching the name pattern,
// or nil. Supports % wildcards (see FindByTag).
func (idx *NodeIndex) FindByName(pattern string) *Node {
	inner, mode := parsePattern(pattern)
	if mode == wildNone {
		if nodes := idx.nameMap[inner]; len(nodes) > 0 {
			return nodes[0]
		}
		return nil
	}
	for name, nodes := range idx.nameMap {
		if matchPattern(name, inner, mode) && len(nodes) > 0 {
			return nodes[0]
		}
	}
	return nil
}

// FindAllByName returns all registered nodes matching the name pattern.
// Supports % wildcards (see FindByTag).
func (idx *NodeIndex) FindAllByName(pattern string) []*Node {
	inner, mode := parsePattern(pattern)
	if mode == wildNone {
		return idx.nameMap[inner]
	}
	var result []*Node
	for name, nodes := range idx.nameMap {
		if matchPattern(name, inner, mode) {
			result = append(result, nodes...)
		}
	}
	return result
}

// FindByTags returns all nodes that have every one of the given tags.
func (idx *NodeIndex) FindByTags(tags ...string) []*Node {
	if len(tags) == 0 {
		return nil
	}
	candidates := idx.tagMap[tags[0]]
	if len(candidates) == 0 {
		return nil
	}
	var result []*Node
	for _, node := range candidates {
		nodeTags := idx.nodeSet[node]
		hasAll := true
		for _, tag := range tags[1:] {
			if !nodeTags[tag] {
				hasAll = false
				break
			}
		}
		if hasAll {
			result = append(result, node)
		}
	}
	return result
}

// CountByTag returns the number of nodes with the given tag without
// allocating a slice.
func (idx *NodeIndex) CountByTag(tag string) int {
	return len(idx.tagMap[tag])
}

// Each iterates all registered nodes. Return false from fn to stop early.
func (idx *NodeIndex) Each(fn func(n *Node) bool) {
	for node := range idx.nodeSet {
		if !fn(node) {
			return
		}
	}
}

// EachByTag iterates nodes with the given tag. Return false from fn to stop early.
func (idx *NodeIndex) EachByTag(tag string, fn func(n *Node) bool) {
	for _, node := range idx.tagMap[tag] {
		if !fn(node) {
			return
		}
	}
}

func (idx *NodeIndex) removeFromSlice(m map[string][]*Node, key string, node *Node) {
	nodes := m[key]
	for i, n := range nodes {
		if n == node {
			nodes[i] = nodes[len(nodes)-1]
			nodes[len(nodes)-1] = nil
			m[key] = nodes[:len(nodes)-1]
			if len(m[key]) == 0 {
				delete(m, key)
			}
			return
		}
	}
}

// wildMode describes the type of % wildcard match.
type wildMode byte

const (
	wildNone     wildMode = iota // exact match
	wildPrefix                   // "foo%" — starts with
	wildSuffix                   // "%foo" — ends with
	wildContains                 // "%foo%" — contains
)

// parsePattern checks for % wildcards at the start/end of pattern.
func parsePattern(pattern string) (inner string, mode wildMode) {
	startWild := len(pattern) > 0 && pattern[0] == '%'
	endWild := len(pattern) > 1 && pattern[len(pattern)-1] == '%'
	if !startWild && !endWild {
		return pattern, wildNone
	}
	s := pattern
	if startWild {
		s = s[1:]
	}
	if endWild && len(s) > 0 {
		s = s[:len(s)-1]
	}
	switch {
	case startWild && endWild:
		return s, wildContains
	case startWild:
		return s, wildSuffix
	default:
		return s, wildPrefix
	}
}

// matchPattern tests s against a parsed pattern.
func matchPattern(s, inner string, mode wildMode) bool {
	switch mode {
	case wildPrefix:
		return strings.HasPrefix(s, inner)
	case wildSuffix:
		return strings.HasSuffix(s, inner)
	case wildContains:
		return strings.Contains(s, inner)
	}
	return s == inner
}
