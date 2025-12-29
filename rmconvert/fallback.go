package rmconvert

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConvertRmdocToPDFWithFallback converts a .rmdoc file to PDF with fallback strategies
func ConvertRmdocToPDFWithFallback(rmdocPath, pdfPath string) error {
	// First try native conversion
	err := ConvertRmdocToPDF(rmdocPath, pdfPath)
	if err == nil {
		return nil
	}

	fmt.Printf("Native conversion failed (%v), trying SVG fallback...\n", err)

	// Fallback: Convert to SVG first, then try external conversion
	svgPath := strings.TrimSuffix(pdfPath, ".pdf") + ".svg"
	err = ConvertRmdocToSVG(rmdocPath, svgPath)
	if err != nil {
		return fmt.Errorf("SVG fallback also failed: %v", err)
	}

	// Try to convert SVG to PDF using external tools
	err = convertSVGToPDFExternal(svgPath, pdfPath)
	if err != nil {
		fmt.Printf("Warning: Could not convert SVG to PDF (%v). SVG available at: %s\n", err, svgPath)
		return fmt.Errorf("conversion incomplete: SVG created but PDF conversion failed")
	}

	// Clean up SVG if PDF was successful
	os.Remove(svgPath)
	return nil
}

// ConvertRmdocToSVG converts a .rmdoc file to SVG
func ConvertRmdocToSVG(rmdocPath, svgPath string) error {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "rmdoc_svg_*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract .rmdoc file
	err = extractZip(rmdocPath, tempDir)
	if err != nil {
		return fmt.Errorf("failed to extract .rmdoc: %v", err)
	}

	// Find the document directory and get page order
	pageOrder, docDir, err := getPageOrderAndDocDir(tempDir)
	if err != nil {
		return fmt.Errorf("failed to get page order: %v", err)
	}

	if len(pageOrder) == 0 {
		return fmt.Errorf("no pages found in document")
	}

	// Convert to SVG - for multiple pages, we'll create individual SVGs
	var svgFiles []string
	successCount := 0

	for i, pageID := range pageOrder {
		rmFile := filepath.Join(docDir, pageID+".rm")
		if _, err := os.Stat(rmFile); err != nil {
			fmt.Printf("Warning: page %s not found, skipping\n", pageID)
			continue
		}

		var pageSVG string
		if len(pageOrder) == 1 {
			pageSVG = svgPath
		} else {
			// Multi-page: create individual SVGs
			baseName := strings.TrimSuffix(svgPath, ".svg")
			pageSVG = fmt.Sprintf("%s_page_%d.svg", baseName, i+1)
		}

		err := convertRMToSVG(rmFile, pageSVG)
		if err != nil {
			fmt.Printf("Warning: failed to convert page %s: %v\n", pageID, err)
			continue
		}

		svgFiles = append(svgFiles, pageSVG)
		successCount++
	}

	if successCount == 0 {
		return fmt.Errorf("no pages were successfully converted")
	}

	fmt.Printf("Created %d SVG file(s)\n", successCount)
	return nil
}

// convertRMToSVG converts a single .rm file to SVG
func convertRMToSVG(rmFile, svgFile string) error {
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

	// Generate SVG
	svgContent, err := page.GenerateSVG()
	if err != nil {
		return fmt.Errorf("failed to generate SVG: %v", err)
	}

	// Write SVG file
	return os.WriteFile(svgFile, []byte(svgContent), 0644)
}

// convertSVGToPDFExternal tries to convert SVG to PDF using external tools
func convertSVGToPDFExternal(svgPath, pdfPath string) error {
	// Try inkscape first
	if _, err := exec.LookPath("inkscape"); err == nil {
		cmd := exec.Command("inkscape", svgPath, "--export-filename", pdfPath)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	// Try cairosvg (Python tool)
	if _, err := exec.LookPath("cairosvg"); err == nil {
		cmd := exec.Command("cairosvg", svgPath, "-o", pdfPath)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	// Try rsvg-convert
	if _, err := exec.LookPath("rsvg-convert"); err == nil {
		cmd := exec.Command("rsvg-convert", "-f", "pdf", "-o", pdfPath, svgPath)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no suitable SVG to PDF converter found (tried: inkscape, cairosvg, rsvg-convert)")
}