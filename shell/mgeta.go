package shell

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/abiosoft/ishell"
	"github.com/juruen/rmapi/filetree"
	"github.com/juruen/rmapi/model"
	"github.com/juruen/rmapi/util"
)

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

// checkRmcTool verifies that rmc tool is available
func checkRmcTool() error {
	_, err := exec.LookPath("rmc")
	if err != nil {
		return fmt.Errorf("rmc tool not found. Please install with: pipx install rmc")
	}
	return nil
}

// checkInkscapeTool verifies that inkscape tool is available
func checkInkscapeTool() error {
	_, err := exec.LookPath("inkscape")
	if err != nil {
		return fmt.Errorf("inkscape not found. Install with: brew install inkscape (recommended for PDF conversion)")
	}
	return nil
}

// convertRmdocToPdf converts a .rmdoc file to PDF
func convertRmdocToPdf(rmdocPath, pdfPath string, ctx *ShellCtxt) error {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "rmdoc_convert_*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract .rmdoc file (it's a zip archive)
	err = extractRmdoc(rmdocPath, tempDir)
	if err != nil {
		return fmt.Errorf("failed to extract .rmdoc: %v", err)
	}

	// Find the document directory and get page order
	pageOrder, docDir, err := getPageOrder(tempDir)
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
	for i, pageID := range pageOrder {
		rmFile := filepath.Join(docDir, pageID+".rm")
		if _, err := os.Stat(rmFile); err != nil {
			// Page might not exist, skip it
			continue
		}

		tempPdf := filepath.Join(tempDir, fmt.Sprintf("page_%d.pdf", i+1))
		err := convertRmToPdf(rmFile, tempPdf)
		if err != nil {
			// Print warning but continue with other pages
			fmt.Printf("Warning: failed to convert page %s: %v\n", pageID, err)
			continue
		}
		tempPdfs = append(tempPdfs, tempPdf)
	}

	if len(tempPdfs) == 0 {
		return fmt.Errorf("no pages were successfully converted")
	}

	// Combine PDFs or copy single PDF
	if len(tempPdfs) == 1 {
		// Single page, just copy
		return copyFile(tempPdfs[0], pdfPath)
	}

	// Multiple pages, combine with pdfunite
	return combinePdfs(tempPdfs, pdfPath)
}

