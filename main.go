package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"io"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/plotinum/plotter"
	"gopkg.in/alecthomas/kingpin.v1"
)

const (
	ESC_HIDE_CURSOR      = "\033[?25l"
	ESC_SHOW_CURSOR      = "\033[?25h"
	ESC_SAVE_POSITION    = "\0337" // "\033[s"
	ESC_RESTORE_POSITION = "\0338" // "\033[u"
	ESC_CLEAR_SCREEN     = "\033[H\033[2J"
)

type renderer func(xy plotter.XYs) (image.Image, error)

type valuer func() (plotter.Values, error)

var (
	optAnimate           bool
	optIgnoreParseErrors bool
	optInlineImages      bool
	optStream            bool
	optCharGeo           *charGeo
	optCharWidth         = 7
	optCharHeight        = 17
	optRows              = 3
	optMaxCols           = 80
	optVerbose           = 1
	optRenderer          *string
	optValues            []string
)

var renderers = map[string]renderer{}

func main() {
	optCharGeo = new(charGeo)
	values := make(plotter.Values, 0)

	kingpin.Flag("stream", "stream").Short('s').BoolVar(&optStream)
	kingpin.Flag("animate", "start animation").BoolVar(&optAnimate)
	kingpin.Flag("lazy", "ignore parse errors").BoolVar(&optIgnoreParseErrors)
	kingpin.Flag("char-size", "Pixel size of a single character. Can also be set via env ITERM_CHARACTER_SIZE. The default 7:17 corresponds to 12p Monaco.").
		OverrideDefaultFromEnvar("ITERM_CHARACTER_SIZE").
		Default("7:17").
		SetValue(optCharGeo)
	kingpin.Flag("rows", "height in number of rows").Default("3").IntVar(&optRows)

	availRenderers := []string{}
	for k := range renderers {
		availRenderers = append(availRenderers, k)
	}
	sort.StringSlice(availRenderers).Sort()
	rendererHelp := fmt.Sprintf("available renderers: %s", strings.Join(availRenderers, ", "))
	rendererName := kingpin.Flag("renderer", rendererHelp).Default("sparks").Enum(availRenderers...)

	kingpin.Arg("values", "Numeric values to render. Can also be read from stdin.").SetValue((*argValues)(&values))
	kingpin.Parse()

	optCharHeight = optCharGeo.Height
	optCharWidth = optCharGeo.Width

	optInlineImages = IsTerminal(os.Stdout)

	if IsTerminal(os.Stdin) && len(values) == 0 {
		kingpin.Usage()
		return
	}

	var renderFn = renderers[*rendererName]

	b := bufio.NewReader(os.Stdin)
	var valuesProvider valuer
	if !IsTerminal(os.Stdin) {
		firstChar, _ := b.Peek(1)
		if string(firstChar) == "[" {
			valuesProvider = valuerForJsonArray(b)
		} else if string(firstChar) == "{" {
			valuesProvider = valuerForCloudWatchJson(b)
		} else {
			valuesProvider = valuerForPlainNumbers(b)
		}
	} else {
		// null valuer
		valuesProvider = func() (plotter.Values, error) {
			return nil, io.EOF
		}
	}
	var drawer animationDrawer
	if IsTerminal(os.Stdout) {
		drawer = &iTermAnimationDrawer{
			out: os.Stdout,
		}
	} else {
		drawer = &gifAnimationDrawer{
			out: os.Stdout,
		}
	}

	var err error
	if optStream {
		optInlineImages = true
		if IsTerminal(os.Stdin) {
			err = fmt.Errorf("expected pipe on stdin when using stream option")
		}
		err = renderAnimated(drawer, valuesProvider, renderFn, optMaxCols)
	} else {
		if !IsTerminal(os.Stdin) {
			values, err = appendAllValues(values, valuesProvider, optMaxCols)
		}
		if err == nil {
			if optAnimate {
				optInlineImages = true
				times := 2
				t := 0
				i := 0
				err = renderAnimated(drawer,
					func() (plotter.Values, error) {
						time.Sleep(100 * time.Millisecond)
						for t < times {
							if i < len(values) {
								v := append(values[i:], values[0:i]...)
								i++
								return v, nil
							}
							t++
						}
						if t == times {
							t++
							return values, nil
						}
						return nil, io.EOF
					}, renderFn, len(values))
			} else {
				var img image.Image
				img, err = renderFn(Values2XYs(values))
				if err == nil {
					renderImg(img, os.Stdout)
				}
			}
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}

func appendAllValues(values plotter.Values, valuesProvider valuer, window int) (plotter.Values, error) {
	for {
		v, err := valuesProvider()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		values = append(values, v...)
		if m := len(values); m > window {
			values = values[m-window : m]
		}
	}
	return values, nil
}

type animationDrawer interface {
	Begin() error
	DrawFrame(image.Image) error
	End() error
}

type iTermAnimationDrawer struct {
	out io.Writer
}

func (i *iTermAnimationDrawer) Begin() error {
	io.WriteString(i.out, ESC_HIDE_CURSOR)
	io.WriteString(i.out, strings.Repeat("\n", optRows))
	return nil
}

func (i *iTermAnimationDrawer) End() error {
	io.WriteString(i.out, ESC_SHOW_CURSOR)
	return nil
}

func (i *iTermAnimationDrawer) DrawFrame(img image.Image) error {
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "\033[%dA", optRows)
	itermImg := &ITermImage{img}
	_, err := itermImg.WriteTo(b)
	if err != nil {
		return err
	}
	b.WriteTo(i.out)
	return nil
}

type gifAnimationDrawer struct {
	out    io.Writer
	last   *time.Time
	frames []image.Image
	delays []int
}

func (g *gifAnimationDrawer) Begin() error {
	fmt.Fprint(os.Stderr, "Start buffering to generate animated GIF...\n")

	return nil
}
func (g *gifAnimationDrawer) End() error {
	fmt.Fprint(os.Stderr, "Writing animated GIF...")

	var pFrames []*image.Paletted
	if len(g.frames) > 0 {
		b := g.frames[len(g.frames)-1].Bounds()
		for _, img := range g.frames {
			pimg := image.NewPaletted(b, palette.Plan9)
			draw.FloydSteinberg.Draw(pimg, b, img, image.ZP)
			pFrames = append(pFrames, pimg)
		}
	}

	return gif.EncodeAll(g.out, &gif.GIF{
		Image: pFrames,
		Delay: append(g.delays, 0),
	})
}
func (g *gifAnimationDrawer) DrawFrame(img image.Image) error {

	g.frames = append(g.frames, img)

	curr := time.Now()
	if g.last != nil {
		delay := Centiseconds(curr.Sub(*g.last))
		//log.Printf("DrawFrame: delay %#v", delay)
		g.delays = append(g.delays, delay)
	}
	// log.Printf("Buffer frame %d", len(g.frames))
	hour := []string{"|", "/", "-", "\\"}
	i := len(g.frames)
	fmt.Fprintf(os.Stderr, "\033[0K%s buffered %d frames\r", hour[i%4], i)

	g.last = &curr
	return nil
}

func Centiseconds(t time.Duration) int {
	return int(float64(10E-8) * float64(t.Nanoseconds()))
}

func renderAnimated(drawer animationDrawer, valuesProvider valuer, renderFn renderer, window int) error {
	values := plotter.Values{}

	redrawCh := time.Tick(100 * time.Millisecond)
	errorCh := make(chan error)
	valuesCh := make(chan plotter.Values)
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	go func() {
		for {
			v, err := valuesProvider()
			if v != nil {
				valuesCh <- v
			}
			if err != nil {
				errorCh <- err
				break
			}
		}
	}()

	drawer.Begin()

	valuesChanged := false
	var lastError error

	redraw := func() error {
		if valuesChanged {
			img, err := renderFn(Values2XYs(values))
			if err != nil {
				return err
			}
			if err := drawer.DrawFrame(img); err != nil {
				return err
			}
			valuesChanged = false
		}
		return nil
	}

loop:
	for {
		select {
		case <-signalCh:
			//log.Print("reveived signal: ", s)
			break loop

		case v := <-valuesCh:
			//log.Print("reveived values: ", v)
			values = append(values, v...)
			if m := len(values); m > window {
				values = values[m-window : m]
			}
			valuesChanged = true
			//log.Print("all values: ", values)

		case err := <-errorCh:
			//log.Print("reveived error: ", err)
			if err != io.EOF {
				lastError = err
				// os.Stdout.WriteString(ESC_CLEAR_SCREEN)
				// log.Print(err)
			} else {
				// reached the end.. last trigger last redraw
				lastError = redraw()
			}
			break loop

		case <-redrawCh:
			//log.Print("reveived redraw: ", r)
			if err := redraw(); err != nil {
				lastError = err
				break loop
			}
		}
	}

	if err := drawer.End(); err != nil && lastError == nil {
		lastError = err
	}

	return lastError
}

func valuerForPlainNumbers(in io.Reader) valuer {
	lineScanner := bufio.NewScanner(in)
	lineScanner.Split(bufio.ScanLines)

	return func() (plotter.Values, error) {
		if lineScanner.Scan() {
			values := plotter.Values{}
			line := lineScanner.Text()
			wordScanner := bufio.NewScanner(strings.NewReader(line))
			wordScanner.Split(bufio.ScanWords)
			for wordScanner.Scan() {
				val := wordScanner.Text()
				f, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return nil, err
				}
				values = append(values, f)
			}
			err := lineScanner.Err()
			return values, err
		}
		return nil, io.EOF
	}
}

