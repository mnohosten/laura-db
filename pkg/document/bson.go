package document

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Encoder encodes documents to BSON format
type Encoder struct {
	buf *bytes.Buffer
}

// NewEncoder creates a new BSON encoder
func NewEncoder() *Encoder {
	return &Encoder{
		buf: new(bytes.Buffer),
	}
}

// Encode encodes a document to BSON format
// BSON format: [4-byte size][elements...][0x00 terminator]
// Element format: [1-byte type][cstring key][value]
func (e *Encoder) Encode(doc *Document) ([]byte, error) {
	e.buf.Reset()

	// Reserve space for document size
	sizePos := e.buf.Len()
	binary.Write(e.buf, binary.LittleEndian, int32(0))

	// Encode each field
	for _, key := range doc.Keys() {
		value, _ := doc.GetValue(key)
		if err := e.encodeElement(key, value); err != nil {
			return nil, fmt.Errorf("failed to encode field %s: %w", key, err)
		}
	}

	// Write terminator
	e.buf.WriteByte(0x00)

	// Write document size at the beginning
	data := e.buf.Bytes()
	binary.LittleEndian.PutUint32(data[sizePos:], uint32(len(data)))

	return data, nil
}

// encodeElement encodes a single document element
func (e *Encoder) encodeElement(key string, value *Value) error {
	// Write type
	e.buf.WriteByte(byte(value.Type))

	// Write key as C-string (null-terminated)
	e.buf.WriteString(key)
	e.buf.WriteByte(0x00)

	// Write value based on type
	switch value.Type {
	case TypeNull:
		// No data for null
	case TypeBoolean:
		if value.Data.(bool) {
			e.buf.WriteByte(0x01)
		} else {
			e.buf.WriteByte(0x00)
		}
	case TypeInt32:
		binary.Write(e.buf, binary.LittleEndian, value.Data.(int32))
	case TypeInt64:
		binary.Write(e.buf, binary.LittleEndian, value.Data.(int64))
	case TypeFloat64:
		binary.Write(e.buf, binary.LittleEndian, value.Data.(float64))
	case TypeString:
		str := value.Data.(string)
		// String: [4-byte length including null][string bytes][0x00]
		binary.Write(e.buf, binary.LittleEndian, int32(len(str)+1))
		e.buf.WriteString(str)
		e.buf.WriteByte(0x00)
	case TypeBinary:
		data := value.Data.([]byte)
		// Binary: [4-byte length][subtype][data]
		binary.Write(e.buf, binary.LittleEndian, int32(len(data)))
		e.buf.WriteByte(0x00) // Generic binary subtype
		e.buf.Write(data)
	case TypeObjectID:
		id := value.Data.(ObjectID)
		e.buf.Write(id[:])
	case TypeArray:
		// Array is encoded as a document with numeric keys
		arr := value.Data.([]interface{})
		arrDoc := NewDocument()
		for i, item := range arr {
			arrDoc.Set(fmt.Sprintf("%d", i), item)
		}
		arrData, err := NewEncoder().Encode(arrDoc)
		if err != nil {
			return err
		}
		e.buf.Write(arrData)
	case TypeDocument:
		var subDoc *Document
		switch v := value.Data.(type) {
		case *Document:
			subDoc = v
		case map[string]interface{}:
			subDoc = NewDocumentFromMap(v)
		default:
			return fmt.Errorf("invalid document type: %T", value.Data)
		}
		subData, err := NewEncoder().Encode(subDoc)
		if err != nil {
			return err
		}
		e.buf.Write(subData)
	case TypeTimestamp:
		binary.Write(e.buf, binary.LittleEndian, value.Data.(int64))
	default:
		return fmt.Errorf("unsupported type: %v", value.Type)
	}

	return nil
}

// Decoder decodes BSON data to documents
type Decoder struct {
	reader *bytes.Reader
}

// NewDecoder creates a new BSON decoder
func NewDecoder(data []byte) *Decoder {
	return &Decoder{
		reader: bytes.NewReader(data),
	}
}

