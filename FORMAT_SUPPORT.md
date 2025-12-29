# reMarkable .rm File Format Support

This document describes the reMarkable `.rm` file format versions supported by rmapi and how version detection works.

## Supported Versions

rmapi supports **three versions** of the `.rm` format:

### Version 3 (V3)
- **Header**: `"reMarkable .lines file, version=3          "` (43 bytes)
- **Structure**: Binary format with layers, lines, and points
- **Features**: Basic stroke data with brush type, color, size, and point coordinates
- **Test File**: `encoding/rm/test_v3.rm` (89,855 bytes)
- **Status**: ✅ Fully supported and tested

### Version 5 (V5)
- **Header**: `"reMarkable .lines file, version=5          "` (43 bytes)
- **Structure**: Similar to V3 with additional fields
- **Features**: Adds an `Unknown` float32 field per line (new brush types: v5 brush type IDs 12-18)
- **Changes from V3**:
  - Additional 4-byte `Unknown` field after `BrushSize` in each line
  - New brush type constants (BallPointV5=15, MarkerV5=16, FinelinerV5=17, etc.)
- **Test File**: `encoding/rm/test_v5.rm` (23,723 bytes)
- **Status**: ✅ Fully supported and tested

### Version 6 (V6)
- **Header**: `"reMarkable .lines file, version=6          "` (43 bytes)
- **Structure**: Tagged block format (TLV-like structure)
- **Features**: Complete rewrite with CRDT support, scene trees, text items
- **Block Types**:
  - `0x00`: Migration Info
  - `0x02`: Page Info
  - `0x04`: Tree Node
  - `0x05`: Scene Item (Lines, Groups)
  - `0x06`: Text Item
  - `0x09`: Author IDs
- **Changes from V5**:
  - Completely different binary structure
  - Tagged blocks instead of flat layers
  - CRDT IDs for collaborative editing
  - Support for text items and groups
  - Different point format (compressed integers for speed/width/direction/pressure)
- **Test File**: None available (format introduced in reMarkable software v3+)
- **Status**: ✅ Supported via native Go parser in `encoding/rm/v6.go`

## Version Detection Logic

Version detection happens automatically in the `UnmarshalBinary` function:

```go
func (rm *Rm) UnmarshalBinary(data []byte) error {
    // Check header (first 43 bytes)
    if len(data) >= HeaderLen {
        header := string(data[:HeaderLen])

        // Detect version from header string
        switch header {
        case HeaderV6:
            return ParseV6(data)  // Use v6 block parser
        case HeaderV5:
            version = V5
        case HeaderV3:
            version = V3
        default:
            return fmt.Errorf("Unknown header: %s", header)
        }
    }

    // Use v3/v5 parser (similar structure)
    // ...
}
```

### Detection Flow

1. **Read header**: First 43 bytes of the `.rm` file
2. **Match header string**: Compare against known version strings
3. **Select parser**:
   - V6: Use `ParseV6()` with tagged block parser
   - V3/V5: Use shared binary parser with version-specific field handling
4. **Parse data**: Extract layers, lines, and points using appropriate format

## File Structure Comparison

### V3/V5 Format (Binary)
```
[Header: 43 bytes]
[Number of Layers: uint32]
For each layer:
    [Number of Lines: uint32]
    For each line:
        [BrushType: uint32]
        [BrushColor: uint32]
        [Padding: uint32]
        [BrushSize: float32]
        [Unknown: float32]  // Only in V5
        [Number of Points: uint32]
        For each point:
            [X: float32]
            [Y: float32]
            [Speed: float32]
            [Direction: float32]
            [Width: float32]
            [Pressure: float32]
```

### V6 Format (Tagged Blocks)
```
[Header: 43 bytes]
[Block 1]
    [Tag byte with type and size info]
    [Block data: variable length]
[Block 2]
    [Tag byte]
    [Block data]
...

Block structure:
- BlockType (1 byte)
- MinVersion (1 byte)
- CurrentVersion (1 byte)
- Size (4 bytes, uint32)
- Data (Size bytes)
```

## Coordinate System

All versions use the same coordinate system:
- **Width**: 1404 pixels
- **Height**: 1872 pixels
- **DPI**: ~226 (physical device)
- **Origin**: Top-left corner (0,0)

Points are stored as `float32` coordinates relative to this system.

## Conversion to PDF

The conversion pipeline (`rmconvert` package) handles all versions uniformly:

1. **Parse**: Detect version and unmarshal using `UnmarshalBinary()`
2. **Convert to internal format**: Both v3/v5/v6 are converted to the same `Page` structure
3. **Render**: Convert `Page.Strokes` to PNG images at target DPI
4. **Create PDF**: Use pdfcpu to create PDF from PNG images

See `rmconvert/parser.go` for the conversion logic.

## Testing

Tests are located in `encoding/rm/unmarshal_test.go`:

```bash
# Test v3 and v5 parsing
go test github.com/juruen/rmapi/encoding/rm -run TestUnmarshal

# Test specific version
go test github.com/juruen/rmapi/encoding/rm -run TestUnmarshalBinaryV3
go test github.com/juruen/rmapi/encoding/rm -run TestUnmarshalBinaryV5
```

**Test Results**: ✅ All tests passing

## Adding New Test Files

To add test files for new versions or edge cases:

1. Obtain a `.rm` file from a reMarkable device
2. Place it in `encoding/rm/` directory (e.g., `test_v6.rm`)
3. Add a test function in `unmarshal_test.go`:
   ```go
   func TestUnmarshalBinaryV6(t *testing.T) {
       testUnmarshalBinary(t, "test_v6.rm", V6)
   }
   ```
4. Run `go test github.com/juruen/rmapi/encoding/rm`

## References

- **Original Format Analysis**: [Axel Huebl's blog post](https://plasma.ninja/blog/devices/remarkable/binary/format/2017/12/26/reMarkable-lines-file-format.html)
- **Source Code**: `encoding/rm/` package
- **V6 Format Details**: `encoding/rm/v6.go`

## Related Files

- `encoding/rm/rm.go`: Format definitions and constants
- `encoding/rm/unmarshal.go`: V3/V5 binary parser
- `encoding/rm/v6.go`: V6 tagged block parser
- `encoding/rm/marshal.go`: Binary marshaling (encoding)
- `encoding/rm/unmarshal_test.go`: Test suite
- `rmconvert/parser.go`: Conversion to internal Page format
