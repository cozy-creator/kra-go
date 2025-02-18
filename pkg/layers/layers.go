package layers

import (
	"github.com/google/uuid"
	"github.com/cozy-creator/kritago/pkg/shapes"
)

// TextStyle holds text styling options.
type TextStyle struct {
	FontFamily       string
	FontSize         int
	FillColor        string
	StrokeColor      string
	StrokeWidth      int
	StrokeOpacity    float64
	LetterSpacing    int
	WordSpacing      int
	TextAlign        string
	TextAlignLast    string
	LineHeight       float64
	UseRichText      bool
	TextRendering    string
	DominantBaseline string
	TextAnchor       string
	PaintOrder       string
	StrokeLinecap    string
	StrokeLinejoin   string
}

// NewTextStyle returns a TextStyle with default values.
func NewTextStyle() *TextStyle {
	return &TextStyle{
		FontFamily:       "Segoe UI",
		FontSize:         12,
		FillColor:        "#000000",
		StrokeColor:      "#000000",
		StrokeWidth:      0,
		StrokeOpacity:    0,
		LetterSpacing:    0,
		WordSpacing:      0,
		TextAlign:        "start",
		TextAlignLast:    "auto",
		LineHeight:       1.2,
		UseRichText:      false,
		TextRendering:    "auto",
		DominantBaseline: "middle",
		TextAnchor:       "middle",
		PaintOrder:       "stroke",
		StrokeLinecap:    "square",
		StrokeLinejoin:   "bevel",
	}
}

// TextSpan represents a span of text.
type TextSpan struct {
	Text string
	X    float64
	Dy   *float64 // optional vertical offset
}

// LayerStyle represents a Krita layer style.
type LayerStyle struct {
	Enabled         bool
	Scale           float64
	LayerStyleUUID  string
	StrokeEnabled   bool
	StrokeStyle     string
	StrokeBlendMode string
	StrokeOpacity   float64
	StrokeSize      float64
	StrokeColor     [3]float64
}

// NewLayerStyle returns a new LayerStyle with default values.
func NewLayerStyle() *LayerStyle {
	return &LayerStyle{
		Enabled:         true,
		Scale:           100.0,
		LayerStyleUUID:  uuid.New().String(),
		StrokeEnabled:   false,
		StrokeStyle:     "OutF",
		StrokeBlendMode: "Nrml",
		StrokeOpacity:   100.0,
		StrokeSize:      3.0,
		StrokeColor:     [3]float64{255, 255, 255},
	}
}

// ShapeLayer represents a vector or text layer.
type ShapeLayer struct {
	// For text layers, Content holds []TextSpan.
	// For shape layers, Content holds []shapes.Shape.
	Content        interface{}
	ContentType    string // "text" or "shape"
	Name           string
	Visible        bool
	Opacity        int
	X, Y           float64
	// For text layers, Style is *TextStyle; for shape layers, it can be *shapes.ShapeStyle.
	Style          interface{}
	LayerStyle     *LayerStyle
	UUID           string
	LayerStyleUUID string
}

// FromText creates a ShapeLayer from plain text.
func FromText(text, name string, x, y float64, opacity int, style *TextStyle) *ShapeLayer {
	lines := splitLines(text)
	var spans []TextSpan
	for i, line := range lines {
		var dy *float64
		if i > 0 {
			val := float64(style.FontSize) * style.LineHeight
			dy = &val
		}
		spans = append(spans, TextSpan{Text: line, X: 0, Dy: dy})
	}
	return &ShapeLayer{
		Content:        spans,
		ContentType:    "text",
		Name:           name,
		Visible:        true,
		Opacity:        opacity,
		X:              x,
		Y:              y,
		Style:          style,
		LayerStyle:     nil,
		UUID:           "{" + uuid.New().String() + "}",
		LayerStyleUUID: uuid.New().String(),
	}
}

// FromShapes creates a ShapeLayer from a slice of shapes.
func FromShapes(shapesArr []shapes.Shape, name string, x, y float64, opacity int, style *shapes.ShapeStyle) *ShapeLayer {
	return &ShapeLayer{
		Content:        shapesArr,
		ContentType:    "shape",
		Name:           name,
		Visible:        true,
		Opacity:        opacity,
		X:              x,
		Y:              y,
		Style:          style,
		LayerStyle:     nil,
		UUID:           "{" + uuid.New().String() + "}",
		LayerStyleUUID: uuid.New().String(),
	}
}

// Helper to split a string into lines.
func splitLines(s string) []string {
	return []string{} // implement line splitting (e.g., using strings.Split)
}

// PaintLayer represents an image (pixel) layer.
type PaintLayer struct {
	// Either ImagePath (if loaded from disk) or an image.Image.
	Image     interface{} // use image.Image from the standard library
	ImagePath string
	Name      string
	Visible   bool
	Opacity   int
	X, Y      int
}
