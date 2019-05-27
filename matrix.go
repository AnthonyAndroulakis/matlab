package matlab

type Matrix struct {
	Name      string
	Dimension []int32 // at least length 2
	flags     Flags
	Class     mxClass
	value     []interface{}
}

// hint to the compiler
var _ Element = &Matrix{}

func (m *Matrix) Type() DataType {
	return DTmiMATRIX
}

func (m *Matrix) Value() interface{} {
	return m.value
}

// IntArray is a convenience method to extract the matrix value as []int64. Warning: It panics if the matlab class
// is not an integer type
func (m *Matrix) IntArray() []int64 {
	var res []int64
	for _, e := range m.value {
		switch m.Class {
		case mxINT8:
			res = append(res, int64(e.(int8)))
		case mxINT16:
			res = append(res, int64(e.(int16)))
		case mxINT32:
			res = append(res, int64(e.(int32)))
		case mxINT64:
			res = append(res, int64(e.(int64)))
		case mxUINT8:
			res = append(res, int64(e.(uint8)))
		case mxUINT16:
			res = append(res, int64(e.(uint16)))
		case mxUINT32:
			res = append(res, int64(e.(uint32)))
		case mxUINT64:
			res = append(res, int64(e.(uint64)))
		default:
			panic("unable to convert matrix to int64 array")
		}
	}
	return res
}

// DoubleArray is a convenience method to extract the matrix value as []float64. Warning: It panics if the matlab class
// is not Double or Single
func (m *Matrix) DoubleArray() []float64 {
	var res []float64
	for _, e := range m.value {
		switch m.Class {
		case mxDOUBLE:
			res = append(res, e.(float64))
		case mxSINGLE:
			res = append(res, float64(e.(float32)))
		default:
			panic("unable to convert matrix to double array")
		}
	}
	return res
}
