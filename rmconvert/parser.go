package rmconvert

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	// reMarkable v6 format header
	rmHeaderSize = 43
	rmHeader     = "reMarkable .lines file, version=6"
)

// ParseRMFile parses a reMarkable .rm file and returns a Page with strokes
func ParseRMFile(filename string) (*Page, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	return ParseRM(file)
}

// ParseRM parses a reMarkable .rm file from an io.Reader
func ParseRM(reader io.Reader) (*Page, error) {
	buf := bufio.NewReader(reader)

	// Read and verify header
	header := make([]byte, rmHeaderSize)
	if _, err := io.ReadFull(buf, header); err != nil {
		return nil, fmt.Errorf("failed to read header: %v", err)
	}

	headerStr := string(header[:len(rmHeader)])
	if headerStr != rmHeader {
		return nil, fmt.Errorf("invalid header: expected '%s', got '%s'", rmHeader, headerStr)
	}

	// Initialize page with reMarkable dimensions
	page := &Page{
		Width:   1404,
		Height:  1872,
		Strokes: make([]Stroke, 0),
	}

	// Skip to layer data - this is a simplified parser
	// In the full format, there are nested blocks with metadata
	// For now, we'll try to find stroke data patterns

	err := parseStrokesSimplified(buf, page)
	if err != nil {
		// If simplified parsing fails, try to read any remaining stroke-like data
		fmt.Printf("Warning: simplified parsing failed: %v\n", err)
		// Continue with empty page rather than failing completely
	}

	return page, nil
}

// parseStrokesSimplified attempts to parse stroke data using a simplified approach
func parseStrokesSimplified(buf *bufio.Reader, page *Page) error {
	data, err := io.ReadAll(buf)
	if err != nil {
		return fmt.Errorf("failed to read data: %v", err)
	}

	return parseStrokeData(data, page)
}

// parseStrokeData tries to extract stroke information from binary data
func parseStrokeData(data []byte, page *Page) error {
	reader := bytes.NewReader(data)

	// This is a very simplified parser that looks for patterns that might be strokes
	// The actual format is much more complex with nested tagged blocks

	for reader.Len() > 0 {
		// Try to find stroke-like patterns
		stroke, err := tryParseStroke(reader)
		if err != nil {
			// Skip one byte and try again
			reader.Seek(1, io.SeekCurrent)
			continue
		}

		if stroke != nil && len(stroke.Points) > 0 {
			page.Strokes = append(page.Strokes, *stroke)
		}
	}

	return nil
}

// tryParseStroke attempts to parse a stroke from the current position
func tryParseStroke(reader *bytes.Reader) (*Stroke, error) {
	// This is a very basic attempt to parse stroke data
	// The actual format is much more complex

	// Skip if not enough data
	if reader.Len() < 20 {
		return nil, fmt.Errorf("not enough data")
	}

	// Try to read what might be stroke header info
	var tool, color uint32
	var width float32
	var pointCount uint32

	if err := binary.Read(reader, binary.LittleEndian, &tool); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &color); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &width); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &pointCount); err != nil {
		return nil, err
	}

	// Validate reasonable values
	if tool > 10 || color > 10 || width < 0 || width > 100 || pointCount == 0 || pointCount > 10000 {
		return nil, fmt.Errorf("invalid stroke parameters")
	}

	stroke := &Stroke{
		Tool:   int(tool),
		Color:  int(color),
		Width:  width,
		Points: make([]Point, 0, pointCount),
	}

	// Try to read points
	for i := uint32(0); i < pointCount && reader.Len() >= 24; i++ {
		var point Point
		if err := binary.Read(reader, binary.LittleEndian, &point.X); err != nil {
			break
		}
		if err := binary.Read(reader, binary.LittleEndian, &point.Y); err != nil {
			break
		}
		if err := binary.Read(reader, binary.LittleEndian, &point.Speed); err != nil {
			break
		}
		if err := binary.Read(reader, binary.LittleEndian, &point.Direction); err != nil {
			break
		}
		if err := binary.Read(reader, binary.LittleEndian, &point.Width); err != nil {
			break
		}
		if err := binary.Read(reader, binary.LittleEndian, &point.Pressure); err != nil {
			break
		}

		// Validate point coordinates are reasonable
		if point.X >= 0 && point.X <= 2000 && point.Y >= 0 && point.Y <= 3000 {
			stroke.Points = append(stroke.Points, point)
		}
	}

	return stroke, nil
}

// CreateTestPage creates a simple test page with some basic strokes for testing
func CreateTestPage() *Page {
	page := &Page{
		Width:  1404,
		Height: 1872,
	}

	// Create a simple test stroke that should be clearly visible
	// Use coordinates that won't be scaled to negative values
	stroke1 := Stroke{
		Tool:  ToolFineliner,
		Color: ColorBlack,
		Width: 10.0, // Much thicker stroke
		Points: []Point{
			{X: 700, Y: 200, Speed: 1.0, Direction: 0, Width: 10.0, Pressure: 0.8},
			{X: 900, Y: 400, Speed: 1.0, Direction: 0, Width: 10.0, Pressure: 0.8},
			{X: 1100, Y: 600, Speed: 1.0, Direction: 0, Width: 10.0, Pressure: 0.8},
		},
	}

	// Create another test stroke (horizontal line)
	stroke2 := Stroke{
		Tool:  ToolBallpoint,
		Color: ColorBlack,
		Width: 8.0, // Thick stroke
		Points: []Point{
			{X: 600, Y: 800, Speed: 1.0, Direction: 0, Width: 8.0, Pressure: 0.7},
			{X: 1200, Y: 800, Speed: 1.0, Direction: 0, Width: 8.0, Pressure: 0.7},
		},
	}

	page.Strokes = []Stroke{stroke1, stroke2}
	return page
}