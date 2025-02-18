package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/cozy-creator/kritago/pkg/document"
	"github.com/cozy-creator/kritago/pkg/layers"
	"github.com/cozy-creator/kritago/pkg/shapes"
)

func main() {
	// Create a new document.
	doc := document.NewKritaDocument(1024, 1024)

	// Add a text layer.
	textStyle := layers.NewTextStyle()
	doc.AddTextLayer("Hello, Krita!\nThis is a text layer.", "Text Layer", 10, 10, 255, textStyle)

	// Add a shape layer with a rectangle.
	shapeStyle := shapes.NewShapeStyle()
	rect := &shapes.Rectangle{
		BaseShape: shapes.BaseShape{Style: shapeStyle, Transform: ""},
		X:         50, Y: 50, Width: 200, Height: 100,
	}
	doc.AddShapeLayer([]shapes.Shape{rect}, "Shape Layer", 0, 0, 255, &shapeStyle)

	// (Optional) Create and add an image layer.
	img := createDummyImage(1024, 1024)
	doc.AddImageLayer(img, "dummy.png", "Image Layer", 0, 0, 255)

	// Save the document.
	if err := doc.Save("output.kra"); err != nil {
		fmt.Println("Error saving document:", err)
	} else {
		fmt.Println("Krita document saved as output.kra")
	}
}

// createDummyImage creates a dummy RGBA image.
func createDummyImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{200, 200, 200, 255}}, image.Point{}, draw.Src)
	f, _ := os.Create("dummy.png")
	defer f.Close()
	png.Encode(f, img)
	return img
}
