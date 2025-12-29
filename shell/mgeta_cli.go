package shell

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/juruen/rmapi/filetree"
	"github.com/juruen/rmapi/model"
	"github.com/juruen/rmapi/rmconvert"
	"github.com/juruen/rmapi/util"
)

// convertRmdocToPdfCLI converts a .rmdoc file to PDF using image-based rendering with optional OCR
func convertRmdocToPdfCLI(rmdocPath, pdfPath string, dpi int, enableOCR bool, tessPath, lang string, psm int) error {
	// Try OCR-enabled rendering if requested
	if enableOCR {
		err := rmconvert.ConvertRmdocToSearchablePDF(rmdocPath, pdfPath, dpi, tessPath, lang, psm)
		if err == nil {
			return nil
		}
		fmt.Printf("OCR rendering failed (%v), falling back to non-OCR rendering\n", err)
	}

	// Try image-based rendering (now with native v3/v5/v6 support)
	err := rmconvert.ConvertRmdocToImagePDF(rmdocPath, pdfPath, dpi)
	if err == nil {
		return nil
	}

	// Fallback to direct PDF rendering
	return rmconvert.ConvertRmdocToPDFWithFallback(rmdocPath, pdfPath)
}

func mgetaCommand(ctx *Context) Command {
	return Command{
		Name: "mgeta",
		Help: "recursively copy remote directory to local and convert to PDF",
		Func: func(ctx *Context, args []string) error {
			flagSet := flag.NewFlagSet("mgeta", flag.ContinueOnError)
			incremental := flagSet.Bool("i", false, "incremental mode (only download/convert if modified)")
			outputDir := flagSet.String("o", ".", "output directory")
			removeDeleted := flagSet.Bool("d", false, "remove deleted/moved files from local")
			skipConversion := flagSet.Bool("s", false, "skip PDF conversion, only download .rmdoc files")
			dpi := flagSet.Int("dpi", 300, "render DPI (default: 300)")
			enableOCR := flagSet.Bool("ocr", false, "enable OCR for searchable PDFs (requires tesseract)")
			tessPath := flagSet.String("tess-path", "tesseract", "path to tesseract binary")
			tessLang := flagSet.String("tess-lang", "eng", "tesseract language")
			tessPSM := flagSet.Int("tess-psm", 6, "tesseract page segmentation mode")

			if err := flagSet.Parse(args); err != nil {
				return err
			}

			// Check native conversion support unless skipping conversion
			if !*skipConversion {
				if err := checkNativeConversionSupport(); err != nil {
					return err
				}
			}

			target := path.Clean(*outputDir)
			if *removeDeleted && target == "." {
				return fmt.Errorf("set a folder explicitly with the -o flag when removing deleted (and not .)")
			}

			argRest := flagSet.Args()
			if len(argRest) == 0 {
				return errors.New("missing source dir")
			}
			srcName := argRest[0]

			node, err := ctx.api.Filetree().NodeByPath(srcName, ctx.node)
			if err != nil || node.IsFile() {
				return errors.New("directory doesn't exist")
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
						fmt.Printf("downloading [%s]...", rmdocPath)

						err = ctx.api.FetchDocument(currentNode.Document.ID, rmdocPath)
						if err != nil {
							fmt.Printf(" FAILED: %v\n", err)
							return filetree.ContinueVisiting
						}

						fmt.Println(" OK")

						err = os.Chtimes(rmdocPath, lastModified, lastModified)
						if err != nil {
							fmt.Printf("warning: can't set lastModified for %s: %v\n", rmdocPath, err)
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
							if *enableOCR {
								fmt.Printf("converting [%s] to searchable PDF (DPI: %d, OCR: %s)...", rmdocPath, *dpi, *tessLang)
							} else {
								fmt.Printf("converting [%s] to PDF (DPI: %d)...", rmdocPath, *dpi)
							}
							err = convertRmdocToPdfCLI(rmdocPath, pdfPath, *dpi, *enableOCR, *tessPath, *tessLang, *tessPSM)
							if err != nil {
								fmt.Printf(" FAILED: %v\n", err)
							} else {
								fmt.Println(" OK")
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
						fmt.Printf("warning: can't read %s: %v\n", path, err)
						return nil
					}
					//just to be sure
					if path == target {
						return nil
					}
					if _, ok := fileMap[path]; !ok {
						var err error
						if info.IsDir() {
							fmt.Println("Removing folder ", path)
							err = os.RemoveAll(path)
							if err != nil {
								fmt.Printf("error removing folder: %v\n", err)
							}
							return filepath.SkipDir
						}

						fmt.Println("Removing ", path)
						err = os.Remove(path)
						if err != nil {
							fmt.Printf("error removing file: %v\n", err)
						}
					}
					return nil
				})
			}

			return nil
		},
	}
}
