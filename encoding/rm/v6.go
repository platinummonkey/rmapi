package rm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// V6 specific constants
const (
	HEADER_V6 = "reMarkable .lines file, version=6          "

	// Block types
	BLOCK_MIGRATION_INFO   = 0x00
	BLOCK_PAGE_INFO        = 0x02
	BLOCK_TREE_NODE        = 0x04
	BLOCK_SCENE_ITEM       = 0x05  // Lines, Groups, etc.
	BLOCK_TEXT_ITEM        = 0x06  // Text
	BLOCK_AUTHOR_IDS       = 0x09

	// Tag types (lower 4 bits of tag varint)
	TAG_BYTE1   = 0x01
	TAG_BYTE4   = 0x04
	TAG_BYTE8   = 0x08
	TAG_LENGTH4 = 0x0C
	TAG_ID      = 0x0F

	// Item types (within block data)
	ITEM_TYPE_GROUP = 0x00
	ITEM_TYPE_LINE  = 0x03
	ITEM_TYPE_TEXT  = 0x05
)

// V6Block represents a tagged block
type V6Block struct {
	BlockType      byte
	MinVersion     byte
	CurrentVersion byte
	Size           uint32
	Data           []byte
}

// V6Point represents a point in v6 format (version 2)
type V6Point struct {
	X         float32
	Y         float32
	Speed     uint16
	Width     uint16
	Direction uint8
	Pressure  uint8
}

// V6Line represents a line in v6 format
type V6Line struct {
	Color          int32
	Tool           int32
	Points         []V6Point
	ThicknessScale float64
	StartingLength float32
}

// V6CrdtId represents a CRDT ID
type V6CrdtId struct {
	Part1 uint8
	Part2 uint64
}

// ParseV6 parses a v6 format .rm file
func ParseV6(data []byte) (*Rm, error) {
	// Skip header (43 bytes)
	if len(data) < HeaderLen {
		return nil, fmt.Errorf("file too small")
	}

	header := string(data[:HeaderLen])
	if header != HeaderV6 {
		return nil, fmt.Errorf("not a v6 file")
	}

	r := bytes.NewReader(data[HeaderLen:])

	// Parse all blocks
	blocks, err := parseV6Blocks(r)
	if err != nil {
		return nil, err
	}

	// Extract lines from blocks
	lines := extractLinesFromV6Blocks(blocks)

	// Convert to Rm format
	rm := &Rm{
		Version: V6,
		Layers:  make([]Layer, 1),
	}

	if len(lines) > 0 {
		rm.Layers[0].Lines = make([]Line, len(lines))
		for i, v6line := range lines {
			rm.Layers[0].Lines[i] = convertV6Line(v6line)
		}
	}

	return rm, nil
}

