package rmconvert

import (
	"fmt"
	"image/color"
	"io"
	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

// ConvertPageToPDF converts a reMarkable page directly to PDF using canvas
func (page *Page) ConvertToPDF(writer io.Writer) error {
	// Calculate bounding box
	minX, minY, maxX, maxY := page.GetBoundingBox()
	width := maxX - minX
	height := maxY - minY

	// Create canvas with calculated dimensions
	c := canvas.New(float64(width), float64(height))
	ctx := canvas.NewContext(c)

	// Set white background using direct path operations
	ctx.SetFillColor(canvas.White)
	ctx.MoveTo(0, 0)
	ctx.LineTo(float64(width), 0)
	ctx.LineTo(float64(width), float64(height))
	ctx.LineTo(0, float64(height))
	ctx.Close()
	ctx.Fill()

	// Render each stroke
	for _, stroke := range page.Strokes {
		if len(stroke.Points) < 2 {
			continue
		}

		err := renderStrokeToCanvas(ctx, &stroke, float64(minX), float64(minY))
		if err != nil {
			fmt.Printf("Warning: failed to render stroke: %v\n", err)
			continue
		}
	}

	// Use PDF renderer to write to the io.Writer
	pdfWriter := renderers.PDF()
	return c.Write(writer, pdfWriter)
}

// renderStrokeToCanvas renders a single stroke to the canvas context
func renderStrokeToCanvas(ctx *canvas.Context, stroke *Stroke, offsetX, offsetY float64) error {
	if len(stroke.Points) < 2 {
		return fmt.Errorf("stroke must have at least 2 points")
	}

	props := GetToolProperties(stroke.Tool, stroke.Color, stroke.Width)

	// Set stroke properties
	color := parseColor(props.Color)
	ctx.SetStrokeColor(color)
	ctx.SetStrokeWidth(float64(props.StrokeWidth))
	// Note: canvas doesn't seem to have SetStrokeOpacity, so we'll handle opacity differently
	// ctx.SetStrokeOpacity(float64(props.Opacity))
	ctx.SetStrokeCapper(canvas.RoundCap)
	ctx.SetStrokeJoiner(canvas.RoundJoin)

	// Start path by moving to first point
	firstPoint := ScalePoint(stroke.Points[0])
	ctx.MoveTo(float64(firstPoint.X-float32(offsetX)), float64(firstPoint.Y-float32(offsetY)))

	// Add subsequent points
	for i := 1; i < len(stroke.Points); i++ {
		point := ScalePoint(stroke.Points[i])
		ctx.LineTo(float64(point.X-float32(offsetX)), float64(point.Y-float32(offsetY)))
	}

	// Stroke the path
	ctx.Stroke()

	return nil
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

// ConvertSVGToPDF converts an SVG string to PDF using canvas
func ConvertSVGToPDF(svgContent string, writer io.Writer) error {
	// Parse SVG and extract dimensions
	width, height := extractSVGDimensions(svgContent)
	if width == 0 || height == 0 {
		width = 595  // A4 width in points
		height = 842 // A4 height in points
	}

	// Create canvas
	c := canvas.New(width, height)
	ctx := canvas.NewContext(c)

	// Set white background
	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0, 0, canvas.Rectangle(width, height))

	// For now, we'll implement a basic SVG parser
	// In a production system, you'd want a full SVG parser
	err := renderBasicSVGToCanvas(ctx, svgContent)
	if err != nil {
		return fmt.Errorf("failed to render SVG: %v", err)
	}

	// Use PDF renderer to write to the io.Writer
	pdfWriter := renderers.PDF()
	return c.Write(writer, pdfWriter)
}

// extractSVGDimensions extracts width and height from SVG content
func extractSVGDimensions(svgContent string) (float64, float64) {
	// Simple regex-like parsing for width and height
	// This is very basic - a real implementation would use proper XML parsing

	var width, height float64 = 595, 842 // Default A4 size

	// Look for width="..." and height="..." patterns
	lines := strings.Split(svgContent, "\n")
	for _, line := range lines {
		if strings.Contains(line, "<svg") {
			// Try to extract width and height
			if strings.Contains(line, `width="`) {
				start := strings.Index(line, `width="`) + 7
				if start < len(line) {
					end := strings.Index(line[start:], `"`)
					if end > 0 {
						fmt.Sscanf(line[start:start+end], "%f", &width)
					}
				}
			}
			if strings.Contains(line, `height="`) {
				start := strings.Index(line, `height="`) + 8
				if start < len(line) {
					end := strings.Index(line[start:], `"`)
					if end > 0 {
						fmt.Sscanf(line[start:start+end], "%f", &height)
					}
				}
			}
			break
		}
	}

	return width, height
}

// renderBasicSVGToCanvas renders basic SVG elements to canvas
func renderBasicSVGToCanvas(ctx *canvas.Context, svgContent string) error {
	// This is a very basic SVG renderer that handles simple paths
	// A full implementation would use a proper SVG parser

	lines := strings.Split(svgContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "<path") {
			err := renderSVGPath(ctx, line)
			if err != nil {
				fmt.Printf("Warning: failed to render SVG path: %v\n", err)
			}
		}
	}

	return nil
}

