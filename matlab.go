// Package matlab defines readers & writers for working with matlab .mat files
package matlab

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

// DataType represents matlab data types
type DataType uint32

func (d DataType) String() string {
	switch d {
	case DTmiINT8:
		return "miINT8"
	case DTmiUINT8:
		return "miUINT8"
	case DTmiINT16:
		return "miINT16"
	case DTmiUINT16:
		return "miUINT16"
	case DTmiINT32:
		return "miINT32"
	case DTmiUINT32:
		return "miUINT32"
	case DTmiSINGLE:
		return "miSINGLE"
	case DTmiDOUBLE:
		return "miDOUBLE"
	case DTmiINT64:
		return "miINT64"
	case DTmiUINT64:
		return "miUINT64"
	case DTmiMATRIX:
		return "miMATRIX"
	case DTmiCOMPRESSED:
		return "miCOMPRESSED"
	case DTmiUTF8:
		return "miUTF8"
	case DTmiUTF16:
		return "miUTF16"
	case DTmiUTF32:
		return "miUTF32"
	default:
		return "unknown"
	}
}

// NumBytes returns the number of bytes needed to represent the datatype
func (d DataType) NumBytes() int {
	switch d {
	case DTmiINT8:
		return 1
	case DTmiUINT8:
		return 1
	case DTmiUTF8:
		return 1
	case DTmiINT16:
		return 2
	case DTmiUINT16:
		return 2
	case DTmiUTF16:
		return 2
	case DTmiINT32:
		return 4
	case DTmiUINT32:
		return 4
	case DTmiUTF32:
		return 4
	case DTmiSINGLE:
		return 4
	case DTmiDOUBLE:
		return 8
	case DTmiINT64:
		return 8
	case DTmiUINT64:
		return 8
	case DTmiMATRIX:
	case DTmiCOMPRESSED:
	default:
	}
	panic("Cannot get NumBytes of variable length type: " + d.String())
}

// Data Types as specified according to byte indicators
const (
	DataTypeUnknown DataType = iota // errored data type
	DTmiINT8                        // 8 bit, signed
	DTmiUINT8                       // 8 bit, unsigned
	DTmiINT16                       // 16-bit, signed
	DTmiUINT16                      // 16-bit, unsigned
	DTmiINT32                       // 32-bit, signed
	DTmiUINT32                      // 32-bit, unsigned
	DTmiSINGLE                      // IEEE® 754 single format
	_
	DTmiDOUBLE // IEEE 754 double format
	_
	_
	DTmiINT64      // 64-bit, signed
	DTmiUINT64     // 64-bit, unsigned
	DTmiMATRIX     // MATLAB array
	DTmiCOMPRESSED // Compressed Data
	DTmiUTF8       // Unicode UTF-8 Encoded Character Data
	DTmiUTF16      // Unicode UTF-16 Encoded Character Data
	DTmiUTF32      // Unicode UTF-32 Encoded Character Data
)

// File represents a .mat matlab file
type File struct {
	Header *Header
	r      io.Reader
	w      io.Writer

	hasReadAll bool
	vars       map[string]*Matrix
}

// Header is a matlab .mat file header
type Header struct {
	Level     string
	Platform  string
	Created   time.Time
	Endianess binary.ByteOrder
}

// String implements the stringer interface for Header
// with the standard .mat file prefix (without the filler bytes)
func (h *Header) String() string {
	return fmt.Sprintf("MATLAB %s MAT-file, Platform: %s, Created on: %s", h.Level, h.Platform, h.Created.Format(time.ANSIC))
}

// NewFileFromReader creates a file from a reader and attempts to read
// the header
func NewFileFromReader(r io.Reader) (f *File, err error) {
	f = &File{r: r, vars: map[string]*Matrix{}}
	err = f.readHeader()
	return
}

const (
	headerLen                = 128
	headerTextLen            = 116
	headerSubsystemOffsetLen = 8
	headerFlagLen            = 4
)

