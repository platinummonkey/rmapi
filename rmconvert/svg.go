package rmconvert

import (
	"bytes"
	"fmt"
	"strings"
)

// GenerateSVG creates an SVG representation of a reMarkable page
func (page *Page) GenerateSVG() (string, error) {
	var buf bytes.Buffer

	// Calculate bounding box
	minX, minY, maxX, maxY := page.GetBoundingBox()
	width := maxX - minX
	height := maxY - minY

	// Start SVG with header
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>`)
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" `+
		`width="%.2f" height="%.2f" `+
		`viewBox="%.2f %.2f %.2f %.2f">`,
		width, height, minX, minY, width, height))
	buf.WriteString("\n")

	// Add background if needed
	buf.WriteString(fmt.Sprintf(`  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" `+
		`fill="white" stroke="none"/>`,
		minX, minY, width, height))
	buf.WriteString("\n")

	// Generate strokes
	for i, stroke := range page.Strokes {
		if len(stroke.Points) < 2 {
			continue // Skip strokes with insufficient points
		}

		strokeSVG, err := generateStrokeSVG(&stroke, i)
		if err != nil {
			continue // Skip problematic strokes
		}

		buf.WriteString(strokeSVG)
		buf.WriteString("\n")
	}

	// Close SVG
	buf.WriteString("</svg>")

	return buf.String(), nil
}

// generateStrokeSVG creates SVG markup for a single stroke
func generateStrokeSVG(stroke *Stroke, strokeID int) (string, error) {
	if len(stroke.Points) < 2 {
		return "", fmt.Errorf("stroke must have at least 2 points")
	}

	props := GetToolProperties(stroke.Tool, stroke.Color, stroke.Width)

	var pathData strings.Builder

	// Start path with MoveTo command
	firstPoint := ScalePoint(stroke.Points[0])
	pathData.WriteString(fmt.Sprintf("M %.2f %.2f", firstPoint.X, firstPoint.Y))

	// Add subsequent points with LineTo commands
	for i := 1; i < len(stroke.Points); i++ {
		point := ScalePoint(stroke.Points[i])
		pathData.WriteString(fmt.Sprintf(" L %.2f %.2f", point.X, point.Y))
	}

	// Generate SVG path element
	svg := fmt.Sprintf(`  <path id="stroke-%d" `+
		`d="%s" `+
		`fill="none" `+
		`stroke="%s" `+
		`stroke-width="%.2f" `+
		`stroke-opacity="%.2f" `+
		`stroke-linecap="round" `+
		`stroke-linejoin="round"/>`,
		strokeID,
		pathData.String(),
		props.Color,
		props.StrokeWidth,
		props.Opacity)

	return svg, nil
}

// generateStrokeSVGWithVariableWidth creates SVG with variable width along the stroke
func generateStrokeSVGWithVariableWidth(stroke *Stroke, strokeID int) (string, error) {
	if len(stroke.Points) < 2 {
		return "", fmt.Errorf("stroke must have at least 2 points")
	}

	props := GetToolProperties(stroke.Tool, stroke.Color, stroke.Width)
	var buf strings.Builder

	// For variable width, we create multiple path segments or use polylines
	// This is a simplified implementation

	buf.WriteString(fmt.Sprintf(`  <g id="stroke-group-%d" stroke="%s" stroke-opacity="%.2f" fill="none">`,
		strokeID, props.Color, props.Opacity))
	buf.WriteString("\n")

	// Create segments with varying width
	for i := 0; i < len(stroke.Points)-1; i++ {
		p1 := ScalePoint(stroke.Points[i])
		p2 := ScalePoint(stroke.Points[i+1])

		// Use the average width of the two points
		avgWidth := (p1.Width + p2.Width) / 2
		if avgWidth <= 0 {
			avgWidth = props.StrokeWidth
		}

		buf.WriteString(fmt.Sprintf(`    <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" `+
			`stroke-width="%.2f" stroke-linecap="round"/>`,
			p1.X, p1.Y, p2.X, p2.Y, avgWidth))
		buf.WriteString("\n")
	}

	buf.WriteString("  </g>")
	return buf.String(), nil
}

// GenerateSVGWithVariableWidth creates an SVG with variable stroke widths
func (page *Page) GenerateSVGWithVariableWidth() (string, error) {
	var buf bytes.Buffer

	// Calculate bounding box
	minX, minY, maxX, maxY := page.GetBoundingBox()
	width := maxX - minX
	height := maxY - minY

	// Start SVG with header
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>`)
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" `+
		`width="%.2f" height="%.2f" `+
		`viewBox="%.2f %.2f %.2f %.2f">`,
		width, height, minX, minY, width, height))
	buf.WriteString("\n")

	// Add background
	buf.WriteString(fmt.Sprintf(`  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" `+
		`fill="white" stroke="none"/>`,
		minX, minY, width, height))
	buf.WriteString("\n")

	// Generate strokes with variable width
	for i, stroke := range page.Strokes {
		if len(stroke.Points) < 2 {
			continue
		}

		strokeSVG, err := generateStrokeSVGWithVariableWidth(&stroke, i)
		if err != nil {
			// Fallback to simple stroke
			strokeSVG, err = generateStrokeSVG(&stroke, i)
			if err != nil {
				continue
			}
		}

		buf.WriteString(strokeSVG)
		buf.WriteString("\n")
	}

	// Close SVG
	buf.WriteString("</svg>")

	return buf.String(), nil
}