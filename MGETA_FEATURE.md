# mgeta Command - reMarkable Export and PDF Conversion

The `mgeta` command is a new feature that combines the functionality of `mget` (recursive export) with automatic PDF conversion, providing a seamless workflow for exporting and converting all your reMarkable content.

## Overview

`mgeta` extends rmapi with the ability to:
1. Recursively export `.rmdoc` files from your reMarkable device (like `mget`)
2. Automatically convert each `.rmdoc` file to a multi-page PDF
3. Place both `.rmdoc` and `.pdf` files side-by-side in the same directory
4. Maintain the original folder structure from your reMarkable device

## Usage

```bash
mgeta [options] <source_directory>
```

### Options

- `-i` - **Incremental mode**: Only download and convert files that have been modified since the last run
- `-o <directory>` - **Output directory**: Specify where to save files (default: current directory)
- `-d` - **Remove deleted**: Remove local files that no longer exist on the device
- `-s` - **Skip conversion**: Only download `.rmdoc` files, skip PDF conversion

### Examples

```bash
# Export and convert everything from reMarkable
mgeta .

# Export specific folder with custom output directory
mgeta -o ~/Documents/ReMarkable "Direct notes"

# Incremental sync - only process changed files
mgeta -i -o ~/Documents/ReMarkable .

# Download only, skip PDF conversion
mgeta -s -o ./backup .
```

## Prerequisites

### Required Tools

1. **rmc** - For converting .rm files to PDF
   ```bash
   pipx install rmc
   ```

2. **poppler** - Provides `pdfunite` for combining multi-page PDFs
   ```bash
   brew install poppler
   ```

3. **inkscape** - For SVG to PDF conversion (recommended)
   ```bash
   brew install inkscape
   ```

### Dependency Checking

The command automatically checks for required dependencies and will:
- **Fail** if `rmc` is not found
- **Warn** if `inkscape` is not found (but continue)
- **Fail during conversion** if `pdfunite` is not found (only when needed)

## File Structure

The command maintains your reMarkable folder structure while creating both formats:

**Example Output:**
```
output_directory/
├── Direct notes/
│   ├── Daniel esponda/
│   │   ├── 2025-09-22.rmdoc
│   │   └── 2025-09-22.pdf       # ← Converted PDF
│   └── Daniel gozalo/
│       ├── 2025-09-22.rmdoc
│       └── 2025-09-22.pdf
├── Training/
│   ├── Eng manager training.rmdoc
│   └── Eng manager training.pdf  # ← Multi-page PDF
└── Quick sheets.rmdoc
└── Quick sheets.pdf
```

## Conversion Process

### How It Works

1. **Download**: Uses the same logic as `mget` to download `.rmdoc` files
2. **Extract**: Unzips the `.rmdoc` file (which is a ZIP archive)
3. **Parse**: Reads the `.content` file to determine correct page ordering
4. **Convert Pages**: Uses `rmc` to convert each `.rm` file to PDF
5. **Combine**: Uses `pdfunite` to merge individual pages into a single PDF
6. **Cleanup**: Removes temporary files

### Technical Details

- **Format Support**: Handles reMarkable v6 .rm files (software version 3+)
- **Page Ordering**: Respects the page order defined in the `.content` metadata
- **Multi-page**: Automatically combines multiple pages into single PDF documents
- **Error Handling**: Continues processing other files if individual conversions fail
- **Performance**: Processes files sequentially to avoid overwhelming external tools

## Features

### Incremental Mode (`-i`)

When using incremental mode:
- **Downloads** are skipped if the local `.rmdoc` file is newer than the remote version
- **PDF conversion** is skipped if the local PDF is newer than the source `.rmdoc`
- Greatly speeds up subsequent runs for large collections

### Error Handling

The command provides robust error handling:
- **Missing dependencies**: Clear error messages with installation instructions
- **Conversion failures**: Warns about individual file failures but continues processing
- **Network issues**: Inherits rmapi's connection error handling
- **File system errors**: Handles permission and disk space issues gracefully

### Compatibility

- **reMarkable Versions**: Works with reMarkable 2 and 3 software versions
- **File Formats**: Handles all notebook types (blank, lined, dotted, etc.)
- **Pen Types**: Preserves all pen types, colors, and stroke properties
- **Annotations**: Maintains drawing fidelity and layer information

## Troubleshooting

### Common Issues

1. **"rmc tool not found"**
   ```bash
   pipx install rmc
   ```

2. **"pdfunite not found"**
   ```bash
   brew install poppler
   ```

3. **Empty or corrupted PDFs**
   ```bash
   brew install inkscape
   ```

4. **Permission errors**
   - Ensure output directory is writable
   - Check file permissions for existing files

5. **Conversion warnings about newer format data**
   - This is normal and expected - conversion will still work
   - The rmscene library may lag behind reMarkable software updates

### Performance Tips

- Use incremental mode (`-i`) for large collections
- Consider using a local output directory first, then moving to cloud storage
- The conversion process is CPU-intensive; avoid running other heavy tasks simultaneously

## Comparison with Python Scripts

This native Go implementation offers several advantages over the separate Python scripts:

- **Integrated workflow**: Single command instead of multi-step process
- **Better error handling**: Integrated with rmapi's authentication and error handling
- **Dependency management**: Clear dependency checking with helpful error messages
- **Performance**: Native Go performance for file operations
- **Consistency**: Same command-line interface and patterns as other rmapi commands

## Contributing

The `mgeta` command is implemented in `/shell/mgeta.go`. Key areas for contribution:

- **Performance optimization**: Parallel processing of conversions
- **Format support**: Additional output formats beyond PDF
- **UI improvements**: Progress bars, better status reporting
- **Error recovery**: More robust handling of partial failures

## Future Enhancements

Potential future improvements:
- **Native conversion**: Implement rmscene parsing directly in Go
- **Format options**: Support for SVG, PNG, or other output formats
- **Parallel processing**: Convert multiple files simultaneously
- **Progress tracking**: Real-time progress reporting for large collections
- **Selective conversion**: Convert only specific file types or date ranges