func (f *File) readHeader() (err error) {
	var buf []byte
	h := &Header{}
	f.Header = h

	// read description
	if buf, err = readAllBytes(headerTextLen, f.r); err != nil {
		return
	}

	r := bufio.NewReader(bytes.NewBuffer(buf))

	if prefix, err := r.ReadBytes(' '); err != nil {
		return err
	} else if !bytes.Equal(prefix, []byte("MATLAB ")) {
		return fmt.Errorf("not a valid .mat file")
	}

	if h.Level, err = r.ReadString(' '); err != nil {
		return err
	}

	h.Level = strings.TrimSpace(h.Level)
	if h.Level != "5.0" {
		return fmt.Errorf("can only read matlab level 5 files")
	}

	if _, err = r.Discard(len("MAT-file Platform: ")); err != nil {
		return
	}

	if h.Platform, err = r.ReadString(','); err != nil {
		return
	}
	h.Platform = strings.TrimRight(h.Platform, ",")

	if _, err = r.Discard(len(" Created on: ")); err != nil {
		return
	}

	date := make([]byte, 24)
	if _, err = r.Read(date); err != nil {
		return
	}

	if h.Created, err = time.Parse(time.ANSIC, strings.TrimSpace(string(date))); err != nil {
		// Tolerate bad parsing. .mat files created by Octave doesn't seem to conform to the format
	}

	if _, err = readAllBytes(headerSubsystemOffsetLen, f.r); err != nil {
		return
	}

	if buf, err = readAllBytes(headerFlagLen, f.r); err != nil {
		return
	}

	byteOrder := string(buf[2:4])
	if byteOrder == "MI" {
		h.Endianess = binary.BigEndian
	} else if byteOrder == "IM" {
		h.Endianess = binary.LittleEndian
	} else {
		return fmt.Errorf("invalid byte order setting: %s", byteOrder)
	}

	return nil
}

func readAllBytes(p int, rdr io.Reader) (buf []byte, err error) {
	var (
		n int
		r []byte
	)
	remaining := p
	for remaining > 0 {
		r = make([]byte, p)
		n, err = rdr.Read(r)
		if err != nil {
			if err.Error() == "EOF" {
				if remaining-n == 0 {
					// Finish reading
					return append(buf, r[:n]...), nil
				} else if p == remaining {
					// Didn't read anything
					return r, err
				} else {
					// Bad unpacking
					return r, fmt.Errorf("EOF reached but we're supposed to read %d more bytes", remaining)
				}
			}
			return
		}
		buf = append(buf, r[:n]...)
		remaining -= n
	}
	return
}

func (f *File) readAll() error {
	if f.hasReadAll == true {
		return nil
	}
	f.hasReadAll = true
	elements, err := readAllElements(f.Header.Endianess, f.r)
	if err != nil {
		return err
	}
	for _, v := range elements {
		if v.Type() != DTmiMATRIX {
			panic("This library assumes top level elements are either Matrix or Compressed. Please file an issue")
		}
		m := v.(*Matrix)
		f.vars[m.Name] = m
	}
	return nil
}

// GetVar returns the variable in the mat file
func (f *File) GetVar(name string) (*Matrix, bool) {
	if !f.hasReadAll {
		if err := f.readAll(); err != nil {
			return nil, false
		}
	}
	vars, found := f.vars[name]
	return vars, found
}

// GetVarsNames returns the list of variables in the given mat file
func (f *File) GetVarsNames() []string {
	if !f.hasReadAll {
		if err := f.readAll(); err != nil {
			return nil
		}
	}
	var res []string
	for n := range f.vars {
		res = append(res, n)
	}
	return res
}

