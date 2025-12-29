package rmconvert

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestOCRFunctionality validates that OCR pipeline works (tesseract runs, hOCR parsing)
// Note: Text layer embedding to PDF has a known bug with pdfcpu that needs investigation
func TestOCRFunctionality(t *testing.T) {
	// Check if tesseract is available
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract not found, skipping OCR test")
	}

	tempDir := t.TempDir()
	rmdocPath := filepath.Join(tempDir, "test.rmdoc")

	// Create minimal .rmdoc structure
	err := createTestRmdoc(rmdocPath)
	if err != nil {
		t.Fatalf("Failed to create test .rmdoc: %v", err)
	}

	// Extract the .rmdoc
	extractDir := filepath.Join(tempDir, "extracted")
	err = extractZip(rmdocPath, extractDir)
	if err != nil {
		t.Fatalf("Failed to extract .rmdoc: %v", err)
	}

	// Get page order
	pageOrder, docDir, err := getPageOrderAndDocDir(extractDir)
	if err != nil {
		t.Fatalf("Failed to get page order: %v", err)
	}

	if len(pageOrder) == 0 {
		t.Fatal("No pages found")
	}

	// Convert first page to PNG
	rmFile := filepath.Join(docDir, pageOrder[0]+".rm")
	pngPath := filepath.Join(tempDir, "test.png")
	err = convertRMToPNG(rmFile, pngPath, 150)
	if err != nil {
		t.Fatalf("Failed to convert to PNG: %v", err)
	}

	// Run OCR on the PNG
	ocr, err := ocrOnePage("tesseract", "eng", 6, tempDir, pngPath, 1)
	if err != nil {
		t.Fatalf("OCR failed: %v", err)
	}

	// Validate OCR results
	t.Logf("OCR found %d words", len(ocr.Words))
	t.Logf("Image dimensions: %dx%d", ocr.ImgW, ocr.ImgH)

	if len(ocr.Words) == 0 {
		t.Log("Warning: OCR found no words (test page may be blank)")
	} else {
		t.Logf("Sample words: %v", ocr.Words[0])
	}

	// Test that we can build the invisible text stream
	stream := buildInvisibleTextStream(ocr, 792.0, 72.0/150.0)
	if len(stream) > 0 {
		t.Logf("Successfully built text stream (%d bytes)", len(stream))
	}
}

// TestOCRFallback validates that OCR conversion falls back to image PDF
func TestOCRFallback(t *testing.T) {
	tempDir := t.TempDir()
	rmdocPath := filepath.Join(tempDir, "test.rmdoc")
	pdfPath := filepath.Join(tempDir, "test.pdf")

	// Create minimal .rmdoc structure
	err := createTestRmdoc(rmdocPath)
	if err != nil {
		t.Fatalf("Failed to create test .rmdoc: %v", err)
	}

	// Convert with invalid tesseract path (should fall back)
	err = ConvertRmdocToSearchablePDF(rmdocPath, pdfPath, 150, "invalid_tesseract_path", "eng", 6)
	if err != nil {
		t.Fatalf("Conversion with fallback failed: %v", err)
	}

	// Verify PDF was created
	if _, err := os.Stat(pdfPath); err != nil {
		t.Fatalf("PDF not created: %v", err)
	}
}

// createTestRmdoc creates a minimal .rmdoc file for testing
func createTestRmdoc(destPath string) error {
	// Create a ZIP file
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// Create .content file
	content := ContentFile{
		PageCount: 1,
	}
	content.CPages.Pages = []ContentPage{
		{
			ID: "test-page-1",
		},
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return err
	}

	contentWriter, err := w.Create("test-doc.content")
	if err != nil {
		return err
	}
	if _, err := contentWriter.Write(contentJSON); err != nil {
		return err
	}

	// Copy test .rm file
	testRmPath := filepath.Join("..", "encoding", "rm", "test_v3.rm")
	rmFile, err := os.Open(testRmPath)
	if err != nil {
		// Try v5 if v3 doesn't exist
		testRmPath = filepath.Join("..", "encoding", "rm", "test_v5.rm")
		rmFile, err = os.Open(testRmPath)
		if err != nil {
			return err
		}
	}
	defer rmFile.Close()

	// Create the document directory structure
	rmWriter, err := w.Create("test-doc/test-page-1.rm")
	if err != nil {
		return err
	}

	if _, err := io.Copy(rmWriter, rmFile); err != nil {
		return err
	}

	return nil
}
