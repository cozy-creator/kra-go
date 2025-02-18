package asl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/cozy-creator/kritago/pkg/layers"
)

// CreateLayerStylesASL creates the ASL data for layer styles.
func CreateLayerStylesASL(layerInfos []struct {
	Layer     interface{}
	UUID      string
	LayerName string
}) ([]byte, error) {
	var asl bytes.Buffer
	// Write ASL header.
	if err := binary.Write(&asl, binary.BigEndian, uint16(2)); err != nil {
		return nil, err
	}
	asl.Write([]byte("8BSL"))
	if err := binary.Write(&asl, binary.BigEndian, uint16(3)); err != nil {
		return nil, err
	}
	if err := binary.Write(&asl, binary.BigEndian, uint32(0)); err != nil {
		return nil, err
	}
	numStyles := 0
	for _, li := range layerInfos {
		if sl, ok := li.Layer.(*layers.ShapeLayer); ok && sl.LayerStyle != nil {
			numStyles++
		}
	}
	if err := binary.Write(&asl, binary.BigEndian, uint32(numStyles)); err != nil {
		return nil, err
	}
	// Write each style.
	for _, li := range layerInfos {
		sl, ok := li.Layer.(*layers.ShapeLayer)
		if !ok || sl.LayerStyle == nil {
			continue
		}
		styleStartPos := asl.Len()
		if err := binary.Write(&asl, binary.BigEndian, uint32(0)); err != nil {
			return nil, err
		}
		asl.Write([]byte("null"))
		asl.Write([]byte("Nm  "))
		asl.Write([]byte("TEXT"))
		name := fmt.Sprintf("<%s> (embedded)", sl.Name)
		if err := writeASLString(&asl, name, "embedded"); err != nil {
			return nil, err
		}
		asl.Write([]byte("Idnt"))
		asl.Write([]byte("TEXT"))
		uuidStr := strings.ReplaceAll(sl.LayerStyleUUID, "-", "")
		uuidStr = "%" + uuidStr
		if err := writeASLString(&asl, uuidStr, "embedded"); err != nil {
			return nil, err
		}
		asl.Write([]byte("StyL"))
		asl.Write([]byte("documentMode"))
		asl.Write([]byte("Objc"))
		asl.Write([]byte("documentMode"))
		asl.Write([]byte("Lefx"))
		asl.Write([]byte("Objc"))
		asl.Write([]byte("Lefx"))
		asl.Write([]byte("Scl "))
		asl.Write([]byte("UntF#Prc"))
		if err := binary.Write(&asl, binary.BigEndian, sl.LayerStyle.Scale); err != nil {
			return nil, err
		}
		asl.Write([]byte("masterFXSwitch"))
		asl.Write([]byte("bool"))
		var masterByte byte = 0
		if sl.LayerStyle.Enabled {
			masterByte = 1
		}
		asl.Write([]byte{masterByte})
		if sl.LayerStyle.StrokeEnabled {
			asl.Write([]byte("FrFX"))
			asl.Write([]byte("Objc"))
			asl.Write([]byte("FrFX"))
			asl.Write([]byte("enab"))
			asl.Write([]byte("bool"))
			asl.Write([]byte{1})
			asl.Write([]byte("Style"))
			asl.Write([]byte("enum"))
			asl.Write([]byte("FStl"))
			asl.Write([]byte("OutF"))
			asl.Write([]byte("PntT"))
			asl.Write([]byte("enum"))
			asl.Write([]byte("FrFl"))
			asl.Write([]byte("SClr"))
			asl.Write([]byte("Md  "))
			asl.Write([]byte("enum"))
			asl.Write([]byte("BlnM"))
			asl.Write([]byte("Nrml"))
			asl.Write([]byte("Opct"))
			asl.Write([]byte("UntF#Prc"))
			if err := binary.Write(&asl, binary.BigEndian, sl.LayerStyle.StrokeOpacity); err != nil {
				return nil, err
			}
			asl.Write([]byte("Sz  "))
			asl.Write([]byte("UntF#Pxl"))
			if err := binary.Write(&asl, binary.BigEndian, sl.LayerStyle.StrokeSize); err != nil {
				return nil, err
			}
			asl.Write([]byte("Clr "))
			asl.Write([]byte("Objc"))
			asl.Write([]byte("RGBC"))
			channels := []string{"Rd  ", "Grn ", "Bl  "}
			for i, ch := range channels {
				asl.Write([]byte(ch))
				asl.Write([]byte("doub"))
				if err := binary.Write(&asl, binary.BigEndian, sl.LayerStyle.StrokeColor[i]); err != nil {
					return nil, err
				}
			}
		}
		styleSize := uint32(asl.Len() - styleStartPos - 4)
		binary.BigEndian.PutUint32(asl.Bytes()[styleStartPos:styleStartPos+4], styleSize)
	}
	return asl.Bytes(), nil
}

// writeASLString writes a string in ASL format.
func writeASLString(buf *bytes.Buffer, s string, stringType string) error {
	var encoded []byte
	if stringType == "embedded" {
		for i := 0; i < len(s); i++ {
			encoded = append(encoded, s[i], 0)
		}
		encoded = append(encoded, 0, 0)
	} else {
		encoded = []byte(s)
	}
	if stringType != "key" {
		if err := binary.Write(buf, binary.BigEndian, uint32(len(encoded))); err != nil {
			return err
		}
	}
	buf.Write(encoded)
	padding := (4 - (len(encoded) % 4)) % 4
	if padding > 0 {
		buf.Write(make([]byte, padding))
	}
	return nil
}