func readElement(bo binary.ByteOrder, r io.Reader) (el Element, err error) {
	sde, dt, p, err := readTag(bo, r)
	if err != nil {
		return nil, err
	}
	// if small element, p will be 0, bail early
	if sde != nil {
		return sde, nil
	}
	switch dt {
	case DTmiCOMPRESSED:
		// data is compressed, use zlib reader
		buf, err := readAllBytes(p, r)
		if err != nil {
			return nil, err
		}
		cr, err := zlib.NewReader(bytes.NewBuffer(buf))
		if err != nil {
			return nil, err
		}
		defer cr.Close()
		allElements, err := readAllElements(bo, cr)
		if err != nil {
			return nil, err
		}
		if len(allElements) != 1 {
			panic("This library assumes compressed elements have exactly one sub element")
		}
		return allElements[0], nil
	case DTmiMATRIX:
		data, err := readAllBytes(p, r)
		if err != nil {
			return nil, err
		}
		return miMatrix(bo, data)
	default:
		p = padTo64Bit(p)
		buf, err := readAllBytes(p, r)
		if err != nil {
			return nil, err
		}
		content, err := parseContent(dt, bo, buf)
		if err != nil {
			return nil, err
		}
		return &subElement{typ: dt, value: content}, err
	}
}

func readAllElements(bo binary.ByteOrder, r io.Reader) ([]Element, error) {
	var res []Element
	for {
		el, err := readElement(bo, r)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
		res = append(res, el)
	}
	return res, nil

}

// Reads the first 8 bytes. The 8 bytes can be one of two formats: Normal and small data element (sde) format.
// Note that contrary to what the specs says, you have to consider endianness before parsing the first type bytes.
func readTag(bo binary.ByteOrder, r io.Reader) (sde *smallDataElement, typ DataType, len int, err error) {
	buf, err := readAllBytes(8, r)
	if err != nil {
		return
	}
	sdeLen, sdeType := binary.LittleEndian.Uint16(buf[2:4]), binary.LittleEndian.Uint16(buf[0:2])
	if bo == binary.BigEndian {
		sdeLen, sdeType = sdeType, sdeLen
	}
	if sdeLen != 0 {
		// handle small data element
		dt := DataType(sdeType)
		numEl := int(sdeLen) / dt.NumBytes()
		sdeContent, err := parseMulti(dt, bo, buf[4:], numEl)
		if err != nil {
			return nil, DataTypeUnknown, 0, err
		}
		return &smallDataElement{typ: dt, value: sdeContent}, typ, 0, nil
	}
	// normal type
	dataType := DataType(bo.Uint32(buf[:4]))
	len = int(bo.Uint32(buf[4:]))
	return nil, dataType, len, nil
}

func parseMulti(t DataType, bo binary.ByteOrder, data []byte, len int) ([]interface{}, error) {
	res := make([]interface{}, len)
	for i := 0; i < len; i++ {
		i2, err := parseContent(t, bo, data[i*t.NumBytes():(i+1)*t.NumBytes()])
		if err != nil {
			return nil, err
		}
		res[i] = i2
	}
	return res, nil
}

func parseContent(t DataType, bo binary.ByteOrder, data []byte) (interface{}, error) {
	switch t {
	case DTmiINT8:
		return int8(data[0]), nil
	case DTmiUINT8:
		return uint8(data[0]), nil
	case DTmiINT16:
		return int16(bo.Uint16(data)), nil
	case DTmiUINT16:
		return bo.Uint16(data), nil
	case DTmiINT32:
		return int32(bo.Uint32(data)), nil
	case DTmiUINT32:
		return bo.Uint32(data), nil
	case DTmiSINGLE:
		return math.Float32frombits(bo.Uint32(data)), nil
	case DTmiDOUBLE:
		return math.Float64frombits(bo.Uint64(data)), nil
	case DTmiINT64:
		return int64(bo.Uint64(data)), nil
	case DTmiUINT64:
		return bo.Uint64(data), nil
	case DTmiMATRIX:
		panic("Should not be parsing matrix here")
	case DTmiUTF8:
		r, _ := utf8.DecodeRune(data)
		return r, nil
	case DTmiUTF16:
		decode := utf16.Decode([]uint16{bo.Uint16(data)})
		return decode[0], nil
	case DTmiUTF32:
		panic("Not supported")
	case DTmiCOMPRESSED:
		panic("should not be parsing compressed data type here")
	default:
		return nil, fmt.Errorf("cannot parseContent data type: %s. Probably just need to implement this", t)
	}
}

