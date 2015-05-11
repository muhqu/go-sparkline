package main

import (
	"image"
	"image/color"

	"code.google.com/p/plotinum/plot"
	"code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/vg"
	"code.google.com/p/plotinum/vg/vgimg"
)

func init() {
	renderers["line"] = plotLine
	renderers["vlines"] = plotVLines
}

func plotVLines(xy plotter.XYs) (image.Image, error) {

	p, err := plot.New()
	if err != nil {
		return nil, err
	}
	p.HideAxes()
	p.BackgroundColor = &color.RGBA{0, 0, 0, 255}

	s, err := NewSparkLines(xy)
	if err != nil {
		return nil, err
	}
	s.Color = &color.RGBA{0, 255, 0, 128}
	p.Add(s)

	// Draw the plot to an in-memory image.
	// _, rows, _ := terminal.GetSize(0)
	charWidth := optCharWidth
	charHeight := optCharHeight
	//width := cols * charWidth
	height := optRows * charHeight

	img := image.NewRGBA(image.Rect(0, 0, 5+(len(xy)*charWidth), height))
	canvas := vgimg.NewImage(img)
	da := plot.MakeDrawArea(canvas)
	p.Draw(da)

	return img, nil
}

func plotLine(xy plotter.XYs) (image.Image, error) {

	p, err := plot.New()
	if err != nil {
		return nil, err
	}
	p.HideAxes()
	p.BackgroundColor = &color.RGBA{0, 0, 0, 255}

	//s, err := NewSparkLines(xy)
	s, err := plotter.NewLine(xy)
	if err != nil {
		return nil, err
	}
	s.Color = &color.RGBA{0, 255, 0, 128}
	p.Add(s)

	// Draw the plot to an in-memory image.
	// _, rows, _ := terminal.GetSize(0)
	charWidth := optCharWidth
	charHeight := optCharHeight
	//width := cols * charWidth
	height := optRows * charHeight

	img := image.NewRGBA(image.Rect(0, 0, 5+(len(xy)*charWidth), height))
	canvas := vgimg.NewImage(img)
	da := plot.MakeDrawArea(canvas)
	p.Draw(da)

	return img, nil
}

func NewSparkLines(xy plotter.XYs) (*SparkLines, error) {
	s := new(SparkLines)
	s.XYs = xy
	return s, nil
}

type SparkLines struct {
	XYs   plotter.XYs
	Color color.Color
}

func (s *SparkLines) Plot(da plot.DrawArea, plt *plot.Plot) {
	trX, trY := plt.Transforms(&da)

	w := vg.Length(1)

	da.SetLineWidth(w)

	_, _, ymin, ymax := s.DataRange()

	for _, d := range s.XYs {
		perc := float64(d.Y-ymin) / float64(ymax-ymin)
		c := BrightColorGradient.GetInterpolatedColorFor((perc*-1+1)*0.5 + 0.6)
		da.SetColor(c)

		// Transform the data x, y coordinate of this bubble
		// to the corresponding drawing coordinate.
		x := trX(d.X)
		y := trY(d.Y * 0.9)

		//rad := vg.Length(10)
		var p vg.Path
		p.Move(x-w, y)
		p.Line(x-w, 0)
		//p.Close()
		da.Stroke(p)

		//da.StrokeLine2(*sty, x, 0, x, y)
	}
}

func (s *SparkLines) DataRange() (xmin, xmax, ymin, ymax float64) {
	return plotter.XYRange(s.XYs)
}
