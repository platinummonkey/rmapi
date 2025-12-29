package rmconvert

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

// ConvertPageToPNG renders a reMarkable page to a PNG image
func (page *Page) ConvertToPNG(writer io.Writer, dpi int) error {
	// reMarkable dimensions: 1404 x 1872 device pixels
	// Convert to desired DPI
	const rmWidth = 1404.0
	const rmHeight = 1872.0

	// Calculate dimensions at target DPI
	// reMarkable is approximately 226 DPI
	const rmDPI = 226.0
	scale := float64(dpi) / rmDPI

	width := rmWidth * scale
	height := rmHeight * scale

	// Create canvas with calculated dimensions
	c := canvas.New(width, height)
	ctx := canvas.NewContext(c)

	// Set white background
	ctx.SetFillColor(canvas.White)
	ctx.MoveTo(0, 0)
	ctx.LineTo(width, 0)
	ctx.LineTo(width, height)
	ctx.LineTo(0, height)
	ctx.Close()
	ctx.Fill()

	// Render each stroke
	for _, stroke := range page.Strokes {
		if len(stroke.Points) < 2 {
			continue
		}

		err := renderStrokeToPNG(ctx, &stroke, scale)
		if err != nil {
			fmt.Printf("Warning: failed to render stroke: %v\n", err)
			continue
		}
	}

	// Render to PNG
	pngWriter := renderers.PNG()
	return c.Write(writer, pngWriter)
}

// renderStrokeToPNG renders a single stroke to the PNG context
func renderStrokeToPNG(ctx *canvas.Context, stroke *Stroke, scale float64) error {
	if len(stroke.Points) < 2 {
		return fmt.Errorf("stroke must have at least 2 points")
	}

	props := GetToolProperties(stroke.Tool, stroke.Color, stroke.Width)

	// Set stroke properties
	color := parseColor(props.Color)
	ctx.SetStrokeColor(color)
	ctx.SetStrokeWidth(float64(props.StrokeWidth) * scale)
	ctx.SetStrokeCapper(canvas.RoundCap)
	ctx.SetStrokeJoiner(canvas.RoundJoin)

	// Start path by moving to first point
	firstPoint := stroke.Points[0]
	ctx.MoveTo(float64(firstPoint.X)*scale, float64(firstPoint.Y)*scale)

	// Add subsequent points
	for i := 1; i < len(stroke.Points); i++ {
		point := stroke.Points[i]
		ctx.LineTo(float64(point.X)*scale, float64(point.Y)*scale)
	}

	// Stroke the path
	ctx.Stroke()

	return nil
}