// parseV6Blocks parses all blocks from v6 file
func parseV6Blocks(r *bytes.Reader) ([]V6Block, error) {
	var blocks []V6Block

	for r.Len() > 0 {
		block, err := parseV6Block(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return blocks, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// parseV6Block parses a single block
// Block header format (8 bytes):
//   - block_length (uint32, 4 bytes) - little endian
//   - unknown (uint8, 1 byte) - always 0
//   - min_version (uint8, 1 byte)
//   - current_version (uint8, 1 byte)
//   - block_type (uint8, 1 byte)
func parseV6Block(r *bytes.Reader) (V6Block, error) {
	var block V6Block

	// Read block length (uint32, 4 bytes)
	var blockLength uint32
	if err := binary.Read(r, binary.LittleEndian, &blockLength); err != nil {
		return block, err
	}
	block.Size = blockLength

	// Read unknown byte (should be 0)
	var unknown byte
	if err := binary.Read(r, binary.LittleEndian, &unknown); err != nil {
		return block, err
	}

	// Read minimum version (1 byte)
	if err := binary.Read(r, binary.LittleEndian, &block.MinVersion); err != nil {
		return block, err
	}

	// Read current version (1 byte)
	if err := binary.Read(r, binary.LittleEndian, &block.CurrentVersion); err != nil {
		return block, err
	}

	// Read block type (1 byte)
	if err := binary.Read(r, binary.LittleEndian, &block.BlockType); err != nil {
		return block, err
	}

	// Read block data
	if block.Size > 0 {
		block.Data = make([]byte, block.Size)
		if _, err := io.ReadFull(r, block.Data); err != nil {
			return block, err
		}
	}

	return block, nil
}

// extractLinesFromV6Blocks extracts line data from blocks
func extractLinesFromV6Blocks(blocks []V6Block) []V6Line {
	var lines []V6Line

	for _, block := range blocks {
		if block.BlockType == BLOCK_SCENE_ITEM {
			line, err := parseSceneItemBlock(block.Data, block.CurrentVersion)
			if err == nil && line != nil {
				lines = append(lines, *line)
			}
		}
	}

	return lines
}

// parseSceneItemBlock parses a scene item block
// Structure:
//   - tagged ID at index 1: parent_id
//   - tagged ID at index 2: item_id
//   - tagged ID at index 3: left_id
//   - tagged ID at index 4: right_id
//   - tagged int at index 5: deleted_length
//   - tagged subblock at index 6: item data (if not deleted)
func parseSceneItemBlock(data []byte, blockVersion byte) (*V6Line, error) {
	r := bytes.NewReader(data)

	// Read parent_id (index 1)
	if _, err := expectTag(r, 1, TAG_ID); err != nil {
		return nil, err
	}
	if _, err := readCrdtId(r); err != nil {
		return nil, err
	}

	// Read item_id (index 2)
	if _, err := expectTag(r, 2, TAG_ID); err != nil {
		return nil, err
	}
	if _, err := readCrdtId(r); err != nil {
		return nil, err
	}

	// Read left_id (index 3)
	if _, err := expectTag(r, 3, TAG_ID); err != nil {
		return nil, err
	}
	if _, err := readCrdtId(r); err != nil {
		return nil, err
	}

	// Read right_id (index 4)
	if _, err := expectTag(r, 4, TAG_ID); err != nil {
		return nil, err
	}
	if _, err := readCrdtId(r); err != nil {
		return nil, err
	}

	// Read deleted_length (index 5)
	if _, err := expectTag(r, 5, TAG_BYTE4); err != nil {
		return nil, err
	}
	var deletedLength uint32
	if err := binary.Read(r, binary.LittleEndian, &deletedLength); err != nil {
		return nil, err
	}

	// If deleted, skip
	if deletedLength > 0 {
		return nil, nil
	}

	// Check for subblock at index 6
	if r.Len() == 0 {
		return nil, nil
	}

	// Read subblock tag and length
	if _, err := expectTag(r, 6, TAG_LENGTH4); err != nil {
		return nil, nil  // No value subblock, skip
	}

	var subblockLen uint32
	if err := binary.Read(r, binary.LittleEndian, &subblockLen); err != nil {
		return nil, err
	}

	// Read item type (first byte of subblock)
	var itemType byte
	if err := binary.Read(r, binary.LittleEndian, &itemType); err != nil {
		return nil, err
	}

	// Only parse LINE items (type 0x03)
	if itemType != ITEM_TYPE_LINE {
		return nil, nil
	}

	// Parse line data
	line, err := parseLineData(r, blockVersion)
	if err != nil {
		return nil, err
	}

	return line, nil
}

// parseLineData parses line data from stream
// Structure:
//   - tagged int at index 1: tool_id
//   - tagged int at index 2: color_id
//   - tagged double at index 3: thickness_scale
//   - tagged float at index 4: starting_length
//   - tagged subblock at index 5: points data
//   - tagged ID at index 6: timestamp (ignored)
//   - tagged ID at index 7: move_id (optional, ignored)
func parseLineData(r *bytes.Reader, version byte) (*V6Line, error) {
	line := &V6Line{}

	// Read tool (index 1)
	if _, err := expectTag(r, 1, TAG_BYTE4); err != nil {
		return nil, err
	}
	var tool uint32
	if err := binary.Read(r, binary.LittleEndian, &tool); err != nil {
		return nil, err
	}
	line.Tool = int32(tool)

	// Read color (index 2)
	if _, err := expectTag(r, 2, TAG_BYTE4); err != nil {
		return nil, err
	}
	var color uint32
	if err := binary.Read(r, binary.LittleEndian, &color); err != nil {
		return nil, err
	}
	line.Color = int32(color)

	// Read thickness_scale (index 3)
	if _, err := expectTag(r, 3, TAG_BYTE8); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &line.ThicknessScale); err != nil {
		return nil, err
	}

	// Read starting_length (index 4)
	if _, err := expectTag(r, 4, TAG_BYTE4); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &line.StartingLength); err != nil {
		return nil, err
	}

	// Read points subblock (index 5)
	if _, err := expectTag(r, 5, TAG_LENGTH4); err != nil {
		return nil, err
	}
	var pointsLen uint32
	if err := binary.Read(r, binary.LittleEndian, &pointsLen); err != nil {
		return nil, err
	}

	// Points are 14 bytes each in version 2, 24 bytes in version 1
	pointSize := 14
	if version == 1 {
		pointSize = 24
	}
	numPoints := int(pointsLen) / pointSize

	// Read points
	line.Points = make([]V6Point, numPoints)
	for i := 0; i < numPoints; i++ {
		point, err := parsePoint(r, version)
		if err != nil {
			return nil, err
		}
		line.Points[i] = point
	}

	// Ignore timestamp and move_id (indices 6, 7)
	// They may or may not be present

	return line, nil
}

// parsePoint parses a single point
// Version 2 format (14 bytes):
//   - X (float32, 4 bytes)
//   - Y (float32, 4 bytes)
//   - Speed (uint16, 2 bytes)
//   - Width (uint16, 2 bytes)
//   - Direction (uint8, 1 byte)
//   - Pressure (uint8, 1 byte)
func parsePoint(r *bytes.Reader, version byte) (V6Point, error) {
	var point V6Point

	// Read X (float32)
	if err := binary.Read(r, binary.LittleEndian, &point.X); err != nil {
		return point, err
	}

	// Read Y (float32)
	if err := binary.Read(r, binary.LittleEndian, &point.Y); err != nil {
		return point, err
	}

	if version == 1 {
		// Version 1: float32 values
		var speed, dir, width, pressure float32
		if err := binary.Read(r, binary.LittleEndian, &speed); err != nil {
			return point, err
		}
		if err := binary.Read(r, binary.LittleEndian, &dir); err != nil {
			return point, err
		}
		if err := binary.Read(r, binary.LittleEndian, &width); err != nil {
			return point, err
		}
		if err := binary.Read(r, binary.LittleEndian, &pressure); err != nil {
			return point, err
		}
		// Convert to version 2 format
		point.Speed = uint16(speed * 4)
		point.Width = uint16(width * 4)
		point.Direction = uint8(dir * 255 / (2 * 3.14159))
		point.Pressure = uint8(pressure * 255)
	} else {
		// Version 2: uint16/uint8 values
		if err := binary.Read(r, binary.LittleEndian, &point.Speed); err != nil {
			return point, err
		}
		if err := binary.Read(r, binary.LittleEndian, &point.Width); err != nil {
			return point, err
		}
		if err := binary.Read(r, binary.LittleEndian, &point.Direction); err != nil {
			return point, err
		}
		if err := binary.Read(r, binary.LittleEndian, &point.Pressure); err != nil {
			return point, err
		}
	}

	return point, nil
}

// convertV6Line converts v6 line to standard Line format
func convertV6Line(v6line V6Line) Line {
	line := Line{
		BrushType:  mapV6Tool(v6line.Tool),
		BrushColor: mapV6Color(v6line.Color),
		BrushSize:  BrushSize(v6line.ThicknessScale * 2.0),
		Points:     make([]Point, len(v6line.Points)),
	}

	for i, v6p := range v6line.Points {
		// V6 coordinates are already in the correct range
		line.Points[i] = Point{
			X:         v6p.X,
			Y:         v6p.Y,
			Speed:     float32(v6p.Speed),
			Direction: float32(v6p.Direction),
			Width:     float32(v6p.Width),
			Pressure:  float32(v6p.Pressure),
		}
	}

	return line
}

// mapV6Tool maps v6 tool to BrushType
func mapV6Tool(tool int32) BrushType {
	// V6 tool IDs
	switch tool {
	case 0, 12:
		return BallPointV5 // Brush/Paintbrush
	case 1, 14:
		return TiltPencilV5 // Pencil
	case 2, 15:
		return BallPointV5 // Ballpoint
	case 3, 16:
		return MarkerV5 // Marker
	case 4, 17:
		return FinelinerV5 // Fineliner
	case 5, 18:
		return HighlighterV5 // Highlighter
	case 6:
		return Eraser
	case 7, 13:
		return TiltPencilV5 // Mechanical pencil
	case 8:
		return EraseArea
	case 21:
		return BallPointV5 // Calligraphy
	default:
		return BallPointV5 // Default
	}
}

// mapV6Color maps v6 color to BrushColor
func mapV6Color(color int32) BrushColor {
	switch color {
	case 0:
		return Black
	case 1:
		return Grey
	case 2:
		return White
	default:
		return Black
	}
}

// readVarint reads a variable-length integer
func readVarint(r *bytes.Reader) (uint64, error) {
	var result uint64
	var shift uint

	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}

		result |= uint64(b&0x7F) << shift

		if b&0x80 == 0 {
			break
		}

		shift += 7
		if shift >= 64 {
			return 0, fmt.Errorf("varint overflow")
		}
	}

	return result, nil
}

// readCrdtId reads a CRDT ID (uint8 + varint)
func readCrdtId(r *bytes.Reader) (V6CrdtId, error) {
	var id V6CrdtId

	// Read part1 (uint8)
	if err := binary.Read(r, binary.LittleEndian, &id.Part1); err != nil {
		return id, err
	}

	// Read part2 (varint)
	part2, err := readVarint(r)
	if err != nil {
		return id, err
	}
	id.Part2 = part2

	return id, nil
}

// expectTag reads and validates a tag
func expectTag(r *bytes.Reader, expectedIndex int, expectedType byte) (bool, error) {
	// Read tag varint
	tagValue, err := readVarint(r)
	if err != nil {
		return false, err
	}

	// Extract index and type
	index := int(tagValue >> 4)
	tagType := byte(tagValue & 0x0F)

	if index != expectedIndex || tagType != expectedType {
		return false, fmt.Errorf("unexpected tag: expected index=%d type=0x%x, got index=%d type=0x%x",
			expectedIndex, expectedType, index, tagType)
	}

	return true, nil
}
