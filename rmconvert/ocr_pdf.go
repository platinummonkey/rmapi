package rmconvert

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"golang.org/x/net/html"
)

// Word represents an OCR'd word with its bounding box
type Word struct {
	Text       string
	X1, Y1     int // top-left (pixels)
	X2, Y2     int // bottom-right (pixels)
	Confidence int
}

// PageOCR holds OCR results for one page
type PageOCR struct {
	PageNumber int
	ImgW, ImgH int // pixels
	Words      []Word
}

// ConvertRmdocToSearchablePDF creates a searchable PDF with OCR text layer
func ConvertRmdocToSearchablePDF(rmdocPath, pdfPath string, dpi int, tessPath, lang string, psm int) error {
	if dpi <= 0 {
		dpi = 300
	}
	if tessPath == "" {
		tessPath = "tesseract"
	}
	if lang == "" {
		lang = "eng"
	}
	if psm <= 0 {
		psm = 6
	}

	// Check if tesseract is available
	if _, err := exec.LookPath(tessPath); err != nil {
		fmt.Printf("Warning: tesseract not found, creating non-searchable PDF\n")
		return ConvertRmdocToImagePDF(rmdocPath, pdfPath, dpi)
	}

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "rmdoc_ocr_*")
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

	// Get page order
	pageOrder, docDir, err := getPageOrderAndDocDir(extractDir)
	if err != nil {
		return fmt.Errorf("failed to get page order: %v", err)
	}

	if len(pageOrder) == 0 {
		return fmt.Errorf("no pages found in document")
	}

	// Convert each page to PNG
	var pngFiles []string
	var ocrResults []PageOCR

	for i, pageID := range pageOrder {
		rmFile := filepath.Join(docDir, pageID+".rm")
		if _, err := os.Stat(rmFile); err != nil {
			fmt.Printf("Warning: page %s not found, skipping\n", pageID)
			continue
		}

		pngPath := filepath.Join(tempDir, fmt.Sprintf("page_%04d.png", i+1))
		err := convertRMToPNG(rmFile, pngPath, dpi)
		if err != nil {
			fmt.Printf("Warning: failed to convert page %s: %v\n", pageID, err)
			continue
		}

		pngFiles = append(pngFiles, pngPath)

		// Run OCR
		fmt.Printf("Running OCR on page %d...\n", i+1)
		ocr, err := ocrOnePage(tessPath, lang, psm, tempDir, pngPath, i+1)
		if err != nil {
			fmt.Printf("Warning: OCR failed for page %d: %v\n", i+1, err)
			// Continue without OCR for this page
		} else {
			ocrResults = append(ocrResults, ocr)
		}
	}

	if len(pngFiles) == 0 {
		return fmt.Errorf("no pages were successfully converted")
	}

	// Create PDF from images
	err = createPDFFromImages(pngFiles, pdfPath)
	if err != nil {
		return err
	}

	// Add OCR text layers if we have results
	if len(ocrResults) > 0 {
		fmt.Printf("Adding searchable text layer to %d pages...\n", len(ocrResults))
		err = addOCRTextToPDF(pdfPath, ocrResults, dpi)
		if err != nil {
			fmt.Printf("Warning: failed to add OCR text layer: %v\n", err)
			// PDF still exists, just without searchable text
		}
	}

	return nil
}

// ocrOnePage runs tesseract OCR on a PNG image
func ocrOnePage(tessPath, lang string, psm int, tmpDir, pngPath string, pageNum int) (PageOCR, error) {
	pageTag := fmt.Sprintf("ocr_p%04d", pageNum)
	hocrPath := filepath.Join(tmpDir, pageTag+".hocr")
	outBase := strings.TrimSuffix(hocrPath, ".hocr")

	// Run tesseract
	cmd := exec.Command(tessPath,
		pngPath,
		outBase,
		"-l", lang,
		"--psm", strconv.Itoa(psm),
		"hocr",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return PageOCR{}, fmt.Errorf("tesseract failed: %v: %s", err, string(output))
	}

	// Tesseract might output .html instead of .hocr
	if _, err := os.Stat(hocrPath); err != nil {
		alt := outBase + ".html"
		if _, err2 := os.Stat(alt); err2 == nil {
			hocrPath = alt
		} else {
			return PageOCR{}, fmt.Errorf("hocr output not found: %s", hocrPath)
		}
	}

	// Parse hOCR
	f, err := os.Open(hocrPath)
	if err != nil {
		return PageOCR{}, err
	}
	defer f.Close()

	words, imgW, imgH, err := parseHOCRWords(f)
	if err != nil {
		return PageOCR{}, err
	}

	return PageOCR{
		PageNumber: pageNum,
		ImgW:       imgW,
		ImgH:       imgH,
		Words:      words,
	}, nil
}