// ConvertRmdocToImagePDF converts a .rmdoc file to PDF using image-based rendering
// This approach renders each page to PNG and then creates a PDF from the images
func ConvertRmdocToImagePDF(rmdocPath, pdfPath string, dpi int) error {
	if dpi <= 0 {
		dpi = 300 // Default DPI
	}

	// Create temporary directory for PNGs
	tempDir, err := os.MkdirTemp("", "rmdoc_images_*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract .rmdoc file
	extractDir := filepath.Join(tempDir, "extracted")
	err = extractZip(rmdocPath, extractDir)
	if err != nil {
		return fmt.Errorf("failed to extract .rmdoc: %v", err)
	}

	// Find the document directory and get page order
	pageOrder, docDir, err := getPageOrderAndDocDir(extractDir)
	if err != nil {
		return fmt.Errorf("failed to get page order: %v", err)
	}

	if len(pageOrder) == 0 {
		return fmt.Errorf("no pages found in document")
	}

	// Create directory for PDF if it doesn't exist
	pdfDir := filepath.Dir(pdfPath)
	if err := os.MkdirAll(pdfDir, 0755); err != nil {
		return fmt.Errorf("failed to create PDF directory: %v", err)
	}

	// Convert each .rm file to PNG
	var pngFiles []string
	successCount := 0

	for i, pageID := range pageOrder {
		rmFile := filepath.Join(docDir, pageID+".rm")
		if _, err := os.Stat(rmFile); err != nil {
			// Page might not exist, skip it
			fmt.Printf("Warning: page %s not found, skipping\n", pageID)
			continue
		}

		pngPath := filepath.Join(tempDir, fmt.Sprintf("page_%04d.png", i+1))
		err := convertRMToPNG(rmFile, pngPath, dpi)
		if err != nil {
			// Print warning but continue with other pages
			fmt.Printf("Warning: failed to convert page %s to PNG: %v\n", pageID, err)
			continue
		}

		pngFiles = append(pngFiles, pngPath)
		successCount++
	}

	if successCount == 0 {
		return fmt.Errorf("no pages were successfully converted")
	}

	// Create PDF from PNGs using pdfcpu
	return createPDFFromImages(pngFiles, pdfPath)
}

// convertRMToPNG converts a single .rm file to PNG
func convertRMToPNG(rmFile, pngFile string, dpi int) error {
	// Parse .rm file
	page, err := ParseRMFile(rmFile)
	if err != nil {
		// If parsing fails, create empty page
		fmt.Printf("Warning: failed to parse %s, creating empty page: %v\n", rmFile, err)
		page = &Page{
			Width:   1404,
			Height:  1872,
			Strokes: []Stroke{},
		}
	}

	// Convert to PNG
	file, err := os.Create(pngFile)
	if err != nil {
		return fmt.Errorf("failed to create PNG file: %v", err)
	}
	defer file.Close()

	return page.ConvertToPNG(file, dpi)
}

// createPDFFromImages creates a PDF from a list of PNG images using pdfcpu
func createPDFFromImages(imagePaths []string, outputPath string) error {
	return CreatePDFFromImagesExport(imagePaths, outputPath)
}

// CreatePDFFromImagesExport creates a PDF from a list of PNG images using pdfcpu (exported for testing)
func CreatePDFFromImagesExport(imagePaths []string, outputPath string) error {
	if len(imagePaths) == 0 {
		return fmt.Errorf("no images to convert")
	}

	// Use pdfcpu's ImportImages API
	// Create a configuration with proper image handling
	conf := model.NewDefaultConfiguration()
	conf.CreateBookmarks = false

	// Import images to create PDF
	// The images will be embedded in the PDF
	err := api.ImportImagesFile(imagePaths, outputPath, nil, conf)
	if err != nil {
		return fmt.Errorf("failed to create PDF from images: %v", err)
	}

	return nil
}

// ConvertRMFileToImage converts a single .rm file to an image for testing
func ConvertRMFileToImage(rmFilePath, imagePath string, dpi int) error {
	return convertRMToPNG(rmFilePath, imagePath, dpi)
}

// RenderPageToImage renders a Page struct directly to an image.Image
func (page *Page) RenderToImage(dpi int) (image.Image, error) {
	const rmWidth = 1404.0
	const rmHeight = 1872.0
	const rmDPI = 226.0
	scale := float64(dpi) / rmDPI

	width := int(rmWidth * scale)
	height := int(rmHeight * scale)

	// Create canvas
	c := canvas.New(float64(width), float64(height))
	ctx := canvas.NewContext(c)

	// Set white background
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

		err := renderStrokeToPNG(ctx, &stroke, scale)
		if err != nil {
			fmt.Printf("Warning: failed to render stroke: %v\n", err)
			continue
		}
	}

	// Render via PNG encoding/decoding
	var buf []byte
	writer := &bufferWriter{buf: &buf}
	pngWriter := renderers.PNG()
	err := c.Write(writer, pngWriter)
	if err != nil {
		return nil, fmt.Errorf("failed to render to PNG: %v", err)
	}

	// Decode back to image.Image
	img, err := png.Decode(&bufferReader{buf: buf})
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %v", err)
	}

	return img, nil
}

// Helper types for in-memory buffer operations
type bufferWriter struct {
	buf *[]byte
}

func (w *bufferWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

type bufferReader struct {
	buf []byte
	pos int
}

func (r *bufferReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.buf) {
		return 0, io.EOF
	}
	n = copy(p, r.buf[r.pos:])
	r.pos += n
	return n, nil
}
