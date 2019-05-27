package matlab

// Element is a parsed matlab data element
type Element interface {
	Type() DataType
	Value() interface{}
}

// Represents a parsed element that can fit into 8 bytes
type smallDataElement struct {
	typ   DataType
	value interface{}
}

var _ Element = smallDataElement{}

func (e smallDataElement) Type() DataType {
	return e.typ
}

func (e smallDataElement) Value() interface{} {
	return e.value
}

// Represents a normal element that takes up more than 8 bytes. This block aligns to 64 bits.
type subElement struct {
	typ   DataType
	value interface{}
}

var _ Element = &subElement{}

func (e *subElement) Type() DataType {
	return e.typ
}

func (e *subElement) Value() interface{} {
	return e.value
}
