package rmconvert

import (
	"fmt"
	"os"

	"github.com/juruen/rmapi/encoding/rm"
)

// ParseRMFile parses a reMarkable .rm file and returns a Page with strokes
// Supports v3, v5, and v6 formats
func ParseRMFile(filename string) (*Page, error) {
	// Read file data
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Use the rm package to parse (supports v3, v5, and v6)
	var rmData rm.Rm
	err = rmData.UnmarshalBinary(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rm file: %v", err)
	}

	// Convert to our Page format
	return convertRmToPage(&rmData), nil
}

// convertRmToPage converts rm.Rm to our Page format
func convertRmToPage(rmData *rm.Rm) *Page {
	page := &Page{
		Width:   1404,
		Height:  1872,
		Strokes: make([]Stroke, 0),
	}

	// Convert all layers and lines to strokes
	for _, layer := range rmData.Layers {
		for _, line := range layer.Lines {
			if len(line.Points) == 0 {
				continue
			}

			stroke := Stroke{
				Tool:   mapBrushTypeToTool(line.BrushType),
				Color:  mapBrushColorToColor(line.BrushColor),
				Width:  float32(line.BrushSize),
				Points: make([]Point, len(line.Points)),
			}

			for i, p := range line.Points {
				stroke.Points[i] = Point{
					X:         p.X,
					Y:         p.Y,
					Speed:     p.Speed,
					Direction: p.Direction,
					Width:     p.Width,
					Pressure:  p.Pressure,
				}
			}

			page.Strokes = append(page.Strokes, stroke)
		}
	}

	return page
}

// mapBrushTypeToTool maps rm.BrushType to our tool constants
func mapBrushTypeToTool(brushType rm.BrushType) int {
	switch brushType {
	case rm.Fineliner, rm.FinelinerV5:
		return ToolFineliner
	case rm.TiltPencil, rm.TiltPencilV5:
		return ToolPencil
	case rm.BallPoint, rm.BallPointV5:
		return ToolBallpoint
	case rm.Marker, rm.MarkerV5:
		return ToolMarker
	case rm.Highlighter, rm.HighlighterV5:
		return ToolHighlighter
	case rm.Eraser:
		return ToolEraser
	default:
		return ToolBallpoint
	}
}

// mapBrushColorToColor maps rm.BrushColor to our color constants
func mapBrushColorToColor(brushColor rm.BrushColor) int {
	switch brushColor {
	case rm.Black:
		return ColorBlack
	case rm.Grey:
		return ColorGray
	case rm.White:
		return ColorWhite
	default:
		return ColorBlack
	}
}

// CreateTestPage creates a simple test page with some basic strokes for testing
func CreateTestPage() *Page {
	page := &Page{
		Width:  1404,
		Height: 1872,
	}

	// Create a simple test stroke that should be clearly visible
	stroke1 := Stroke{
		Tool:  ToolFineliner,
		Color: ColorBlack,
		Width: 10.0,
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
		Width: 8.0,
		Points: []Point{
			{X: 600, Y: 800, Speed: 1.0, Direction: 0, Width: 8.0, Pressure: 0.7},
			{X: 1200, Y: 800, Speed: 1.0, Direction: 0, Width: 8.0, Pressure: 0.7},
		},
	}

	page.Strokes = []Stroke{stroke1, stroke2}
	return page
}