// renderSVGPath renders a simple SVG path element
func renderSVGPath(ctx *canvas.Context, pathLine string) error {
	// Extract path data
	dStart := strings.Index(pathLine, `d="`)
	if dStart == -1 {
		return fmt.Errorf("no path data found")
	}
	dStart += 3

	dEnd := strings.Index(pathLine[dStart:], `"`)
	if dEnd == -1 {
		return fmt.Errorf("malformed path data")
	}

	pathData := pathLine[dStart : dStart+dEnd]

	// Extract stroke color
	strokeColor := canvas.Black
	if strings.Contains(pathLine, `stroke="`) {
		colorStart := strings.Index(pathLine, `stroke="`) + 8
		colorEnd := strings.Index(pathLine[colorStart:], `"`)
		if colorEnd > 0 {
			colorStr := pathLine[colorStart : colorStart+colorEnd]
			strokeColor = parseColor(colorStr)
		}
	}

	// Extract stroke width
	strokeWidth := 1.0
	if strings.Contains(pathLine, `stroke-width="`) {
		widthStart := strings.Index(pathLine, `stroke-width="`) + 14
		widthEnd := strings.Index(pathLine[widthStart:], `"`)
		if widthEnd > 0 {
			fmt.Sscanf(pathLine[widthStart:widthStart+widthEnd], "%f", &strokeWidth)
		}
	}

	// Set stroke properties
	ctx.SetStrokeColor(strokeColor)
	ctx.SetStrokeWidth(strokeWidth)
	ctx.SetStrokeCapper(canvas.RoundCap)
	ctx.SetStrokeJoiner(canvas.RoundJoin)

	// Parse and render path data
	path, err := parseBasicPathData(pathData)
	if err != nil {
		return err
	}

	ctx.DrawPath(0, 0, path)
	return nil
}

// parseBasicPathData parses basic SVG path data (M, L commands only)
func parseBasicPathData(data string) (*canvas.Path, error) {
	path := &canvas.Path{}

	// Split into commands and coordinates
	parts := strings.Fields(data)

	i := 0
	for i < len(parts) {
		if i >= len(parts) {
			break
		}

		command := parts[i]
		switch command {
		case "M":
			if i+2 >= len(parts) {
				return nil, fmt.Errorf("insufficient coordinates for M command")
			}
			var x, y float64
			if _, err := fmt.Sscanf(parts[i+1], "%f", &x); err != nil {
				return nil, fmt.Errorf("invalid x coordinate: %s", parts[i+1])
			}
			if _, err := fmt.Sscanf(parts[i+2], "%f", &y); err != nil {
				return nil, fmt.Errorf("invalid y coordinate: %s", parts[i+2])
			}
			path.MoveTo(x, y)
			i += 3
		case "L":
			if i+2 >= len(parts) {
				return nil, fmt.Errorf("insufficient coordinates for L command")
			}
			var x, y float64
			if _, err := fmt.Sscanf(parts[i+1], "%f", &x); err != nil {
				return nil, fmt.Errorf("invalid x coordinate: %s", parts[i+1])
			}
			if _, err := fmt.Sscanf(parts[i+2], "%f", &y); err != nil {
				return nil, fmt.Errorf("invalid y coordinate: %s", parts[i+2])
			}
			path.LineTo(x, y)
			i += 3
		default:
			// Try to parse as coordinates (assume L command)
			if len(command) > 0 && (command[0] >= '0' && command[0] <= '9' || command[0] == '-' || command[0] == '.') {
				if i+1 >= len(parts) {
					break
				}
				var x, y float64
				if _, err := fmt.Sscanf(parts[i], "%f", &x); err != nil {
					i++
					continue
				}
				if _, err := fmt.Sscanf(parts[i+1], "%f", &y); err != nil {
					i++
					continue
				}
				path.LineTo(x, y)
				i += 2
			} else {
				i++
			}
		}
	}

	return path, nil
}