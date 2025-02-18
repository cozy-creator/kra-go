package document

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/cozy-creator/kritago/pkg/asl"
	"github.com/cozy-creator/kritago/pkg/layers"
	"github.com/cozy-creator/kritago/pkg/shapes"
	"github.com/cozy-creator/kritago/pkg/xmlhelper"
	"github.com/zhuyie/golzf"
)

// KritaDocument represents a Krita document.
type KritaDocument struct {
	Width, Height int
	Layers        []interface{} // each is either *layers.ShapeLayer or *layers.PaintLayer
	TempDir       string
}

// NewKritaDocument creates a new KritaDocument.
func NewKritaDocument(width, height int) *KritaDocument {
	return &KritaDocument{
		Width:   width,
		Height:  height,
		Layers:  []interface{}{},
		TempDir: "krita_temp",
	}
}

// AddTextLayer adds a text layer.
func (doc *KritaDocument) AddTextLayer(text, name string, x, y float64, opacity int, style *layers.TextStyle) {
	layer := layers.FromText(text, name, x, y, opacity, style)
	doc.Layers = append(doc.Layers, layer)
}

// AddShapeLayer adds a shape layer.
func (doc *KritaDocument) AddShapeLayer(shapesArr []shapes.Shape, name string, x, y float64, opacity int, style *shapes.ShapeStyle) {
	layer := layers.FromShapes(shapesArr, name, x, y, opacity, style)
	doc.Layers = append(doc.Layers, layer)
}

// AddImageLayer adds an image layer.
func (doc *KritaDocument) AddImageLayer(img image.Image, imagePath, name string, x, y, opacity int) {
	layer := &layers.PaintLayer{
		Image:     img,
		ImagePath: imagePath,
		Name:      name,
		Visible:   true,
		Opacity:   opacity,
		X:         x,
		Y:         y,
	}
	doc.Layers = append(doc.Layers, layer)
}

// Save writes the document as a .kra file.
func (doc *KritaDocument) Save(outputPath string) error {
	// Create temporary directories.
	if err := os.MkdirAll(filepath.Join(doc.TempDir, "layers"), os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(doc.TempDir, "annotations"), os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(doc.TempDir, "animation"), os.ModePerm); err != nil {
		return err
	}

	// Prepare layer info.
	type LayerInfo struct {
		Layer     interface{}
		UUID      string
		LayerName string
	}
	var layerInfos []LayerInfo
	for i, layer := range doc.Layers {
		var uuidStr string
		var layerName string
		switch l := layer.(type) {
		case *layers.ShapeLayer:
			if l.UUID == "" {
				l.UUID = "{" + uuid.New().String() + "}"
			}
			uuidStr = l.UUID
			layerName = fmt.Sprintf("layer%d", i+2)
		case *layers.PaintLayer:
			uuidStr = "{" + uuid.New().String() + "}"
			layerName = fmt.Sprintf("layer%d", i+2)
		}
		layerInfos = append(layerInfos, LayerInfo{Layer: layer, UUID: uuidStr, LayerName: layerName})
	}

	// Create the output zip.
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	// 1. Write mimetype.
	if err := writeZipFile(zipWriter, "mimetype", []byte("application/x-krita")); err != nil {
		return err
	}

	// 2. Write documentinfo.xml.
	docInfo := doc.createDocumentInfo()
	if err := writeZipFile(zipWriter, "documentinfo.xml", []byte(docInfo)); err != nil {
		return err
	}

	// 3. Write maindoc.xml.
	mainDoc := doc.createMainDoc(layerInfos)
	if err := writeZipFile(zipWriter, "maindoc.xml", []byte(mainDoc)); err != nil {
		return err
	}

	// 4. Process each layer.
	if err := doc.processLayers(zipWriter, layerInfos); err != nil {
		return err
	}

	// 5. Write animation metadata.
	animMeta := doc.createAnimationMetadata()
	if err := writeZipFile(zipWriter, filepath.Join(doc.TempDir, "animation", "index.xml"), []byte(animMeta)); err != nil {
		return err
	}

	// 6. Add ICC profile.
	iccData, err := ioutil.ReadFile("layer3.icc")
	if err != nil {
		return err
	}
	if err := writeZipFile(zipWriter, filepath.Join(doc.TempDir, "annotations", "icc"), iccData); err != nil {
		return err
	}

	// 7. Create preview image.
	if err := doc.createPreview(zipWriter); err != nil {
		return err
	}

	// 8. Write layer styles if any.
	var hasStyle bool
	for _, li := range layerInfos {
		if sl, ok := li.Layer.(*layers.ShapeLayer); ok && sl.LayerStyle != nil {
			hasStyle = true
			break
		}
	}
	if hasStyle {
		aslBytes, err := asl.CreateLayerStylesASL(layerInfos)
		if err != nil {
			return err
		}
		if err := writeZipFile(zipWriter, filepath.Join(doc.TempDir, "annotations", "layerstyles.asl"), aslBytes); err != nil {
			return err
		}
	}

	fmt.Printf("Successfully created Krita file: %s\n", outputPath)
	os.RemoveAll(doc.TempDir)
	return nil
}

