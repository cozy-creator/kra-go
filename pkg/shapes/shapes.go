package shapes

import "fmt"

// ShapeStyle represents styling options for shapes.
type ShapeStyle struct {
	Fill            string
	Stroke          string
	StrokeWidth     float64
	StrokeOpacity   float64
	FillOpacity     float64
	StrokeLinecap   string
	StrokeLinejoin  string
	StrokeDasharray *string // optional
}

// NewShapeStyle returns a ShapeStyle with default values.
func NewShapeStyle() ShapeStyle {
	return ShapeStyle{
		Fill:           "none",
		Stroke:         "#000000",
		StrokeWidth:    1.0,
		StrokeOpacity:  1.0,
		FillOpacity:    1.0,
		StrokeLinecap:  "butt",
		StrokeLinejoin: "miter",
	}
}

// BaseShape is embedded in all shape types.
type BaseShape struct {
	Style     ShapeStyle
	Transform string
}

// GetSVGAttributes returns the common SVG attributes.
func (bs *BaseShape) GetSVGAttributes() map[string]string {
	attrs := map[string]string{
		"fill":            bs.Style.Fill,
		"stroke":          bs.Style.Stroke,
		"stroke-width":    fmt.Sprintf("%v", bs.Style.StrokeWidth),
		"stroke-opacity":  fmt.Sprintf("%v", bs.Style.StrokeOpacity),
		"fill-opacity":    fmt.Sprintf("%v", bs.Style.FillOpacity),
		"stroke-linecap":  bs.Style.StrokeLinecap,
		"stroke-linejoin": bs.Style.StrokeLinejoin,
	}
	if bs.Style.StrokeDasharray != nil {
		attrs["stroke-dasharray"] = *bs.Style.StrokeDasharray
	}
	if bs.Transform != "" {
		attrs["transform"] = bs.Transform
	}
	return attrs
}

// Shape defines the interface for vector shapes.
type Shape interface {
	GetSVGAttributes() map[string]string
	ToSVGElement() *SVGNode
}

// SVGNode is an XML node used in SVG output.
type SVGNode struct {
	Tag      string
	Attrs    map[string]string
	Children []*SVGNode
	Text     string
}

// ToString returns the XML string representation of an SVGNode.
func (n *SVGNode) ToString(indent string) string {
	attrs := ""
	for k, v := range n.Attrs {
		attrs += fmt.Sprintf(` %s="%s"`, k, v)
	}
	inner := n.Text
	for _, child := range n.Children {
		inner += "\n" + indent + "  " + child.ToString(indent+"  ")
	}
	if inner == "" {
		return fmt.Sprintf("<%s%s/>", n.Tag, attrs)
	}
	return fmt.Sprintf("<%s%s>%s</%s>", n.Tag, attrs, inner, n.Tag)
}

// Rectangle shape.
type Rectangle struct {
	BaseShape
	X, Y, Width, Height float64
	Rx, Ry              *float64 // optional
}

func (r *Rectangle) ToSVGElement() *SVGNode {
	attrs := r.GetSVGAttributes()
	attrs["x"] = fmt.Sprintf("%v", r.X)
	attrs["y"] = fmt.Sprintf("%v", r.Y)
	attrs["width"] = fmt.Sprintf("%v", r.Width)
	attrs["height"] = fmt.Sprintf("%v", r.Height)
	if r.Rx != nil {
		attrs["rx"] = fmt.Sprintf("%v", *r.Rx)
	}
	if r.Ry != nil {
		attrs["ry"] = fmt.Sprintf("%v", *r.Ry)
	}
	return &SVGNode{Tag: "rect", Attrs: attrs}
}

// Circle shape.
type Circle struct {
	BaseShape
	CX, CY, R float64
}

func (c *Circle) ToSVGElement() *SVGNode {
	attrs := c.GetSVGAttributes()
	attrs["cx"] = fmt.Sprintf("%v", c.CX)
	attrs["cy"] = fmt.Sprintf("%v", c.CY)
	attrs["r"] = fmt.Sprintf("%v", c.R)
	return &SVGNode{Tag: "circle", Attrs: attrs}
}

// Ellipse shape.
type Ellipse struct {
	BaseShape
	CX, CY, RX, RY float64
}

func (e *Ellipse) ToSVGElement() *SVGNode {
	attrs := e.GetSVGAttributes()
	attrs["cx"] = fmt.Sprintf("%v", e.CX)
	attrs["cy"] = fmt.Sprintf("%v", e.CY)
	attrs["rx"] = fmt.Sprintf("%v", e.RX)
	attrs["ry"] = fmt.Sprintf("%v", e.RY)
	return &SVGNode{Tag: "ellipse", Attrs: attrs}
}

// Line shape.
type Line struct {
	BaseShape
	X1, Y1, X2, Y2 float64
}

func (l *Line) ToSVGElement() *SVGNode {
	attrs := l.GetSVGAttributes()
	attrs["x1"] = fmt.Sprintf("%v", l.X1)
	attrs["y1"] = fmt.Sprintf("%v", l.Y1)
	attrs["x2"] = fmt.Sprintf("%v", l.X2)
	attrs["y2"] = fmt.Sprintf("%v", l.Y2)
	return &SVGNode{Tag: "line", Attrs: attrs}
}

// Path shape.
type Path struct {
	BaseShape
	D string
}

func (p *Path) ToSVGElement() *SVGNode {
	attrs := p.GetSVGAttributes()
	attrs["d"] = p.D
	return &SVGNode{Tag: "path", Attrs: attrs}
}

// ShapeGroup represents a group of shapes.
type ShapeGroup struct {
	Shapes    []Shape
	Transform string
}

func (sg *ShapeGroup) ToSVGElement() *SVGNode {
	attrs := map[string]string{}
	if sg.Transform != "" {
		attrs["transform"] = sg.Transform
	}
	group := &SVGNode{Tag: "g", Attrs: attrs}
	for _, shape := range sg.Shapes {
		group.Children = append(group.Children, shape.ToSVGElement())
	}
	return group
}