func valuerForJsonArray(in io.Reader) valuer {
	dec := json.NewDecoder(in)
	return func() (plotter.Values, error) {
		var m plotter.Values
		err := dec.Decode(&m)
		if err != nil {
			return nil, err
		}
		return m, nil
	}
}

func valuerForCloudWatchJson(in io.Reader) valuer {
	dec := json.NewDecoder(in)
	type CloudWatchDatapoint struct {
		Timestamp   string
		Sum         *float64
		Maximum     *float64
		Minimum     *float64
		SampleCount *float64
		Average     *float64
		Unit        string
	}
	type CloudWatchData struct {
		Label      string
		Datapoints []*CloudWatchDatapoint
	}
	return func() (plotter.Values, error) {
		var m *CloudWatchData
		err := dec.Decode(&m)
		if err != nil {
			return nil, err
		}
		values := plotter.Values{}
		for _, d := range m.Datapoints {
			if d.Sum != nil {
				values = append(values, *d.Sum)
			} else if d.Average != nil {
				values = append(values, *d.Average)
			} else if d.Minimum != nil {
				values = append(values, *d.Minimum)
			} else if d.Maximum != nil {
				values = append(values, *d.Maximum)
			} else if d.SampleCount != nil {
				values = append(values, *d.SampleCount)
			}
		}
		return values, nil
	}
}

