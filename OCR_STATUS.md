# OCR Searchable PDF Status

## Current State

The OCR pipeline for creating searchable PDFs is **partially functional**:

### ✅ Working Components

1. **Tesseract Integration**: Successfully runs tesseract OCR on rendered PNG images
2. **hOCR Parsing**: Correctly parses hOCR HTML output to extract word boundaries and text
3. **Text Stream Generation**: Builds proper PDF content streams with invisible text (render mode 3 Tr)
4. **Fallback Mechanism**: Gracefully falls back to image-only PDFs when tesseract is not available
5. **Image-based PDF Creation**: Works perfectly for creating non-searchable PDFs from .rmdoc files

### ❌ Known Issue

**Text Layer Embedding**: There is a bug in the `appendTextStreamToPage` function (ocr_pdf.go:388-436) that causes a nil pointer dereference when trying to write the modified PDF using pdfcpu.

**Error**: `panic: runtime error: invalid memory address or nil pointer dereference` at `pdfcpu/pkg/pdfcpu/writeObjects.go:397`

**Root Cause**: The StreamDict structure created for adding content streams to PDF pages is missing required initialization that pdfcpu expects. Despite trying multiple approaches to initialize the StreamDict (using `types.NewStreamDict`, setting Raw/Content fields, etc.), the pdfcpu library encounters a nil pointer when attempting to write the stream.

## Test Coverage

Tests in `rmconvert/ocr_test.go`:

- `TestOCRFunctionality`: Validates the OCR pipeline works up to text stream generation (PASSING)
- `TestOCRFallback`: Validates fallback to non-searchable PDF (PASSING)

## Usage

### Image-only PDF (Recommended, Works Perfectly)
```bash
./rmapi mgeta -o output_dir /path/to/notebook
```

### With OCR Attempt (Experimental, Will Fall Back on Error)
```bash
./rmapi mgeta -o output_dir --ocr /path/to/notebook
```

Currently, when `--ocr` is specified:
1. The system attempts to run OCR and create searchable PDFs
2. If text layer embedding fails, it prints an error but continues
3. The result is an image-only PDF (same as non-OCR mode)

## Next Steps

To fix the text layer embedding issue:

1. **Investigate pdfcpu Internals**: Deep dive into pdfcpu's StreamDict requirements and content stream handling
2. **Alternative Libraries**: Consider using a different PDF library (e.g., unipdf/unipdf) that may have better API support for adding text layers
3. **Two-Pass Approach**: Instead of modifying an existing PDF, build the PDF with text layers from scratch
4. **External Tools**: Use external tools like `pdfunite` with searchable PDF layers, or `pdftk` to overlay text

## Related Files

- `rmconvert/ocr_pdf.go`: OCR pipeline and text layer embedding
- `rmconvert/image_pdf.go`: Image-based PDF creation (working)
- `rmconvert/ocr_test.go`: Test coverage
- `shell/mgeta_cli.go`: CLI command that calls OCR conversion
