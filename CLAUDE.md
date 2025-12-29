# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

rmapi is a Go CLI application for interacting with the reMarkable Cloud API. It provides both an interactive shell and non-interactive commands for managing reMarkable documents, downloading/uploading files, and converting reMarkable notebooks to PDF format with optional OCR support.

**Key Point**: This is a fork maintained by platinummonkey after the upstream project was archived.

## Building and Testing

### Build
```bash
go build
```

This produces the `rmapi` binary in the project root.

### Install from source
```bash
go install
```

### Run tests
```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./filetree
go test ./annotations
```

### Docker
```bash
# Build container
docker build -t rmapi .

# Run (mount config directory)
docker run -v $HOME/.config/rmapi/:/home/app/.config/rmapi/ -it rmapi
```

## Code Architecture

### High-Level Structure

The codebase is organized into distinct layers:

**1. Entry Point (`main.go`)**
- Handles authentication with retry logic (3 attempts)
- Parses JWT tokens to extract user info and sync version
- Creates API context and launches shell

**2. API Layer (`api/`)**
- `api.go`: Defines the `ApiCtx` interface that abstracts all cloud operations
- `sync15/`: Implementation of ReMarkable's sync protocol version 1.5
  - `apictx.go`: Core API context implementation with document tree management
  - `tree.go`: Hash tree for tracking document state and changes
  - `blobstorage.go`: Interface to cloud blob storage
  - Uses hash-based synchronization to detect changes

**3. File Tree (`filetree/`)**
- In-memory tree structure representing the document hierarchy
- Provides path-based navigation (like a filesystem)
- Handles parent-child relationships and node lookups
- Special handling for trash directory

**4. Shell (`shell/`)**
- `shell.go`: Shell initialization and command registration
- Each command in its own file (e.g., `ls.go`, `put.go`, `mget.go`)
- Uses `ishell` library for interactive shell with autocomplete
- Non-interactive mode: pass commands as arguments

**5. Document Encoding (`encoding/rm/`)**
- Parses reMarkable `.rm` files (binary stroke data)
- Supports versions 3, 5, and 6 of the format
- V6 uses a completely different tagged block structure (see V6_SUPPORT.md)

**6. Conversion (`rmconvert/`)**
- `image_pdf.go`: Renders reMarkable strokes to high-quality PNG images, then creates PDFs
- `ocr_pdf.go`: Adds searchable text layer to PDFs using Tesseract OCR
- `pdf.go`: Legacy vector-based PDF rendering (fallback)
- `svg.go`: SVG export support
- `parser.go`: Parses `.content` files to determine page ordering
- `convert.go`: Main conversion orchestration

**7. Archive (`archive/`)**
- Handles `.rmdoc` files (which are ZIP archives containing `.rm` files and metadata)
- Reads/writes metadata, content files
- Manages document structure

**8. Model (`model/`)**
- Data structures: `Document`, `Node`, `UserInfo`
- Document represents cloud metadata
- Node represents tree structure with parent/children

**9. Transport (`transport/`)**
- HTTP client with authentication
- Token management

### Key Architectural Patterns

**Hash-Based Sync**: The sync15 implementation uses SHA256 hashes to track document state. Documents are organized in a hash tree that allows efficient detection of changes.

**File Tree Navigation**: The filetree package provides a filesystem-like abstraction over the flat cloud storage, allowing path-based operations like `cd`, `ls`, etc.

**Command Pattern**: Each shell command is implemented as a separate function returning an ishell.Cmd structure, making it easy to add new commands.

**Fallback Strategy**: Conversion has multiple fallback levels:
1. Try image-based rendering with OCR (if enabled)
2. Fall back to image-based rendering without OCR
3. Fall back to vector-based PDF rendering
4. Fall back to external SVG tools

## reMarkable File Formats

### .rmdoc Files
ZIP archives containing:
- `.rm` files: Binary stroke data (one per page)
- `.content`: JSON with page ordering and metadata
- `.metadata`: Document metadata (name, parent, timestamps)

### .rm File Versions
- **V3/V5**: Flat binary format with stroke data - parsed natively in Go
- **V6**: Tagged block format (TLV-like structure) - requires `rmc` tool (Python)

Version detection happens by reading the header string (first 43 bytes).

## mgeta Command (Recursive Export with PDF Conversion)

The `mgeta` command is a major feature that combines recursive export with automatic PDF conversion.

**Implementation**: `shell/mgeta.go`

