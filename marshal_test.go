package binfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --------------------------------------------------------------------------------------------------
type testGenericMarshal struct {
	SomeField1      string  `bin:":2"`
	SomeField2      string  `bin:":4"`
	SomeField3      string  `bin:"8:2"` // add a "jump" of two charachters to force to fill up
	IntField        int     `bin:":3"`
	IntFieldPadding int     `bin:":3,padzero"`
	DecimalField    float32 `bin:":"`
}

func TestMarhalGeneric(t *testing.T) {

	var r testGenericMarshal
	r.SomeField1 = "12"
	r.SomeField2 = "3456"
	r.SomeField3 = "aa"
	r.IntField = 7
	r.IntFieldPadding = 5
	data, err := Marshal(r, 'x', EncodingUTF8, TimezoneEuropeBerlin, "\r")

	assert.Nil(t, err)

	assert.Equal(t, "123456xxaa  7005", data)
}

// --------------------------------------------------------------------------------------------------
func TestMarshalArray(t *testing.T) {
	r := make([]testGenericMarshal, 0)
	r = append(r, testGenericMarshal{SomeField1: "AB", SomeField2: "CDEF"})
	r = append(r, testGenericMarshal{SomeField1: "12", SomeField2: "3456"})

	data, err := Marshal(r, 'x', EncodingUTF8, TimezoneEuropeBerlin, "\r")

	assert.Nil(t, err)
	assert.Equal(t, "ABCDEFxx  \r123456xx  \r", data)
}

// --------------------------------------------------------------------------------------------------
type testFixedLenarrayInnerStruct struct {
	Field1 string `bin:":2"`
}

type testFixedLenArray struct {
	SomeArray []testFixedLenarrayInnerStruct `bin:"array:5"`
}

func TestMarshalArrayWithFixedLength(t *testing.T) {

	FixedLenArray4Elements := testFixedLenArray{
		SomeArray: []testFixedLenarrayInnerStruct{
			{Field1: "N1"},
			{Field1: "N2"},
			{Field1: "N3"},
			{Field1: "N4"},
		},
	}

	FixedLenArray6Elements := testFixedLenArray{
		SomeArray: []testFixedLenarrayInnerStruct{
			{Field1: "N1"},
			{Field1: "N2"},
			{Field1: "N3"},
			{Field1: "N4"},
			{Field1: "N5"},
			{Field1: "N6"},
		},
	}

	// padding zeros for a required length of 5
	data, err := Marshal(FixedLenArray4Elements, 'x', EncodingUTF8, TimezoneEuropeBerlin, "")
	assert.Nil(t, err)
	assert.Equal(t, append([]byte("N1N2N3N4"), []byte{0, 0}...), data)

	// 6 Elements in a 5 elment array expected to b cut to 5 and then terminated
	data, err = Marshal(FixedLenArray6Elements, 'x', EncodingUTF8, TimezoneEuropeBerlin, "_")
	assert.Nil(t, err)
	assert.Equal(t, "N1N2N3N4N5_", string(data))
}

// --------------------------------------------------------------------------------------------------
type testDynamicLenArrayInnerStruct struct {
	Field1 string `bin:":2"`
}

type testDynamicLenArray struct {
	ArrayLen  int                              `bin:":3,padzero"`
	SomeArray []testDynamicLenArrayInnerStruct `bin:"dynamicarray:ArrayLen"` // read the field-length from the field "ArrayLen"
}

// ------ Encoding an array with a given length
func TestMarshalArrayWithDynamicLength(t *testing.T) {
	dyna := testDynamicLenArray{
		ArrayLen: 3,
		SomeArray: []testDynamicLenArrayInnerStruct{
			{Field1: "AB"},
			{Field1: "CD"},
			{Field1: "DE"},
		},
	}

	data, err := Marshal(dyna, 'x', EncodingUTF8, TimezoneEuropeBerlin, "\r")
	assert.Nil(t, err)
	assert.Equal(t, []byte("003ABCDDE\r"), data)
}
