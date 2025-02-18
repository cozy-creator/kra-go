package xmlhelper

import "fmt"

// XMLNode is a simple XML element tree.
type XMLNode struct {
	Tag      string
	Attrs    map[string]string
	Children []*XMLNode
	Text     string
}

// ToString returns an XML string representation.
func (n *XMLNode) ToString(indent string) string {
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