// parseHOCRWords extracts words from hOCR HTML
func parseHOCRWords(r *os.File) ([]Word, int, int, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, 0, 0, err
	}

	var words []Word
	var imgW, imgH int

	reBBox := regexp.MustCompile(`bbox\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)`)
	reConf := regexp.MustCompile(`x_wconf\s+(\d+)`)

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			cls := getAttr(n, "class")
			title := getAttr(n, "title")

			// Get page dimensions
			if strings.Contains(cls, "ocr_page") {
				if m := reBBox.FindStringSubmatch(title); m != nil {
					imgW, _ = strconv.Atoi(m[3])
					imgH, _ = strconv.Atoi(m[4])
				}
			}

			// Get words
			if strings.Contains(cls, "ocrx_word") {
				if m := reBBox.FindStringSubmatch(title); m != nil {
					x1, _ := strconv.Atoi(m[1])
					y1, _ := strconv.Atoi(m[2])
					x2, _ := strconv.Atoi(m[3])
					y2, _ := strconv.Atoi(m[4])

					conf := -1
					if cm := reConf.FindStringSubmatch(title); cm != nil {
						conf, _ = strconv.Atoi(cm[1])
					}

					txt := strings.TrimSpace(textContent(n))
					if txt != "" {
						words = append(words, Word{
							Text:       txt,
							X1:         x1,
							Y1:         y1,
							X2:         x2,
							Y2:         y2,
							Confidence: conf,
						})
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)
	return words, imgW, imgH, nil
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func textContent(n *html.Node) string {
	var buf bytes.Buffer
	var f func(*html.Node)
	f = func(x *html.Node) {
		if x.Type == html.TextNode {
			buf.WriteString(x.Data)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return buf.String()
}

// addOCRTextToPDF adds invisible searchable text layer to PDF
func addOCRTextToPDF(pdfPath string, ocrResults []PageOCR, dpi int) error {
	ctx, err := api.ReadContextFile(pdfPath)
	if err != nil {
		return fmt.Errorf("failed to read PDF: %v", err)
	}

	pageDims, err := ctx.XRefTable.PageDims()
	if err != nil {
		return fmt.Errorf("failed to get page dimensions: %v", err)
	}

	// NOTE: pdfcpu imports PNGs without DPI metadata as 72 DPI (1 pixel = 1 point)
	// So we use 1:1 pixel-to-point mapping regardless of render DPI
	pxToPt := 1.0

	for _, ocr := range ocrResults {
		if ocr.PageNumber > len(pageDims) {
			continue
		}

		dim := pageDims[ocr.PageNumber-1]
		pageHpt := dim.Height

		stream := buildInvisibleTextStream(ocr, pageHpt, pxToPt)
		if len(stream) == 0 {
			continue
		}

		err := appendTextStreamToPage(ctx, ocr.PageNumber, stream)
		if err != nil {
			return fmt.Errorf("failed to add text to page %d: %v", ocr.PageNumber, err)
		}
	}

	return api.WriteContextFile(ctx, pdfPath)
}

// buildInvisibleTextStream creates PDF content stream with invisible text
func buildInvisibleTextStream(ocr PageOCR, pageHpt float64, pxToPt float64) []byte {
	if len(ocr.Words) == 0 {
		return nil
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	fmt.Fprintln(w, "q")
	fmt.Fprintln(w, "BT")
	fmt.Fprintln(w, "3 Tr") // Invisible text mode
	fmt.Fprintln(w, "0 g")

	lastFontSize := -1.0
	for _, word := range ocr.Words {
		// Convert OCR bounding box from pixels to PDF points (pxToPt = 1.0)
		x1pt := float64(word.X1) * pxToPt
		y1pt := float64(word.Y1) * pxToPt
		y2pt := float64(word.Y2) * pxToPt

		// Calculate text height for font sizing
		hpt := y2pt - y1pt
		fontSize := clamp(hpt*0.85, 4, 72)

		// PDF coordinate system: (0,0) at bottom-left, Y increases upward
		// OCR coordinates: (0,0) at top-left, Y increases downward
		// pdfcpu embeds images with Y-flip, so we need to flip OCR coordinates
		// Position text at baseline (bottom of bbox): y2
		ypt := pageHpt - y2pt

		if abs(fontSize-lastFontSize) > 0.25 {
			fmt.Fprintf(w, "/F0 %.2f Tf\n", fontSize)
			lastFontSize = fontSize
		}

		fmt.Fprintf(w, "1 0 0 1 %.2f %.2f Tm\n", x1pt, ypt)
		fmt.Fprintf(w, "(%s) Tj\n", pdfEscapeString(word.Text))
	}

	fmt.Fprintln(w, "ET")
	fmt.Fprintln(w, "Q")
	w.Flush()

	return buf.Bytes()
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func pdfEscapeString(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '(':
			b.WriteString(`\(`)
		case ')':
			b.WriteString(`\)`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// appendTextStreamToPage adds text stream to PDF page
func appendTextStreamToPage(ctx *model.Context, pageNr int, content []byte) error {
	x := ctx.XRefTable

	pageDict, pageIndRef, inh, err := x.PageDict(pageNr, false)
	if err != nil {
		return err
	}

	// Ensure Helvetica font resource
	if err := ensureHelveticaFont(x, pageDict, inh); err != nil {
		return err
	}

	// Create new stream dict properly
	length := int64(len(content))
	sd := types.NewStreamDict(types.Dict{}, length, nil, nil, nil)
	sd.Content = content
	sd.Raw = content

	newIR, err := x.IndRefForNewObject(sd)
	if err != nil {
		return err
	}

	// Append to Contents
	co := pageDict["Contents"]
	switch c := co.(type) {
	case nil:
		pageDict["Contents"] = *newIR
	case types.IndirectRef:
		pageDict["Contents"] = types.Array{c, *newIR}
	case types.Array:
		pageDict["Contents"] = append(c, *newIR)
	default:
		return fmt.Errorf("unsupported Contents type: %T", co)
	}

	// Update the page dictionary in the xref table
	objNr := pageIndRef.ObjectNumber.Value()
	if entry, found := x.Table[objNr]; found {
		entry.Object = pageDict
	} else {
		return fmt.Errorf("page object %d not found in xref table", objNr)
	}

	return nil
}

// ensureHelveticaFont ensures Helvetica font is available in page resources
func ensureHelveticaFont(x *model.XRefTable, pageDict types.Dict, inh *model.InheritedPageAttrs) error {
	// Get or create Resources
	resObj := pageDict["Resources"]
	var resDict types.Dict

	switch r := resObj.(type) {
	case nil:
		resDict = types.Dict(map[string]types.Object{})
		pageDict["Resources"] = resDict
	case types.Dict:
		resDict = r
	case types.IndirectRef:
		o, err := x.Dereference(r)
		if err != nil {
			return err
		}
		d, ok := o.(types.Dict)
		if !ok {
			return fmt.Errorf("Resources not a dict: %T", o)
		}
		resDict = d
	default:
		return fmt.Errorf("unsupported Resources type: %T", resObj)
	}

	// Get or create Font dict
	fdObj := resDict["Font"]
	var fontDict types.Dict

	switch f := fdObj.(type) {
	case nil:
		fontDict = types.Dict(map[string]types.Object{})
		resDict["Font"] = fontDict
	case types.Dict:
		fontDict = f
	case types.IndirectRef:
		o, err := x.Dereference(f)
		if err != nil {
			return err
		}
		d, ok := o.(types.Dict)
		if !ok {
			return fmt.Errorf("Font not a dict: %T", o)
		}
		fontDict = d
	default:
		return fmt.Errorf("unsupported Font type: %T", fdObj)
	}

	// Add Helvetica if not present
	if _, ok := fontDict["F0"]; !ok {
		helv := types.Dict(map[string]types.Object{
			"Type":     types.Name("Font"),
			"Subtype":  types.Name("Type1"),
			"BaseFont": types.Name("Helvetica"),
			"Encoding": types.Name("WinAnsiEncoding"),
		})
		ir, err := x.IndRefForNewObject(helv)
		if err != nil {
			return err
		}
		fontDict["F0"] = *ir
	}

	return nil
}