// Helper: writeZipFile writes data to the zip archive.
func writeZipFile(zf *zip.Writer, name string, data []byte) error {
	w, err := zf.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// createDocumentInfo creates documentinfo.xml.
func (doc *KritaDocument) createDocumentInfo() string {
	now := time.Now().Format("2006-01-02T15:04:05")
	root := &xmlhelper.XMLNode{
		Tag:   "document-info",
		Attrs: map[string]string{"xmlns": "http://www.calligra.org/DTD/document-info"},
	}
	about := &xmlhelper.XMLNode{Tag: "about"}
	about.Children = append(about.Children, &xmlhelper.XMLNode{Tag: "title"})
	about.Children = append(about.Children, &xmlhelper.XMLNode{Tag: "description"})
	about.Children = append(about.Children, &xmlhelper.XMLNode{Tag: "subject"})
	abstract := &xmlhelper.XMLNode{Tag: "abstract", Text: "\n"}
	about.Children = append(about.Children, abstract)
	about.Children = append(about.Children, &xmlhelper.XMLNode{Tag: "keyword"})
	creator := &xmlhelper.XMLNode{Tag: "initial-creator", Text: "Unknown"}
	about.Children = append(about.Children, creator)
	cycles := &xmlhelper.XMLNode{Tag: "editing-cycles", Text: "1"}
	about.Children = append(about.Children, cycles)
	about.Children = append(about.Children, &xmlhelper.XMLNode{Tag: "editing-time"})
	date := &xmlhelper.XMLNode{Tag: "date", Text: now}
	about.Children = append(about.Children, date)
	creation := &xmlhelper.XMLNode{Tag: "creation-date", Text: now}
	about.Children = append(about.Children, creation)
	about.Children = append(about.Children, &xmlhelper.XMLNode{Tag: "language"})
	about.Children = append(about.Children, &xmlhelper.XMLNode{Tag: "license"})
	root.Children = append(root.Children, about)

	author := &xmlhelper.XMLNode{Tag: "author"}
	author.Children = append(author.Children, &xmlhelper.XMLNode{Tag: "full-name"})
	author.Children = append(author.Children, &xmlhelper.XMLNode{Tag: "creator-first-name"})
	author.Children = append(author.Children, &xmlhelper.XMLNode{Tag: "creator-last-name"})
	author.Children = append(author.Children, &xmlhelper.XMLNode{Tag: "initial"})
	author.Children = append(author.Children, &xmlhelper.XMLNode{Tag: "author-title"})
	author.Children = append(author.Children, &xmlhelper.XMLNode{Tag: "position"})
	author.Children = append(author.Children, &xmlhelper.XMLNode{Tag: "company"})
	root.Children = append(root.Children, author)

	out := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + root.ToString("")
	return out
}

// LayerInfo represents layer metadata
type LayerInfo struct {
	Layer     interface{}
	UUID      string
	LayerName string
}

// createMainDoc creates maindoc.xml.
func (doc *KritaDocument) createMainDoc(layerInfos []LayerInfo) string {
	root := &xmlhelper.XMLNode{
		Tag: "DOC",
		Attrs: map[string]string{
			"xmlns":         "http://www.calligra.org/DTD/krita",
			"kritaVersion":  "5.2.9",
			"syntaxVersion": "2.0",
			"editor":        "Krita",
		},
	}
	imageAttrs := map[string]string{
		"width":          strconv.Itoa(doc.Width),
		"height":         strconv.Itoa(doc.Height),
		"mime":           "application/x-kra",
		"description":    "",
		"name":           "Unnamed",
		"y-res":          "300",
		"colorspacename": "RGBA",
		"x-res":          "300",
		"profile":        "sRGB-elle-V2-srgbtrc.icc",
	}
	imageNode := &xmlhelper.XMLNode{Tag: "IMAGE", Attrs: imageAttrs}

	layersNode := &xmlhelper.XMLNode{Tag: "layers"}
	// Build layer nodes (simplified).
	for _, li := range layerInfos {
		switch layer := li.Layer.(type) {
		case *layers.ShapeLayer:
			attrs := map[string]string{
				"collapsed":   "0",
				"visible":     "1",
				"locked":      "0",
				"y":           fmt.Sprintf("%v", layer.Y),
				"filename":    li.LayerName,
				"name":        layer.Name,
				"nodetype":    "shapelayer",
				"colorlabel":  "0",
				"compositeop": "normal",
				"x":           fmt.Sprintf("%v", layer.X),
				"uuid":        li.UUID,
				"intimeline":  "0",
				"opacity":     fmt.Sprintf("%v", layer.Opacity),
			}
			if layer.LayerStyle != nil {
				attrs["layerstyle"] = "{" + layer.LayerStyleUUID + "}"
			}
			layersNode.Children = append(layersNode.Children, &xmlhelper.XMLNode{Tag: "layer", Attrs: attrs})
		case *layers.PaintLayer:
			attrs := map[string]string{
				"intimeline":      "0",
				"visible":         "1",
				"locked":          "0",
				"y":               fmt.Sprintf("%v", layer.Y),
				"uuid":            li.UUID,
				"x":               fmt.Sprintf("%v", layer.X),
				"collapsed":       "0",
				"filename":        li.LayerName,
				"opacity":         fmt.Sprintf("%v", layer.Opacity),
				"name":            layer.Name,
				"nodetype":        "paintlayer",
				"colorspacename":  "RGBA",
				"compositeop":     "normal",
			}
			layersNode.Children = append(layersNode.Children, &xmlhelper.XMLNode{Tag: "layer", Attrs: attrs})
		}
	}
	imageNode.Children = append(imageNode.Children, layersNode)
	// Additional elements omitted for brevity.
	root.Children = append(root.Children, imageNode)
	header := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		"<!DOCTYPE DOC PUBLIC '-//KDE//DTD krita 2.0//EN' 'http://www.calligra.org/DTD/krita-2.0.dtd'>\n"
	return header + root.ToString("")
}

// createAnimationMetadata returns animation metadata XML.
func (doc *KritaDocument) createAnimationMetadata() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<animation-metadata xmlns="http://www.calligra.org/DTD/krita">
<framerate type="value" value="24"/>
<range from="0" type="timerange" to="100"/>
<currentTime type="value" value="0"/>
<export-settings>
<sequenceFilePath type="value" value=""/>
<sequenceBaseName type="value" value=""/>
<sequenceInitialFrameNumber type="value" value="-1"/>
</export-settings>
</animation-metadata>`
}

// createPreview creates a preview image and writes it into the zip.
func (doc *KritaDocument) createPreview(zf *zip.Writer) error {
	var preview image.Image
	// Use the last paint layer if available.
	if len(doc.Layers) > 0 {
		if pl, ok := doc.Layers[len(doc.Layers)-1].(*layers.PaintLayer); ok {
			if pl.Image != nil {
				preview = pl.Image.(image.Image)
			}
		}
	}
	if preview == nil {
		preview = image.NewRGBA(image.Rect(0, 0, doc.Width, doc.Height))
		draw.Draw(preview.(*image.RGBA), preview.Bounds(), &image.Uniform{C: image.Transparent}, image.Point{}, draw.Src)
	}
	thumb := image.NewRGBA(image.Rect(0, 0, 256, 256))
	draw.ApproxBiLinear.Scale(thumb, thumb.Bounds(), preview, preview.Bounds(), draw.Over, nil)
	var buf bytes.Buffer
	if err := png.Encode(&buf, thumb); err != nil {
		return err
	}
	return writeZipFile(zf, "preview.png", buf.Bytes())
}

// processLayers processes each layer and writes it to the zip.
func (doc *KritaDocument) processLayers(zf *zip.Writer, layerInfos []LayerInfo) error {
	for _, li := range layerInfos {
		switch layer := li.Layer.(type) {
		case *layers.ShapeLayer:
			if err := doc.addShapeLayerToZip(zf, layer, li.LayerName); err != nil {
				return err
			}
		case *layers.PaintLayer:
			if err := doc.addPaintLayerToZip(zf, layer, li.LayerName); err != nil {
				return err
			}
		}
	}
	return nil
}

// addShapeLayerToZip writes a shape layer's SVG content.
func (doc *KritaDocument) addShapeLayerToZip(zf *zip.Writer, layer *layers.ShapeLayer, layerName string) error {
	dirName := filepath.Join(doc.TempDir, "layers", layerName+".shapelayer")
	if err := os.MkdirAll(dirName, os.ModePerm); err != nil {
		return err
	}
	
	svgContent, err := GenerateSVGContent(layer, doc.Width, doc.Height)
	if err != nil {
		return err
	}
	svgPath := filepath.Join(dirName, "content.svg")
	if err := ioutil.WriteFile(svgPath, []byte(svgContent), 0644); err != nil {
		return err
	}
	data, err := ioutil.ReadFile(svgPath)
	if err != nil {
		return err
	}
	return writeZipFile(zf, filepath.Join("layers", layerName+".shapelayer", "content.svg"), data)
}

// addPaintLayerToZip writes a paint layer's data.
func (doc *KritaDocument) addPaintLayerToZip(zf *zip.Writer, layer *layers.PaintLayer, layerName string) error {
	// Load and process the image (tiled format).
	img, ok := layer.Image.(image.Image)
	if !ok {
		return errors.New("invalid image type")
	}
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, img.Bounds(), img, img.Bounds().Min, draw.Src)
	layerPath := filepath.Join(doc.TempDir, "layers", layerName)
	if err := SaveKritaLayer(rgba, layerPath); err != nil {
		return err
	}
	defaultPixelPath := layerPath + ".defaultpixel"
	if err := ioutil.WriteFile(defaultPixelPath, []byte{0, 0, 0, 0}, 0644); err != nil {
		return err
	}
	iccData, err := ioutil.ReadFile("layer3.icc")
	if err != nil {
		return err
	}
	iccPath := layerPath + ".icc"
	if err := ioutil.WriteFile(iccPath, iccData, 0644); err != nil {
		return err
	}
	filesToAdd := []string{layerPath, defaultPixelPath, iccPath}
	for _, filePath := range filesToAdd {
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return err
		}
		relPath := filepath.Join("layers", filepath.Base(filePath))
		if err := writeZipFile(zf, relPath, data); err != nil {
			return err
		}
	}
	return nil
}

// SaveKritaLayer saves an image as a Krita tiled layer.
func SaveKritaLayer(img image.Image, outputPath string) error {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	nx := int(math.Ceil(float64(w) / 64.0))
	ny := int(math.Ceil(float64(h) / 64.0))
	var tileEntries []struct {
		Header []byte
		Data   []byte
	}
	for ty := 0; ty < ny; ty++ {
		for tx := 0; tx < nx; tx++ {
			left := tx * 64
			top := ty * 64
			tileRect := image.Rect(0, 0, 64, 64)
			tileImg := image.NewRGBA(tileRect)
			srcRect := image.Rect(left, top, int(math.Min(float64(left+64), float64(w))), int(math.Min(float64(top+64), float64(h))))
			draw.Draw(tileImg, tileRect, img, srcRect.Min, draw.Src)
			var blue, green, red, alpha []byte
			for y := 0; y < 64; y++ {
				for x := 0; x < 64; x++ {
					c := tileImg.At(x, y)
					r, g, b, a := c.RGBA()
					red = append(red, uint8(r>>8))
					green = append(green, uint8(g>>8))
					blue = append(blue, uint8(b>>8))
					alpha = append(alpha, uint8(a>>8))
				}
			}
			planeData := append(append(blue, green...), append(red, alpha...)...)
			compressed, err := lzf.Compress(planeData)
			if err != nil {
				compressed = []byte{}
			}
			tileData := append([]byte{0x01}, compressed...)
			headerLine := fmt.Sprintf("%d,%d,LZF,%d\n", left, top, len(tileData))
			tileEntries = append(tileEntries, struct {
				Header []byte
				Data   []byte
			}{Header: []byte(headerLine), Data: tileData})
		}
	}
	var headerBuf bytes.Buffer
	headerBuf.WriteString("VERSION 2\n")
	headerBuf.WriteString("TILEWIDTH 64\n")
	headerBuf.WriteString("TILEHEIGHT 64\n")
	headerBuf.WriteString("PIXELSIZE 4\n")
	headerBuf.WriteString(fmt.Sprintf("DATA %d\n", len(tileEntries)))
	var outBuf bytes.Buffer
	outBuf.Write(headerBuf.Bytes())
	for _, entry := range tileEntries {
		outBuf.Write(entry.Header)
		outBuf.Write(entry.Data)
	}
	return ioutil.WriteFile(outputPath, outBuf.Bytes(), 0644)
}

// GenerateSVGContent generates SVG content for a shape layer.
// This is a stub; implement according to your SVG needs using pkg/xmlhelper.
func GenerateSVGContent(layer *layers.ShapeLayer, width, height int) (string, error) {
	// Build your SVG document with proper namespaces.
	// For example, create an XML tree using xmlhelper.XMLNode.
	header := `<?xml version="1.0" standalone="no"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 20010904//EN" "http://www.w3.org/TR/2001/REC-SVG-20010904/DTD/svg10.dtd">`
	// Return a dummy SVG.
	svg := fmt.Sprintf(`<svg width="%d" height="%d" viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">
  <!-- SVG content for layer %s -->
</svg>`, width, height, width, height, layer.Name)
	return header + "\n" + svg, nil
}