type charGeo struct {
	Width  int
	Height int
}

func (c *charGeo) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected WIDTH:HEIGHT got '%s'", value)
	}
	width, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || width <= 0 {
		return fmt.Errorf("expected WIDTH:HEIGHT, WIDTH must be a positive number, got '%s'", parts[0])
	}
	height, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || height <= 0 {
		return fmt.Errorf("expected WIDTH:HEIGHT, HEIGHT must be a positive number, got '%s'", parts[1])
	}
	c.Width = int(width)
	c.Height = int(height)
	return nil
}

func (c *charGeo) String() string {
	return fmt.Sprintf("%d:%d", c.Width, c.Height)
}

type argValues plotter.Values

func (i *argValues) Set(value string) error {
	if !IsTerminal(os.Stdin) {
		return fmt.Errorf("command line values not allowed when reading from stdin")
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("expected NUMERIC VALUE got '%s'", value)
	}
	*i = append(*i, f)
	return nil
}

func (i *argValues) String() string {
	return ""
}

func (i *argValues) IsCumulative() bool {
	return true
}

func Values2XYs(values plotter.Values) plotter.XYs {
	XYs := make(plotter.XYs, 0)
	for i, v := range values {
		XYs = append(XYs, plotter.XYs{{float64(i), v}}...)
	}
	return XYs
}

func init() {
	renderers["sparks"] = plotSparks
}

func plotSparks(xys plotter.XYs) (image.Image, error) {

	border := 4
	if optRows == 1 {
		border = 1
	} else if optRows == 2 {
		border = 2
	}
	height := optCharHeight * optRows
	width := (optCharWidth * 2) + (optCharWidth * len(xys))
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	_, _, ymin, ymax := plotter.XYRange(xys)
	ymin = 0

	for i, xy := range xys {

		//xf := (xy.X - xmin) / xmax
		yf := (xy.Y - ymin) / ymax
		hm := height - int(float64(height-(border*2))*yf) - border
		h0 := height - border

		w := ((i + 1) * optCharWidth)
		for h := h0; h >= hm; h = h - 2 {
			var c color.RGBA
			if h == hm || h == hm+1 {
				c = color.RGBA{0, 255, 0, 255}
			} else if h == h0 {
				c = color.RGBA{0, 128, 0, 255}
			} else {
				p := float64(h-hm)/float64(h0-hm)*0.5 + 0.3
				g := uint8(float64(255) - (p * float64(255)))
				c = color.RGBA{0, g, 0, 255}
			}
			for j := 0; j < (optCharWidth - 2); j++ {
				img.SetRGBA(w+j, h, c)
			}
		}
	}

	return img, nil
}

func renderImg(img image.Image, out io.Writer) error {
	if optInlineImages {
		itermImg := &ITermImage{img}
		if _, err := itermImg.WriteTo(out); err != nil {
			return err
		}
	} else {
		b := new(bytes.Buffer)
		err := png.Encode(b, img)
		if err != nil {
			return err
		}
		if _, err := b.WriteTo(out); err != nil {
			return err
		}
	}
	return nil
}
