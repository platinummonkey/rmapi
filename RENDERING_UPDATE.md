# PDF Rendering Update - Image-Based Rendering with OCR

This update completely rewrites the PDF rendering pipeline inspired by [remarkable-searchable](https://github.com/platinummonkey/remarkable-searchable), providing high-quality image-based PDFs with optional OCR support.

## Summary of Changes

### 1. New Image-Based PDF Renderer (`rmconvert/image_pdf.go`)

**Key Features:**
- Renders reMarkable strokes to high-quality PNG images at configurable DPI
- Creates PDFs from PNG images using pdfcpu
- Much higher compatibility than vector-based rendering
- Configurable DPI for quality/size trade-offs

**Main Functions:**
- `ConvertRmdocToImagePDF(rmdocPath, pdfPath string, dpi int)` - Converts .rmdoc to image-based PDF
- `ConvertPageToPNG(writer io.Writer, dpi int)` - Renders a page to PNG
- `CreatePDFFromImagesExport(imagePaths []string, outputPath string)` - Creates PDF from PNGs

### 2. OCR Support for Searchable PDFs (`rmconvert/ocr_pdf.go`)

**Inspired by remarkable-searchable's approach:**
- Renders pages to PNG at high DPI
- Runs Tesseract OCR to extract text
- Adds invisible searchable text layer to PDF
- Text is positioned exactly where it appears in the image

**Main Functions:**
- `ConvertRmdocToSearchablePDF(rmdocPath, pdfPath string, dpi int, tessPath, lang string, psm int)` - Creates searchable PDF
- `ocrOnePage(...)` - Runs OCR on a single page
- `addOCRTextToPDF(...)` - Adds invisible text layer to PDF

**Requirements:**
- Tesseract OCR installed (optional)
- Language data files for desired languages

### 3. Updated mgeta Command (`shell/mgeta.go`)

**New Flags:**
```
-dpi <int>         Render DPI (default: 300)
-ocr               Enable OCR for searchable PDFs
-tess-path <path>  Path to tesseract binary (default: tesseract)
-tess-lang <lang>  Tesseract language (default: eng)
-tess-psm <mode>   Page segmentation mode (default: 6)
```

**Examples:**
```bash
# Basic usage - high-quality image-based PDFs
./rmapi shell -c "mgeta -o ~/Documents/ReMarkable -dpi 300 ."

# Higher quality (larger files)
./rmapi shell -c "mgeta -o ~/Documents/ReMarkable -dpi 600 ."

# With OCR for searchable PDFs
./rmapi shell -c "mgeta -o ~/Documents/ReMarkable -dpi 300 -ocr ."

# OCR with specific language
./rmapi shell -c "mgeta -o ~/Documents/ReMarkable -dpi 300 -ocr -tess-lang deu ."
```

## How It Works

### Without OCR (Default)
1. Extract .rmdoc file (ZIP archive)
2. Parse page order from .content file
3. For each page:
   - Parse .rm file (stroke data)
   - Render strokes to PNG at specified DPI
   - Store PNG temporarily
4. Use pdfcpu to create PDF from all PNGs
5. Clean up temporary files

### With OCR (Optional)
1-4. Same as above
5. For each PNG:
   - Run Tesseract OCR to generate hOCR output
   - Parse hOCR to extract word positions and text
6. Add invisible text layer to PDF:
   - Open PDF with pdfcpu
   - For each page with OCR results:
     - Create PDF content stream with invisible text (render mode 3)
     - Position text at exact coordinates from OCR
   - Write updated PDF
7. Clean up temporary files

## Benefits Over Previous Approach

### Image-Based Rendering

**Pros:**
- ✅ Higher compatibility with PDF viewers
- ✅ Predictable, consistent output
- ✅ Better handling of complex strokes
- ✅ Easy to verify visually (just check PNG)
- ✅ Configurable quality via DPI

**Cons:**
- ❌ Larger file sizes than vector PDFs
- ❌ Not true vector graphics (can't zoom infinitely)

### OCR Support

**Pros:**
- ✅ Searchable PDFs
- ✅ Text selection in PDF viewers
- ✅ Accessibility (screen readers)
- ✅ Copy/paste text from handwritten notes

**Cons:**
- ❌ Requires Tesseract installation
- ❌ Slower conversion
- ❌ OCR accuracy depends on handwriting quality

## Technical Details

### Coordinate Systems

- **reMarkable device**: 1404 x 1872 pixels (~226 DPI)
- **PNG rendering**: Scales to target DPI (e.g., 300 DPI = 1.327x scale)
- **PDF points**: 72 DPI (pdfcpu handles conversion)

### DPI Recommendations

- **150 DPI**: Small files, acceptable quality for simple notes
- **300 DPI** (default): Good balance of quality and file size
- **600 DPI**: High quality, larger files, good for archiving

### Fallback Strategy

The conversion has multiple fallback levels:
1. Try image-based rendering with OCR (if enabled)
2. Fall back to image-based rendering without OCR
3. Fall back to original vector-based PDF rendering
4. Fall back to SVG export (external tool required)

This ensures maximum compatibility and reliability.

## Testing

Tests confirmed that:
- ✅ Strokes are rendered correctly to PNG
- ✅ PNGs contain visible content
- ✅ PDFs created from PNGs have correct dimensions
- ✅ PDF embedded images have correct pixel data
- ✅ Build completes without errors

Visual inspection of test output:
- Test PNG: /tmp/test_visible.png (cross pattern, 73KB)
- Test PDF: /tmp/test_visible.pdf (cross pattern, 73KB)

## Installation

No additional dependencies are required for basic functionality:
```bash
go build
```

For OCR support, install Tesseract:
```bash
# macOS
brew install tesseract

# Ubuntu/Debian
sudo apt-get install tesseract-ocr

# Install additional languages (optional)
brew install tesseract-lang  # macOS
sudo apt-get install tesseract-ocr-deu  # German on Ubuntu
```

## Migration Notes

- Existing .rmdoc files will work without changes
- Old vector-based rendering is still available as fallback
- No breaking changes to API or command-line interface
- OCR is opt-in via `-ocr` flag

## Future Enhancements

Possible improvements:
- [ ] Parallel page processing for faster conversion
- [ ] Progress bar for large documents
- [ ] OCR confidence filtering
- [ ] Multi-language OCR support in single document
- [ ] PDF optimization/compression
- [ ] Incremental OCR (only re-OCR changed pages)

## Credits

Inspired by [remarkable-searchable](https://github.com/platinummonkey/remarkable-searchable) by the same author, which provides OCR for reMarkable-exported PDFs.