**Flow**:
1. Recursively download `.rmdoc` files (like `mget`)
2. For each `.rmdoc`:
   - Extract ZIP contents
   - Parse `.content` to get page order
   - Detect `.rm` file version (v3/v5/v6)
   - Convert each page to PDF (using appropriate method)
   - Merge pages into single PDF using `pdfunite`
3. Place `.rmdoc` and `.pdf` side-by-side in output directory

**Dependencies**:
- `rmc` (Python): Required for v6 file support
- `inkscape`: Required by rmc for PDF conversion
- `pdfunite` (from poppler): Required for merging multi-page PDFs
- `tesseract`: Optional, for OCR support

**Flags**:
- `-i`: Incremental mode (only process changed files)
- `-o <dir>`: Output directory
- `-d`: Remove local files deleted on device
- `-s`: Skip PDF conversion
- `-dpi <int>`: Render DPI (default: 300)
- `-ocr`: Enable OCR for searchable PDFs
- `-tess-path`, `-tess-lang`, `-tess-psm`: Tesseract configuration

## Image-Based PDF Rendering

As of the RENDERING_UPDATE, the default PDF rendering uses an image-based approach:

1. Parse `.rm` stroke data
2. Render strokes to PNG at specified DPI (default 300)
3. Create PDF from PNG images using pdfcpu
4. Optionally run Tesseract OCR and add invisible text layer

**Why image-based?**
- Higher compatibility with PDF viewers
- Predictable, consistent output
- Better handling of complex strokes

**Trade-offs**:
- Larger file sizes than vector PDFs
- Not true vector graphics (can't zoom infinitely)

**Coordinate systems**:
- reMarkable device: 1404 x 1872 pixels (~226 DPI)
- PNG rendering: Scales to target DPI
- PDF points: 72 DPI (pdfcpu handles conversion)

## Environment Variables

- `RMAPI_CONFIG`: Custom path for authentication tokens (default: `~/.rmapi`)
- `RMAPI_TRACE=1`: Enable trace logging
- `RMAPI_USE_HIDDEN_FILES=1`: Show/traverse hidden files
- `RMAPI_THUMBNAILS`: Generate PDF thumbnails
- `RMAPI_AUTH`: Override authorization URL
- `RMAPI_DOC`: Override document storage URL
- `RMAPI_HOST`: Override all URLs
- `RMAPI_CONCURRENT`: Max concurrent HTTP requests (default: 20)

## Common Development Workflows

### Adding a new shell command

1. Create new file in `shell/` (e.g., `mycommand.go`)
2. Implement a function returning `*ishell.Cmd`:
   ```go
   func myCommandCmd(ctx *ShellCtxt) *ishell.Cmd {
       return &ishell.Cmd{
           Name: "mycommand",
           Help: "description",
           Func: func(c *ishell.Context) {
               // implementation
           },
       }
   }
   ```
3. Register in `shell.go`'s `RunShell()` function: `shell.AddCmd(myCommandCmd(ctx))`

### Working with the file tree

Use `filetree.FileTreeCtx` for navigation:
- `NodeByPath(path, current)`: Get node by path string
- `NodesByPath(path, current, ignoreTrailingSlash)`: Get multiple nodes (glob support)
- `NodeToPath(node)`: Get path string from node

### Adding conversion format support

1. Create new file in `rmconvert/` (e.g., `newformat.go`)
2. Implement conversion function that:
   - Extracts `.rmdoc` archive
   - Parses `.rm` files using `encoding/rm`
   - Renders/exports to new format
3. Update mgeta or geta commands to support new format flag

## Important Notes

### Authentication
- Uses OAuth with device code flow
- Tokens stored in `~/.config/rmapi` or `RMAPI_CONFIG` path
- JWT tokens expire and need refresh
- Command `reset` removes stored credentials

### Sync Protocol
- Uses sync version 1.5 (only supported version)
- Hash-based change detection
- Concurrent operations limited by `RMAPI_CONCURRENT`

### V6 File Support
- V6 format introduced in reMarkable software version 3+
- Cannot be parsed natively - requires external `rmc` tool
- Detection is automatic based on file header
- Fallback to rmc adds significant overhead (~0.8s per page vs ~0.1s for native)

### File Operations
- All operations go through the API layer (`api.ApiCtx`)
- Changes are synced immediately to cloud
- Local file tree is rebuilt from cloud state on startup

## Debugging

Enable trace logging:
```bash
RMAPI_TRACE=1 ./rmapi
```

Non-interactive mode useful for debugging specific commands:
```bash
./rmapi ls /
./rmapi find . "pattern"
```

Check authentication status:
```bash
cat ~/.config/rmapi  # or $RMAPI_CONFIG
```
