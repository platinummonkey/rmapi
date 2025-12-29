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

// ConvertRmdocToPDF is deprecated - use ConvertRmdocToImagePDF or ConvertRmdocToSearchablePDF instead
func ConvertRmdocToPDF(rmdocPath, pdfPath string) error {
	return fmt.Errorf("ConvertRmdocToPDF is deprecated - use ConvertRmdocToImagePDF instead")
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

// TestConversion is deprecated - vector PDF rendering has been removed
func TestConversion(outputPath string) error {
	return fmt.Errorf("TestConversion is deprecated - use image-based rendering instead")
}

// ConvertRMFileToPDF is deprecated - vector PDF rendering has been removed
func ConvertRMFileToPDF(rmFilePath, pdfPath string) error {
	return fmt.Errorf("ConvertRMFileToPDF is deprecated - use image-based rendering instead")
}