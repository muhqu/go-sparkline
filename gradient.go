package main

import "github.com/lucasb-eyer/go-colorful"

type GradientTable []struct {
	Col colorful.Color
	Pos float64
}

var BrightColorGradient = GradientTable{
	{MustParseHex("#EC0047"), 0.0},
	{MustParseHex("#CA001B"), 0.1},
	{MustParseHex("#EF270C"), 0.2},
	{MustParseHex("#FA6A09"), 0.3},
	{MustParseHex("#FEB10D"), 0.4},
	{MustParseHex("#FFFF14"), 0.5},
	{MustParseHex("#CEF82C"), 0.6},
	{MustParseHex("#39DC20"), 0.7},
	{MustParseHex("#26CA75"), 0.8},
	{MustParseHex("#1571F4"), 0.9},
	{MustParseHex("#3900D9"), 1.0},
	// {MustParseHex("#9e0142"), 0.0},
	// {MustParseHex("#d53e4f"), 0.1},
	// {MustParseHex("#f46d43"), 0.2},
	// {MustParseHex("#fdae61"), 0.3},
	// {MustParseHex("#fee090"), 0.4},
	// {MustParseHex("#ffffbf"), 0.5},
	// {MustParseHex("#e6f598"), 0.6},
	// {MustParseHex("#abdda4"), 0.7},
	// {MustParseHex("#66c2a5"), 0.8},
	// {MustParseHex("#3288bd"), 0.9},
	// {MustParseHex("#5e4fa2"), 1.0},
}

// This is the meat of the gradient computation. It returns a HCL-blend between
// the two colors around `t`.
// Note: It relies heavily on the fact that the gradient keypoints are sorted.
func (self GradientTable) GetInterpolatedColorFor(t float64) colorful.Color {
	for i := 0; i < len(self)-1; i++ {
		c1 := self[i]
		c2 := self[i+1]
		if c1.Pos <= t && t <= c2.Pos {
			// We are in between c1 and c2. Go blend them!
			t := (t - c1.Pos) / (c2.Pos - c1.Pos)
			return c1.Col.BlendHcl(c2.Col, t).Clamped()
		}
	}

	// Nothing found? Means we're at (or past) the last gradient keypoint.
	return self[len(self)-1].Col
}

// This is a very nice thing Golang forces you to do!
// It is necessary so that we can write out the literal of the colortable below.
func MustParseHex(s string) colorful.Color {
	c, err := colorful.Hex(s)
	if err != nil {
		panic("MustParseHex: " + err.Error())
	}
	return c
}
