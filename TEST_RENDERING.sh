#!/bin/bash

# Test script for image-based PDF rendering

echo "=== reMarkable PDF Rendering Tests ==="
echo ""

# Check if we have test data
if [ ! -f "test_data/sample.rmdoc" ]; then
    echo "No test data found. This script expects a sample .rmdoc file at:"
    echo "  test_data/sample.rmdoc"
    echo ""
    echo "To test with real data, download a document from your reMarkable and run:"
    echo "  ./rmapi get YourDocumentName"
    echo "  mkdir -p test_data"
    echo "  mv YourDocumentName.rmdoc test_data/sample.rmdoc"
    echo ""
    echo "Then re-run this test script."
    exit 1
fi

# Create output directory
mkdir -p test_output

echo "Test 1: Basic image-based PDF rendering (300 DPI)"
./rmapi shell -c "mgeta -o test_output -dpi 300 -s test_data/sample.rmdoc" || true

echo ""
echo "Test 2: High-quality rendering (600 DPI)"
./rmapi shell -c "mgeta -o test_output/hq -dpi 600 -s test_data/sample.rmdoc" || true

echo ""
echo "Test 3: With OCR for searchable PDF (requires tesseract)"
if command -v tesseract &> /dev/null; then
    echo "Tesseract found, running OCR test..."
    ./rmapi shell -c "mgeta -o test_output/ocr -dpi 300 -ocr test_data/sample.rmdoc" || true
else
    echo "Tesseract not found, skipping OCR test"
    echo "Install with: brew install tesseract (macOS) or apt-get install tesseract-ocr (Linux)"
fi

echo ""
echo "=== Test Results ==="
if [ -d "test_output" ]; then
    echo "Output files:"
    ls -lh test_output/*.pdf 2>/dev/null || echo "No PDFs generated"
    echo ""
    echo "To view results:"
    echo "  open test_output/*.pdf"
else
    echo "No output generated - check errors above"
fi

echo ""
echo "=== Usage Examples ==="
echo ""
echo "Download and convert all documents:"
echo "  ./rmapi shell -c \"mgeta -o ~/Documents/ReMarkable -dpi 300 .\""
echo ""
echo "With OCR for searchable PDFs:"
echo "  ./rmapi shell -c \"mgeta -o ~/Documents/ReMarkable -dpi 300 -ocr .\""
echo ""
echo "Incremental sync (only update changed files):"
echo "  ./rmapi shell -c \"mgeta -i -o ~/Documents/ReMarkable -dpi 300 .\""
echo ""
echo "High quality for archiving:"
echo "  ./rmapi shell -c \"mgeta -o ~/Documents/Archive -dpi 600 .\""