// extractRmdoc extracts a .rmdoc zip file to the specified directory
func extractRmdoc(rmdocPath, destDir string) error {
	reader, err := zip.OpenReader(rmdocPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		path := filepath.Join(destDir, file.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
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
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		_, err = io.Copy(targetFile, fileReader)
		if err != nil {
			return err
		}
	}

	return nil
}

// getPageOrder reads the .content file and returns the correct page order
func getPageOrder(extractDir string) ([]string, string, error) {
	// Find .content file
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

// convertRmToPdf converts a single .rm file to PDF using rmc
func convertRmToPdf(rmFile, pdfFile string) error {
	cmd := exec.Command("rmc", "--from", "rm", "--to", "pdf", "-o", pdfFile, rmFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rmc conversion failed: %v, output: %s", err, output)
	}
	return nil
}

// combinePdfs combines multiple PDF files into one using pdfunite
func combinePdfs(pdfFiles []string, outputFile string) error {
	args := []string{"pdfunite"}
	args = append(args, pdfFiles...)
	args = append(args, outputFile)

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try installing pdfunite if not found
		if strings.Contains(err.Error(), "executable file not found") {
			return fmt.Errorf("pdfunite not found. Install with: brew install poppler")
		}
		return fmt.Errorf("pdfunite failed: %v, output: %s", err, output)
	}
	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func mgetACmd(ctx *ShellCtxt) *ishell.Cmd {
	return &ishell.Cmd{
		Name:      "mgeta",
		Help:      "recursively copy remote directory to local and convert to PDF\n\nUsage: mgeta [options] <source_dir>\n\nOptions:\n  -i    incremental mode (only download/convert if modified)\n  -o    output directory (default: current directory)\n  -d    remove deleted/moved files from local\n  -s    skip PDF conversion, only download .rmdoc files\n\nRequirements:\n  - rmc tool (install with: pipx install rmc)\n  - inkscape (recommended, install with: brew install inkscape)\n  - poppler (for pdfunite, install with: brew install poppler)\n\nExample:\n  mgeta -o ~/Documents/ReMarkable .",
		Completer: createDirCompleter(ctx),
		Func: func(c *ishell.Context) {
			flagSet := flag.NewFlagSet("mgeta", flag.ContinueOnError)
			incremental := flagSet.Bool("i", false, "incremental")
			outputDir := flagSet.String("o", ".", "output folder")
			removeDeleted := flagSet.Bool("d", false, "remove deleted/moved")
			skipConversion := flagSet.Bool("s", false, "skip PDF conversion, only download .rmdoc files")

			if err := flagSet.Parse(c.Args); err != nil {
				if err != flag.ErrHelp {
					c.Err(err)
				}
				return
			}

			// Check dependencies unless skipping conversion
			if !*skipConversion {
				if err := checkRmcTool(); err != nil {
					c.Err(err)
					return
				}

				// Inkscape is recommended but not required
				if err := checkInkscapeTool(); err != nil {
					c.Printf("Warning: %v\n", err)
				}
			}

			target := path.Clean(*outputDir)
			if *removeDeleted && target == "." {
				c.Err(fmt.Errorf("set a folder explicitly with the -o flag when removing deleted (and not .)"))
				return
			}

			argRest := flagSet.Args()
			if len(argRest) == 0 {
				c.Err(errors.New("missing source dir"))
				return
			}
			srcName := argRest[0]

			node, err := ctx.api.Filetree().NodeByPath(srcName, ctx.node)

			if err != nil || node.IsFile() {
				c.Err(errors.New("directory doesn't exist"))
				return
			}

			fileMap := make(map[string]struct{})
			fileMap[target] = struct{}{}

			visitor := filetree.FileTreeVistor{
				func(currentNode *model.Node, currentPath []string) bool {
					idxDir := 0
					if srcName == "." && len(currentPath) > 0 {
						idxDir = 1
					}

					fileName := fmt.Sprintf("%s.%s", currentNode.Name(), util.RMDOC)
					pdfFileName := fmt.Sprintf("%s.pdf", currentNode.Name())

					rmdocPath := path.Join(target, filetree.BuildPath(currentPath[idxDir:], fileName))
					pdfPath := path.Join(target, filetree.BuildPath(currentPath[idxDir:], pdfFileName))

					fileMap[rmdocPath] = struct{}{}
					fileMap[pdfPath] = struct{}{}

					dir := path.Dir(rmdocPath)
					fileMap[dir] = struct{}{}

					os.MkdirAll(dir, 0766)

					if currentNode.IsDirectory() {
						return filetree.ContinueVisiting
					}

					lastModified, err := currentNode.LastModified()
					if err != nil {
						fmt.Printf("%v for %s\n", err, rmdocPath)
						lastModified = time.Now()
					}

					// Check if we need to download/convert based on timestamps
					needsUpdate := true
					if *incremental {
						stat, err := os.Stat(rmdocPath)
						if err == nil {
							localMod := stat.ModTime()
							if !lastModified.After(localMod) {
								needsUpdate = false
							}
						}
					}

					if needsUpdate {
						c.Printf("downloading [%s]...", rmdocPath)

						err = ctx.api.FetchDocument(currentNode.Document.ID, rmdocPath)

						if err != nil {
							c.Err(fmt.Errorf("Failed to download file %s", currentNode.Name()))
							return filetree.ContinueVisiting
						}

						c.Println(" OK")

						err = os.Chtimes(rmdocPath, lastModified, lastModified)
						if err != nil {
							c.Err(fmt.Errorf("cant set lastModified for %s", rmdocPath))
						}
					}

					// Convert to PDF if not skipping conversion
					if !*skipConversion {
						// Check if PDF needs update
						needsPdfUpdate := true
						if *incremental {
							stat, err := os.Stat(pdfPath)
							if err == nil {
								pdfMod := stat.ModTime()
								rmdocStat, rmdocErr := os.Stat(rmdocPath)
								if rmdocErr == nil && !rmdocStat.ModTime().After(pdfMod) {
									needsPdfUpdate = false
								}
							}
						}

						if needsPdfUpdate {
							c.Printf("converting [%s] to PDF...", rmdocPath)
							err = convertRmdocToPdf(rmdocPath, pdfPath, ctx)
							if err != nil {
								c.Printf(" FAILED: %v\n", err)
							} else {
								c.Println(" OK")
							}
						}
					}

					return filetree.ContinueVisiting
				},
			}

			filetree.WalkTree(node, visitor)

			if *removeDeleted {
				filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						c.Err(fmt.Errorf("can't read %s %v", path, err))
						return nil
					}
					//just to be sure
					if path == target {
						return nil
					}
					if _, ok := fileMap[path]; !ok {
						var err error
						if info.IsDir() {
							c.Println("Removing folder ", path)
							err = os.RemoveAll(path)
							if err != nil {
								c.Err(err)
							}
							return filepath.SkipDir
						}

						c.Println("Removing ", path)
						err = os.Remove(path)
						if err != nil {
							c.Err(err)
						}
					}
					return nil
				})
			}
		},
	}
}