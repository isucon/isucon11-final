package generate

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
)

//go:embed data/images/*
var f embed.FS

var (
	images     []*Image
	imageCount int
	next       int
)

type Image struct {
	format     string
	colorModel color.Model
	width      int
	height     int
	data       []byte
}

func init() {
	files, _ := f.ReadDir("data/images")
	imageCount = len(files)
	for _, file := range files {
		fmt.Println(file.Name())
		data, err := f.ReadFile("data/images/" + file.Name())
		if err != nil {
			panic(err)
		}
		r := bytes.NewReader(data)
		config, format, err := image.DecodeConfig(r)
		images = append(images, &Image{
			format:     format,
			colorModel: config.ColorModel,
			width:      config.Width,
			height:     config.Height,
			data:       data,
		})
	}
}

func cyclicGetImage() *Image {
	img := images[next]
	fmt.Println(img.height)
	next = (next + 1) % imageCount
	return img
}