func miMatrix(bo binary.ByteOrder, data []byte) (*Matrix, error) {
	r := bytes.NewBuffer(data)
	flags, class, err := arrayFlags(bo, r)
	if err != nil {
		return nil, err
	}
	dim, err := dimensionsArray(bo, r)
	if err != nil {
		return nil, err
	}
	name, err := arrayName(bo, r)
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}

	var res []interface{}
	switch class {
	case mxSPARSE:
		panic("Sparse matrix unsupported") // has 6 sub elements
	case mxCELL: // has 4 sub elements. Each cell is also a miMatrix
		elements, err := readAllElements(bo, r)
		if err != nil {
			return nil, err
		}
		for _, e := range elements {
			// we know they are matrices
			res = append(res, e.(*Matrix))
		}
	case mxSTRUCT:
		panic("Struct matrix unsupported") // has 6 sub elements
	case mxOBJECT:
		panic("Object matrix unsupported") // has 7 sub elements
	default: // 4 elements: Numeric and character array. Pass through
		pr, err := readNumericalData(bo, r)
		if err != nil {
			return nil, err
		}
		if flags.isComplex {
			if _, err := readNumericalData(bo, r); err != nil && err.Error() != "EOF" {
				return nil, err
			}
			// TODO: Handle returning of complex numbers
		}
		res = pr.Value().([]interface{})
	}
	return &Matrix{
		Name:      name,
		flags:     flags,
		Class:     class,
		Dimension: dim,
		value:     res,
	}, nil
}

// flags indicating whether the numeric data is complex, global or logical. See 1-16 of specs.
type Flags struct {
	isLogical bool
	isComplex bool
	isGlobal  bool
}

// Docs is wrong about this. This is packed as two blocks of uint16. The first u16 in the data is for flags and class
// and the second is for sparse matrix.
func arrayFlags(bo binary.ByteOrder, r io.Reader) (flags Flags, class mxClass, err error) {
	_, dt, p, err := readTag(bo, r)
	if err != nil {
		return
	}
	if dt != DTmiUINT32 {
		err = fmt.Errorf("invalid matrix, the array flags sub element in a matrix should have tag of type %s\n", DTmiUINT32)
		return
	}
	if p != 8 {
		err = fmt.Errorf("invalid matrix, the size of array tag should be 8 bytes. Got %d bytes instead", p)
		return
	}
	buf, err := readAllBytes(8, r)
	if err != nil {
		return
	}
	// NonZeroMax is used to indicate the maximum number of nonzero array elements in the sparse array
	flagsAndClass, nonZeroMax := binary.LittleEndian.Uint16(buf[:4]), binary.LittleEndian.Uint16(buf[4:])
	if bo == binary.BigEndian {
		flagsAndClass, nonZeroMax = nonZeroMax, flagsAndClass
	}
	flags = Flags{
		isLogical: flagsAndClass>>9 == 1,
		isGlobal:  flagsAndClass>>10 == 1,
		isComplex: flagsAndClass>>11 == 1,
	}
	class = mxClass(uint8(flagsAndClass & 0xFF))
	return
}

// Page 1-10 of spec says the value of num bytes field does not include padding for types other than matrix.
// This function returns the number of bytes to read for an element that may or may not have padding.
func padTo64Bit(p int) int {
	offset := 0
	if p%8 != 0 {
		offset = 1
	}
	return ((p / 8) + offset) * 8
}

