package rmconvert

import (
	"fmt"
	"image/color"
	"math"
	"strings"
)

// Point represents a point in a stroke with pressure, speed, direction, and width
type Point struct {
	X         float32
	Y         float32
	Speed     float32
	Direction float32
	Width     float32
	Pressure  float32
}

// Stroke represents a drawing stroke with tool information and points
type Stroke struct {
	Tool   int     // Tool type (0=fineliner, 1=pencil, 2=ballpoint, etc)
	Color  int     // Color index (0=black, 1=gray, 2=white)
	Width  float32 // Base stroke width
	Points []Point
}

// Page represents a reMarkable page with all its strokes
type Page struct {
	Width   float32
	Height  float32
	Strokes []Stroke
}

// Tool type constants based on reMarkable format
const (
	ToolFineliner   = 0
	ToolPencil      = 1
	ToolBallpoint   = 2
	ToolMarker      = 3
	ToolHighlighter = 4
	ToolEraser      = 5
)

// Color constants
const (
	ColorBlack = 0
	ColorGray  = 1
	ColorWhite = 2
)

// Tool properties for SVG generation
type ToolProperties struct {
	Name        string
	Color       string
	Opacity     float32
	StrokeWidth float32
}

// GetToolProperties returns SVG properties for a tool and color
func GetToolProperties(tool, color int, baseWidth float32) ToolProperties {
	props := ToolProperties{
		StrokeWidth: baseWidth,
		Opacity:     1.0,
	}

	// Set color
	switch color {
	case ColorBlack:
		props.Color = "black"
	case ColorGray:
		props.Color = "#777777"
	case ColorWhite:
		props.Color = "white"
	default:
		props.Color = "black"
	}

	// Adjust properties based on tool
	switch tool {
	case ToolFineliner:
		props.Name = "fineliner"
	case ToolPencil:
		props.Name = "pencil"
		props.Opacity = 0.8
	case ToolBallpoint:
		props.Name = "ballpoint"
	case ToolMarker:
		props.Name = "marker"
		props.StrokeWidth = baseWidth * 2
		props.Opacity = 0.7
	case ToolHighlighter:
		props.Name = "highlighter"
		props.StrokeWidth = baseWidth * 3
		props.Opacity = 0.4
	case ToolEraser:
		props.Name = "eraser"
		props.Color = "white"
		props.StrokeWidth = baseWidth * 2
	default:
		props.Name = "unknown"
	}

	return props
}

// ScalePoint applies reMarkable to PDF coordinate transformation
func ScalePoint(p Point) Point {
	// reMarkable coordinate system: 1404 x 1872 device pixels
	// Scale to standard page units (points: 72 DPI)
	// Based on rmc library scaling: simple scale without X centering
	const scale = 72.0 / 226.0

	return Point{
		X:         p.X * scale,
		Y:         p.Y * scale,
		Speed:     p.Speed,
		Direction: p.Direction,
		Width:     p.Width * scale,
		Pressure:  p.Pressure,
	}
}

// GetBoundingBox returns the bounding box of all strokes
func (page *Page) GetBoundingBox() (minX, minY, maxX, maxY float32) {
	if len(page.Strokes) == 0 {
		return 0, 0, page.Width, page.Height
	}

	minX = math.MaxFloat32
	minY = math.MaxFloat32
	maxX = -math.MaxFloat32
	maxY = -math.MaxFloat32

	for _, stroke := range page.Strokes {
		for _, point := range stroke.Points {
			scaled := ScalePoint(point)
			if scaled.X < minX {
				minX = scaled.X
			}
			if scaled.Y < minY {
				minY = scaled.Y
			}
			if scaled.X > maxX {
				maxX = scaled.X
			}
			if scaled.Y > maxY {
				maxY = scaled.Y
			}
		}
	}

	// Add padding
	padding := float32(10)
	minX -= padding
	minY -= padding
	maxX += padding
	maxY += padding

	return minX, minY, maxX, maxY
}

// String returns a string representation of the page
func (page *Page) String() string {
	return fmt.Sprintf("Page{Width: %.1f, Height: %.1f, Strokes: %d}",
		page.Width, page.Height, len(page.Strokes))
}

// parseColor converts a color string to color.RGBA
func parseColor(colorStr string) color.RGBA {
	switch strings.ToLower(colorStr) {
	case "black":
		return color.RGBA{0, 0, 0, 255}
	case "white":
		return color.RGBA{255, 255, 255, 255}
	case "#777777", "gray", "grey":
		return color.RGBA{119, 119, 119, 255}
	default:
		return color.RGBA{0, 0, 0, 255}
	}
}