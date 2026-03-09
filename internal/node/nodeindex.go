package node

import "strings"

// NodeIndex is an opt-in registry for looking up nodes by name or tag.
type NodeIndex struct {
	nameMap map[string][]*Node
	tagMap  map[string][]*Node
	nodeSet map[*Node]map[string]bool
}

// NewNodeIndex creates an empty NodeIndex.
func NewNodeIndex() *NodeIndex {
	return &NodeIndex{
		nameMap: make(map[string][]*Node),
		tagMap:  make(map[string][]*Node),
		nodeSet: make(map[*Node]map[string]bool),
	}
}

// Add registers a node in the index.
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
// is unregistered entirely.
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
func (idx *NodeIndex) FindByTag(pattern string) []*Node {
	inner, mode := ParsePattern(pattern)
	if mode == WildNone {
		return idx.tagMap[inner]
	}
	seen := make(map[*Node]bool)
	var result []*Node
	for tag, nodes := range idx.tagMap {
		if MatchPattern(tag, inner, mode) {
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

// FindByName returns the first registered node matching the name pattern.
func (idx *NodeIndex) FindByName(pattern string) *Node {
	inner, mode := ParsePattern(pattern)
	if mode == WildNone {
		if nodes := idx.nameMap[inner]; len(nodes) > 0 {
			return nodes[0]
		}
		return nil
	}
	for name, nodes := range idx.nameMap {
		if MatchPattern(name, inner, mode) && len(nodes) > 0 {
			return nodes[0]
		}
	}
	return nil
}

// FindAllByName returns all registered nodes matching the name pattern.
func (idx *NodeIndex) FindAllByName(pattern string) []*Node {
	inner, mode := ParsePattern(pattern)
	if mode == WildNone {
		return idx.nameMap[inner]
	}
	var result []*Node
	for name, nodes := range idx.nameMap {
		if MatchPattern(name, inner, mode) {
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

// CountByTag returns the number of nodes with the given tag.
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

// EachByTag iterates nodes with the given tag.
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

// --- Wildcard pattern matching ---

// WildMode describes the type of % wildcard match.
type WildMode byte

const (
	WildNone     WildMode = iota // exact match
	WildPrefix                   // "foo%" — starts with
	WildSuffix                   // "%foo" — ends with
	WildContains                 // "%foo%" — contains
)

// ParsePattern checks for % wildcards at the start/end of pattern.
func ParsePattern(pattern string) (inner string, mode WildMode) {
	startWild := len(pattern) > 0 && pattern[0] == '%'
	endWild := len(pattern) > 1 && pattern[len(pattern)-1] == '%'
	if !startWild && !endWild {
		return pattern, WildNone
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
		return s, WildContains
	case startWild:
		return s, WildSuffix
	default:
		return s, WildPrefix
	}
}

// MatchPattern tests s against a parsed pattern.
func MatchPattern(s, inner string, mode WildMode) bool {
	switch mode {
	case WildPrefix:
		return strings.HasPrefix(s, inner)
	case WildSuffix:
		return strings.HasSuffix(s, inner)
	case WildContains:
		return strings.Contains(s, inner)
	}
	return s == inner
}
