package node

import "math"

// IdentityTransform is the identity affine matrix.
var IdentityTransform = [6]float64{1, 0, 0, 1, 0, 0}

// AnyTransformDirty is set when any node's transform or alpha is dirtied.
// The scene resets this after the first UpdateWorldTransform pass and checks
// it before the second pass to skip a redundant full tree walk.
var AnyTransformDirty bool

// ComputeLocalTransform computes the local affine matrix from the node's
// transform properties. Returns [a, b, c, d, tx, ty].
func ComputeLocalTransform(n *Node) [6]float64 {
	sx := n.ScaleX_
	sy := n.ScaleY_

	if n.Rotation_ == 0 && n.SkewX_ == 0 && n.SkewY_ == 0 {
		return [6]float64{sx, 0, 0, sy, -n.PivotX_*sx + n.X_, -n.PivotY_*sy + n.Y_}
	}

	var sin, cos float64
	if n.Rotation_ != 0 {
		sin, cos = math.Sincos(n.Rotation_)
	} else {
		cos = 1
	}

	var tanSkewX, tanSkewY float64
	if n.SkewX_ != 0 {
		tanSkewX = math.Tan(n.SkewX_)
	}
	if n.SkewY_ != 0 {
		tanSkewY = math.Tan(n.SkewY_)
	}

	a := sx
	b := tanSkewY * sx
	c := tanSkewX * sy
	d := sy

	px := n.PivotX_
	py := n.PivotY_
	preTx := -px*sx - tanSkewX*py*sy
	preTy := -tanSkewY*px*sx - py*sy

	ra := cos*a - sin*b
	rb := sin*a + cos*b
	rc := cos*c - sin*d
	rd := sin*c + cos*d
	rtx := cos*preTx - sin*preTy
	rty := sin*preTx + cos*preTy

	return [6]float64{ra, rb, rc, rd, rtx + n.X_, rty + n.Y_}
}

// MultiplyAffine multiplies two 2D affine matrices: result = parent * child.
func MultiplyAffine(p, c [6]float64) [6]float64 {
	return [6]float64{
		p[0]*c[0] + p[2]*c[1],
		p[1]*c[0] + p[3]*c[1],
		p[0]*c[2] + p[2]*c[3],
		p[1]*c[2] + p[3]*c[3],
		p[0]*c[4] + p[2]*c[5] + p[4],
		p[1]*c[4] + p[3]*c[5] + p[5],
	}
}

// InvertAffine computes the inverse of a 2D affine matrix.
func InvertAffine(m [6]float64) [6]float64 {
	det := m[0]*m[3] - m[2]*m[1]
	if det > -1e-12 && det < 1e-12 {
		return IdentityTransform
	}
	invDet := 1.0 / det
	a := m[3] * invDet
	b := -m[1] * invDet
	c := -m[2] * invDet
	d := m[0] * invDet
	return [6]float64{
		a, b, c, d,
		-(a*m[4] + c*m[5]),
		-(b*m[4] + d*m[5]),
	}
}

// TransformPoint applies an affine matrix to a point.
func TransformPoint(m [6]float64, x, y float64) (float64, float64) {
	return m[0]*x + m[2]*y + m[4], m[1]*x + m[3]*y + m[5]
}

// UpdateWorldTransform recomputes a node's worldTransform and worldAlpha.
func UpdateWorldTransform(n *Node, parentTransform [6]float64, parentAlpha float64, parentRecomputed bool, parentAlphaChanged bool) {
	if !n.Visible_ {
		return
	}
	recompute := n.TransformDirty || parentRecomputed
	alphaChanged := n.AlphaDirty || parentAlphaChanged
	if recompute {
		local := ComputeLocalTransform(n)
		n.WorldTransform = MultiplyAffine(parentTransform, local)
		n.WorldAlpha = parentAlpha * n.Alpha_
		n.TransformDirty = false
		n.AlphaDirty = false
	} else if alphaChanged {
		n.WorldAlpha = parentAlpha * n.Alpha_
		n.AlphaDirty = false
	}

	for _, child := range n.Children_ {
		UpdateWorldTransform(child, n.WorldTransform, n.WorldAlpha, recompute, recompute || alphaChanged)
	}
}

// --- Transform property setters ---

func (n *Node) SetPosition(x, y float64) {
	n.X_ = x
	n.Y_ = y
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

func (n *Node) SetX(x float64) {
	n.X_ = x
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

func (n *Node) SetY(y float64) {
	n.Y_ = y
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

func (n *Node) SetScale(sx, sy float64) {
	n.ScaleX_ = sx
	n.ScaleY_ = sy
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

func (n *Node) SetRotation(r float64) {
	n.Rotation_ = r
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

func (n *Node) SetSkew(sx, sy float64) {
	n.SkewX_ = sx
	n.SkewY_ = sy
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

func (n *Node) SetPivot(px, py float64) {
	n.PivotX_ = px
	n.PivotY_ = py
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

func (n *Node) Pivot() (float64, float64) {
	return n.PivotX_, n.PivotY_
}

func (n *Node) SetAlpha(a float64) {
	n.Alpha_ = a
	n.AlphaDirty = true
	invalidateAncestorCache(n)
}

// --- Coordinate conversion ---

func (n *Node) WorldToLocal(wx, wy float64) (lx, ly float64) {
	inv := InvertAffine(n.WorldTransform)
	return TransformPoint(inv, wx, wy)
}

func (n *Node) LocalToWorld(lx, ly float64) (wx, wy float64) {
	return TransformPoint(n.WorldTransform, lx, ly)
}
