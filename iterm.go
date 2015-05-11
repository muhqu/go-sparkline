package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
)

type FileDescriptor interface {
	Stat() (fi os.FileInfo, err error)
}

func IsTerminal(fd FileDescriptor) bool {
	fi, _ := fd.Stat()
	return (fi.Mode() & os.ModeCharDevice) != 0
}

type ITermImage struct {
	img image.Image
}

// WriteTo implements the io.WriterTo interface, writing an iTerm 1337 escaped .
func (i *ITermImage) WriteTo(w io.Writer) (int64, error) {
	str, err := ITermEncodePNGToString(i.img, "[iTerm Image]")
	if err != nil {
		return 0, err
	}
	n, err := w.Write([]byte(str))
	return int64(n), err
}

func (i *ITermImage) String() string {
	str, err := ITermEncodePNGToString(i.img, "[iTerm Image]")
	if err != nil {
		return err.Error()
	}
	return str
}

func ITermEncodePNGToString(img image.Image, alt string) (str string, err error) {
	b := new(bytes.Buffer)
	err = png.Encode(b, img)
	if err != nil {
		return
	}
	bytes := b.Bytes()
	base64str := base64.StdEncoding.EncodeToString(bytes)
	str = fmt.Sprintf("\033]1337;File=inline=1;size=%d:%s\a%s\n", len(bytes), base64str, alt)
	return
}
