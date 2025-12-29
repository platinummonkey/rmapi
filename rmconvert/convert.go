package rmconvert

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ConvertRmdocToPDF converts a .rmdoc file directly to PDF using native Go libraries
func ConvertRmdocToPDF(rmdocPath, pdfPath string) error {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "rmdoc_convert_*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract .rmdoc file (it's a zip archive)
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

	// Create directory for PDF if it doesn't exist
	pdfDir := filepath.Dir(pdfPath)
	if err := os.MkdirAll(pdfDir, 0755); err != nil {
		return fmt.Errorf("failed to create PDF directory: %v", err)
	}

	// Convert each .rm file to PDF
	var tempPdfs []string
	successCount := 0

	for i, pageID := range pageOrder {
		rmFile := filepath.Join(docDir, pageID+".rm")
		if _, err := os.Stat(rmFile); err != nil {
			// Page might not exist, skip it
			fmt.Printf("Warning: page %s not found, skipping\n", pageID)
			continue
		}

		tempPdf := filepath.Join(tempDir, fmt.Sprintf("page_%d.pdf", i+1))
		err := convertRMToPDF(rmFile, tempPdf)
		if err != nil {
			// Print warning but continue with other pages
			fmt.Printf("Warning: failed to convert page %s: %v\n", pageID, err)
			continue
		}

		tempPdfs = append(tempPdfs, tempPdf)
		successCount++
	}

	if successCount == 0 {
		return fmt.Errorf("no pages were successfully converted")
	}

	// Merge PDFs
	return MergePDFs(tempPdfs, pdfPath)
}

// convertRMToPDF converts a single .rm file to PDF
func convertRMToPDF(rmFile, pdfFile string) error {
	// Parse .rm file
	page, err := ParseRMFile(rmFile)
	if err != nil {
		// If parsing fails, try to create a test page to verify the pipeline works
		fmt.Printf("Warning: failed to parse %s, creating empty page: %v\n", rmFile, err)
		page = &Page{
			Width:   1404,
			Height:  1872,
			Strokes: []Stroke{},
		}
	}

	// Convert directly to PDF using canvas
	file, err := os.Create(pdfFile)
	if err != nil {
		return fmt.Errorf("failed to create PDF file: %v", err)
	}
	defer file.Close()

	return page.ConvertToPDF(file)
}

// extractZip extracts a zip file to the specified directory
func extractZip(src, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Create destination directory
	os.MkdirAll(dest, 0755)

	// Extract files
	for _, file := range reader.File {
		path := filepath.Join(dest, file.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.FileInfo().Mode())
			continue
		}

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		// Extract file
		fileReader, err := file.Open()
		if err != nil {
			return err
		}

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			fileReader.Close()
			return err
		}

		_, err = io.Copy(targetFile, fileReader)
		fileReader.Close()
		targetFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// ContentPage represents a page in the .content file
type ContentPage struct {
	ID       string `json:"id"`
	Modified string `json:"modifed"`
	Template struct {
		Value string `json:"value"`
	} `json:"template"`
	Idx struct {
		Value string `json:"value"`
	} `json:"idx"`
}

// ContentFile represents the structure of a .content file
type ContentFile struct {
	CPages struct {
		Pages []ContentPage `json:"pages"`
	} `json:"cPages"`
	PageCount int `json:"pageCount"`
}

// getPageOrderAndDocDir reads the .content file and returns the correct page order and document directory
func getPageOrderAndDocDir(extractDir string) ([]string, string, error) {
	var contentFile string
	var docDir string

	err := filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), ".content") {
			contentFile = path
		}
		if info.IsDir() && info.Name() != filepath.Base(extractDir) {
			// This should be the UUID directory containing .rm files
			if docDir == "" { // Take the first directory we find
				docDir = path
			}
		}
		return nil
	})

	if err != nil {
		return nil, "", err
	}

	if contentFile == "" {
		return nil, "", fmt.Errorf("no .content file found")
	}

	if docDir == "" {
		return nil, "", fmt.Errorf("no document directory found")
	}

	// Parse .content file
	data, err := os.ReadFile(contentFile)
	if err != nil {
		return nil, "", err
	}

	var content ContentFile
	err = json.Unmarshal(data, &content)
	if err != nil {
		return nil, "", err
	}

	// Extract page IDs in order
	var pageOrder []string
	for _, page := range content.CPages.Pages {
		pageOrder = append(pageOrder, page.ID)
	}

	// If no pages in content file, try to find .rm files directly
	if len(pageOrder) == 0 {
		files, err := os.ReadDir(docDir)
		if err != nil {
			return nil, "", err
		}
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".rm") {
				pageOrder = append(pageOrder, strings.TrimSuffix(file.Name(), ".rm"))
			}
		}
	}

	return pageOrder, docDir, nil
}

// TestConversion creates a test page and converts it to PDF for testing
func TestConversion(outputPath string) error {
	// Create test page
	page := CreateTestPage()

	// Convert to PDF
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create test PDF: %v", err)
	}
	defer file.Close()

	err = page.ConvertToPDF(file)
	if err != nil {
		return fmt.Errorf("failed to convert test page: %v", err)
	}

	fmt.Printf("Test PDF created: %s\n", outputPath)
	return nil
}

// ConvertRMFileToPDF converts a single .rm file to PDF for testing
func ConvertRMFileToPDF(rmFilePath, pdfPath string) error {
	return convertRMToPDF(rmFilePath, pdfPath)
}