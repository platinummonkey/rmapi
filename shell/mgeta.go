package shell

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/abiosoft/ishell"
	"github.com/juruen/rmapi/filetree"
	"github.com/juruen/rmapi/model"
	"github.com/juruen/rmapi/rmconvert"
	"github.com/juruen/rmapi/util"
)


// checkNativeConversionSupport verifies that native conversion is available
func checkNativeConversionSupport() error {
	// Native conversion doesn't require external tools
	// We could add additional checks here if needed
	return nil
}

// convertRmdocToPdf converts a .rmdoc file to PDF using image-based rendering with optional OCR
func convertRmdocToPdf(rmdocPath, pdfPath string, dpi int, enableOCR bool, tessPath, lang string, psm int, ctx *ShellCtxt) error {
	// Try OCR-enabled rendering if requested
	if enableOCR {
		err := rmconvert.ConvertRmdocToSearchablePDF(rmdocPath, pdfPath, dpi, tessPath, lang, psm)
		if err == nil {
			return nil
		}
		fmt.Printf("OCR rendering failed (%v), falling back to non-OCR rendering\n", err)
	}

	// Use image-based rendering (supports v3/v5/v6)
	return rmconvert.ConvertRmdocToImagePDF(rmdocPath, pdfPath, dpi)
}


func mgetACmd(ctx *ShellCtxt) *ishell.Cmd {
	return &ishell.Cmd{
		Name:      "mgeta",
		Help:      "recursively copy remote directory to local and convert to PDF (image-based rendering)\n\nUsage: mgeta [options] <source_dir>\n\nOptions:\n  -i           incremental mode (only download/convert if modified)\n  -o           output directory (default: current directory)\n  -d           remove deleted/moved files from local\n  -s           skip PDF conversion, only download .rmdoc files\n  -dpi         render DPI (default: 300, higher = better quality but larger files)\n  -ocr         enable OCR for searchable PDFs (requires tesseract)\n  -tess-path   path to tesseract binary (default: tesseract)\n  -tess-lang   tesseract language (default: eng)\n  -tess-psm    tesseract page segmentation mode (default: 6)\n\nFeatures:\n  - Image-based PDF rendering (high compatibility)\n  - Optional OCR support for searchable PDFs (like remarkable-searchable)\n  - Multi-page PDF support with proper page ordering\n  - Preserves stroke data and tool properties\n  - Configurable DPI for quality/size trade-off\n  - Fast parallel-safe conversion\n\nExamples:\n  mgeta -o ~/Documents/ReMarkable -dpi 300 .\n  mgeta -o ~/Documents/ReMarkable -dpi 300 -ocr -tess-lang eng .",
		Completer: createDirCompleter(ctx),
		Func: func(c *ishell.Context) {
			flagSet := flag.NewFlagSet("mgeta", flag.ContinueOnError)
			incremental := flagSet.Bool("i", false, "incremental")
			outputDir := flagSet.String("o", ".", "output folder")
			removeDeleted := flagSet.Bool("d", false, "remove deleted/moved")
			skipConversion := flagSet.Bool("s", false, "skip PDF conversion, only download .rmdoc files")
			dpi := flagSet.Int("dpi", 300, "render DPI (default: 300)")
			enableOCR := flagSet.Bool("ocr", false, "enable OCR for searchable PDFs")
			tessPath := flagSet.String("tess-path", "tesseract", "path to tesseract binary")
			tessLang := flagSet.String("tess-lang", "eng", "tesseract language")
			tessPSM := flagSet.Int("tess-psm", 6, "tesseract page segmentation mode")

			if err := flagSet.Parse(c.Args); err != nil {
				if err != flag.ErrHelp {
					c.Err(err)
				}
				return
			}

			// Check native conversion support unless skipping conversion
			if !*skipConversion {
				if err := checkNativeConversionSupport(); err != nil {
					c.Err(err)
					return
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
							if *enableOCR {
								c.Printf("converting [%s] to searchable PDF (DPI: %d, OCR: %s)...", rmdocPath, *dpi, *tessLang)
							} else {
								c.Printf("converting [%s] to PDF (DPI: %d)...", rmdocPath, *dpi)
							}
							err = convertRmdocToPdf(rmdocPath, pdfPath, *dpi, *enableOCR, *tessPath, *tessLang, *tessPSM, ctx)
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