// Decode decodes BSON data to a document
func (d *Decoder) Decode() (*Document, error) {
	doc := NewDocument()

	// Read document size
	var size int32
	if err := binary.Read(d.reader, binary.LittleEndian, &size); err != nil {
		return nil, fmt.Errorf("failed to read document size: %w", err)
	}

	// Read elements until terminator
	for {
		// Read element type
		typeByte, err := d.reader.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("failed to read element type: %w", err)
		}

		// Check for terminator
		if typeByte == 0x00 {
			break
		}

		elemType := Type(typeByte)

		// Read key (C-string)
		key, err := d.readCString()
		if err != nil {
			return nil, fmt.Errorf("failed to read key: %w", err)
		}

		// Read value
		value, err := d.decodeValue(elemType)
		if err != nil {
			return nil, fmt.Errorf("failed to decode value for key %s: %w", key, err)
		}

		doc.Set(key, value)
	}

	return doc, nil
}

// readCString reads a null-terminated string
func (d *Decoder) readCString() (string, error) {
	var buf bytes.Buffer
	for {
		b, err := d.reader.ReadByte()
		if err != nil {
			return "", err
		}
		if b == 0x00 {
			break
		}
		buf.WriteByte(b)
	}
	return buf.String(), nil
}

// decodeValue decodes a value based on its type
func (d *Decoder) decodeValue(t Type) (interface{}, error) {
	switch t {
	case TypeNull:
		return nil, nil
	case TypeBoolean:
		b, err := d.reader.ReadByte()
		return b != 0x00, err
	case TypeInt32:
		var v int32
		err := binary.Read(d.reader, binary.LittleEndian, &v)
		return v, err
	case TypeInt64:
		var v int64
		err := binary.Read(d.reader, binary.LittleEndian, &v)
		return v, err
	case TypeFloat64:
		var v float64
		err := binary.Read(d.reader, binary.LittleEndian, &v)
		return v, err
	case TypeString:
		var length int32
		if err := binary.Read(d.reader, binary.LittleEndian, &length); err != nil {
			return nil, err
		}
		strBytes := make([]byte, length-1) // -1 for null terminator
		if _, err := io.ReadFull(d.reader, strBytes); err != nil {
			return nil, err
		}
		d.reader.ReadByte() // Read null terminator
		return string(strBytes), nil
	case TypeBinary:
		var length int32
		if err := binary.Read(d.reader, binary.LittleEndian, &length); err != nil {
			return nil, err
		}
		d.reader.ReadByte() // Read subtype
		data := make([]byte, length)
		if _, err := io.ReadFull(d.reader, data); err != nil {
			return nil, err
		}
		return data, nil
	case TypeObjectID:
		var id ObjectID
		if _, err := io.ReadFull(d.reader, id[:]); err != nil {
			return nil, err
		}
		return id, nil
	case TypeArray:
		// Read array as document
		currentPos, _ := d.reader.Seek(0, io.SeekCurrent)
		var size int32
		binary.Read(d.reader, binary.LittleEndian, &size)
		d.reader.Seek(currentPos, io.SeekStart)

		docBytes := make([]byte, size)
		if _, err := io.ReadFull(d.reader, docBytes); err != nil {
			return nil, err
		}

		arrDoc, err := NewDecoder(docBytes).Decode()
		if err != nil {
			return nil, err
		}

		// Convert document to array
		arr := make([]interface{}, arrDoc.Len())
		for i := 0; i < arrDoc.Len(); i++ {
			key := fmt.Sprintf("%d", i)
			if v, ok := arrDoc.Get(key); ok {
				arr[i] = v
			}
		}
		return arr, nil
	case TypeDocument:
		currentPos, _ := d.reader.Seek(0, io.SeekCurrent)
		var size int32
		binary.Read(d.reader, binary.LittleEndian, &size)
		d.reader.Seek(currentPos, io.SeekStart)

		docBytes := make([]byte, size)
		if _, err := io.ReadFull(d.reader, docBytes); err != nil {
			return nil, err
		}

		return NewDecoder(docBytes).Decode()
	case TypeTimestamp:
		var v int64
		err := binary.Read(d.reader, binary.LittleEndian, &v)
		return v, err
	default:
		return nil, fmt.Errorf("unsupported type: %v", t)
	}
}