func dimensionsArray(bo binary.ByteOrder, r io.Reader) ([]int32, error) {
	// Can't be a SDE
	_, dt, p, err := readTag(bo, r)
	if err != nil {
		return nil, err
	}
	if dt != DTmiINT32 {
		return nil, fmt.Errorf("invalid data type. Expects dimension sub element to have type int32, got %s instead", dt)
	}
	buf, err := readAllBytes(padTo64Bit(p), r)
	if err != nil {
		return nil, err
	}

	dimsr := bytes.NewBuffer(buf)
	sBuf := make([]byte, 4)
	dim := make([]int32, p/4)
	for i := 0; i < p/4; i++ {
		if _, err := dimsr.Read(sBuf); err != nil {
			return nil, err
		}
		dim[i] = int32(bo.Uint32(sBuf))
	}
	return dim, nil
}

// Note that array name can be empty!
func arrayName(bo binary.ByteOrder, r io.Reader) (string, error) {
	sde, dt, p, err := readTag(bo, r)
	if err != nil {
		return "", err
	}
	if sde != nil {
		t := sde.Value().([]interface{})
		n := make([]byte, len(t))
		for i, v := range t {
			n[i] = byte(v.(int8))
		}
		return string(n), nil
	}
	if dt != DTmiINT8 {
		return "", fmt.Errorf("invalid data type. Expects array name sub element to have type int8, got %s instead", dt)
	}
	data, err := readAllBytes(padTo64Bit(p), r)
	return string(data[:p]), err
}

// This can read the real part of imaginary part sub elements of a matrix
func readNumericalData(bo binary.ByteOrder, r io.Reader) (Element, error) {
	sde, dt, numBytes, err := readTag(bo, r)
	if err != nil {
		return nil, err
	}
	// SDE
	if numBytes == 0 {
		return sde, nil
	}
	data, err := readAllBytes(numBytes, r)
	if err != nil {
		return nil, err
	}
	numElements := numBytes / dt.NumBytes()
	multi, err := parseMulti(dt, bo, data, numElements)
	if err != nil {
		return nil, err
	}
	return smallDataElement{
		typ:   dt,
		value: multi,
	}, nil
}

type mxClass uint8

func (c mxClass) String() string {
	switch c {
	case mxCELL:
		return "Cell array"
	case mxSTRUCT:
		return "Structure"
	case mxOBJECT:
		return "Object"
	case mxCHAR:
		return "Character array"
	case mxSPARSE:
		return "Sparse array"
	case mxDOUBLE:
		return "Double precision array"
	case mxSINGLE:
		return "Single precision array"
	case mxINT8:
		return "8-bit, signed integer"
	case mxUINT8:
		return "8-bit, unsigned integer"
	case mxINT16:
		return "16-bit, signed integer"
	case mxUINT16:
		return "16-bit, unsigned integer"
	case mxINT32:
		return "32-bit, signed integer"
	case mxUINT32:
		return "32-bit, unsigned integer"
	case mxINT64:
		return "64-bit, signed integer"
	case mxUINT64:
		return "64-bit, unsigned integer"
	default:
		return "unknown"
	}
}

// MATLAB Array Types (Classes)
const (
	mxUNKNOWN mxClass = iota
	mxCELL            // Cell array
	mxSTRUCT          // Structure
	mxOBJECT          // Object
	mxCHAR            // Character array
	mxSPARSE          // Sparse array *NB: don't use*
	mxDOUBLE          // Double precision array
	mxSINGLE          // Single precision array
	mxINT8            // 8-bit, signed integer
	mxUINT8           // 8-bit, unsigned integer
	mxINT16           // 16-bit, signed integer
	mxUINT16          // 16-bit, unsigned integer
	mxINT32           // 32-bit, signed integer
	mxUINT32          // 32-bit, unsigned integer
	mxINT64           // 64-bit, signed integer
	mxUINT64          // 64-bit, unsigned integer
)

func writeHeader(w io.Writer, h *Header) error {
	return fmt.Errorf("not finished")
}

// WriteElement writes a single element to a file's writer
func (f *File) WriteElement(e *Element) error {
	return fmt.Errorf("not finished")
}
