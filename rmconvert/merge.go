package rmconvert

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// MergePDFs merges multiple PDF files into a single PDF file using pdfcpu
func MergePDFs(inputFiles []string, outputFile string) error {
	if len(inputFiles) == 0 {
		return fmt.Errorf("no input files provided")
	}

	if len(inputFiles) == 1 {
		// If only one file, just copy it
		return copyFile(inputFiles[0], outputFile)
	}

	// Use pdfcpu to merge PDFs
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	err := api.MergeCreateFile(inputFiles, outputFile, false, conf)
	if err != nil {
		return fmt.Errorf("failed to merge PDFs: %v", err)
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %v", err)
	}

	// Copy file content
	_, err = destFile.ReadFrom(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %v", err)
	}

	// Copy file permissions
	err = destFile.Chmod(sourceInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to set file permissions: %v", err)
	}

	return nil
}