package binfile

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type TestBinaryStructure1 struct {
	RecordType          string `bin:":2"`      // 0
	UnitNo              int    `bin:":2"`      // 2
	RackNumber          int    `bin:":4"`      // 4
	CupPosition         int    `bin:":2"`      // 8
	SampleType          string `bin:":1,trim"` // 10
	SampleNo            string `bin:":4"`      // 11
	SampleId            string `bin:":11"`     // 15
	Dummy               string `bin:":4,trim"` // 26
	BlockIdentification string `bin:":1,trim"` // 30

	TestResults []TestResultsStructure `bin:"array:terminator"`
}

type TestResultsStructure struct {
	TestCode   string `bin:":2"`      // 1. 30  2. 43
	TestResult string `bin:":9,trim"` // 1. 32  2. 45
	Flags      string `bin:":2,trim"` // 1. 41  2. 54
	//	Terminator string `bin:":1,terminator"` // 1.     2. 56
}

// POC Unmarshal binary protocol generic simple example
func TestUnmarshalBinary1(t *testing.T) {
	data := "D 03116506 044760722905768    E61     6.40  62      935  "
	var r TestBinaryStructure1
	err := Unmarshal([]byte(data), &r, EncodingUTF8, TimezoneUTC, "\r")

	assert.Nil(t, err)
	assert.Equal(t, "D ", r.RecordType)
	assert.Equal(t, 3, r.UnitNo)
	assert.Equal(t, 1165, r.RackNumber)
	assert.Equal(t, 6, r.CupPosition)
	assert.Equal(t, "", r.SampleType)
	assert.Equal(t, "0447", r.SampleNo)
	assert.Equal(t, "60722905768", r.SampleId)
	assert.Equal(t, "", r.Dummy)
	assert.Equal(t, "E", r.BlockIdentification)

	assert.Equal(t, 2, len(r.TestResults))

	assert.Equal(t, "61", r.TestResults[0].TestCode)
	assert.Equal(t, "6.40", r.TestResults[0].TestResult)
	assert.Equal(t, "", r.TestResults[0].Flags)

	assert.Equal(t, "62", r.TestResults[1].TestCode)
	assert.Equal(t, "935", r.TestResults[1].TestResult)
	assert.Equal(t, "", r.TestResults[1].Flags)

}

type TestBinaryStructure2 struct {
	DMessage []TestBinaryStructure1 `bin:"array:terminator"`
}

// This test inspired by the AU-600 protocol:
// Read two consecutive Data messages
func TestUnmarshalMultipleRecords(t *testing.T) {
	data := "D 03116506 044760722905768    E61     6.40  62      935  \u000DD 03116507 044860722905758    E61     6.86  62      883  "

	var r TestBinaryStructure2
	err := Unmarshal([]byte(data), &r, EncodingUTF8, TimezoneUTC, "\r")

	assert.Equal(t, nil, err)

	assert.Equal(t, 2, len(r.DMessage))

	assert.Equal(t, "D ", r.DMessage[0].RecordType)
	assert.Equal(t, 3, r.DMessage[0].UnitNo)
	assert.Equal(t, 1165, r.DMessage[0].RackNumber)
	assert.Equal(t, 6, r.DMessage[0].CupPosition)
	assert.Equal(t, "", r.DMessage[0].SampleType)
	assert.Equal(t, "0447", r.DMessage[0].SampleNo)
	assert.Equal(t, "60722905768", r.DMessage[0].SampleId)
	assert.Equal(t, "", r.DMessage[0].Dummy)
	assert.Equal(t, "E", r.DMessage[0].BlockIdentification)

	assert.Equal(t, 2, len(r.DMessage[0].TestResults))

	assert.Equal(t, "61", r.DMessage[0].TestResults[0].TestCode)
	assert.Equal(t, "6.40", r.DMessage[0].TestResults[0].TestResult)
	assert.Equal(t, "", r.DMessage[0].TestResults[0].Flags)

	assert.Equal(t, "62", r.DMessage[0].TestResults[1].TestCode)
	assert.Equal(t, "935", r.DMessage[0].TestResults[1].TestResult)
	assert.Equal(t, "", r.DMessage[0].TestResults[1].Flags)

	//--------------------------- 2nd message block
	assert.Equal(t, "D ", r.DMessage[1].RecordType)
	assert.Equal(t, 3, r.DMessage[1].UnitNo)
	assert.Equal(t, 1165, r.DMessage[1].RackNumber)
	assert.Equal(t, 7, r.DMessage[1].CupPosition)
	assert.Equal(t, "", r.DMessage[1].SampleType)
	assert.Equal(t, "0448", r.DMessage[1].SampleNo)
	assert.Equal(t, "60722905758", r.DMessage[1].SampleId)
	assert.Equal(t, "", r.DMessage[1].Dummy)
	assert.Equal(t, "E", r.DMessage[1].BlockIdentification)

	assert.Equal(t, 2, len(r.DMessage[1].TestResults))

	assert.Equal(t, "61", r.DMessage[1].TestResults[0].TestCode)
	assert.Equal(t, "6.86", r.DMessage[1].TestResults[0].TestResult)
	assert.Equal(t, "", r.DMessage[1].TestResults[0].Flags)

	assert.Equal(t, "62", r.DMessage[1].TestResults[1].TestCode)
	assert.Equal(t, "883", r.DMessage[1].TestResults[1].TestResult)
	assert.Equal(t, "", r.DMessage[1].TestResults[1].Flags)
}

type invalidAddressAnnoation struct {
	FieldInvalid string `bin:"-1:"` // invalid without a length
}

type onlyValidAddressAnnotations struct {
	FieldValid string `bin:":2"` // this should pass
}

func TestExpectInvalidAddressAnnotationToFail(t *testing.T) {
	data := "123456"

	var invalid invalidAddressAnnoation
	err := Unmarshal([]byte(data), &invalid, EncodingUTF8, TimezoneUTC, "\r")
	assert.Equal(t, "error processing field 'FieldInvalid' `-1:`: non-struct field must have address annotation", err.Error())

	var onlyValid onlyValidAddressAnnotations
	err = Unmarshal([]byte(data), &onlyValid, EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)
}

type absoluteAddressing struct {
	Field1 string `bin:"0:2"`
	Field2 string `bin:"5:2"`
	Field3 string `bin:":2"`
}

func TestAbsoluteAddressing(t *testing.T) {
	data := "01xxx56abxxx"

	var r absoluteAddressing
	err := Unmarshal([]byte(data), &r, EncodingUTF8, TimezoneUTC, "\r")

	assert.Equal(t, nil, err)

	assert.Equal(t, "01", r.Field1)
	assert.Equal(t, "56", r.Field2)
	assert.Equal(t, "ab", r.Field3)
}

type testUnannotated struct {
	AnnotatedField             string `bin:":2"`
	UnannotatedForeignStructre uuid.UUID
	UnannotatedStringField     string
	UnannotatedIntField        int
	UnanntoatedFloat32Field    float32
	UnannotatedFloat64Field    float64
}

// TEST: bug: unannotated fields in struct did cause an error.
// Unannotated fields should just be skipped
func TestUnanotatedFieldsAreSkipped(t *testing.T) {
	data := "1234567"

	var r testUnannotated
	err := Unmarshal([]byte(data), &r, EncodingUTF8, TimezoneUTC, "")

	assert.Nil(t, err)
	assert.Equal(t, "12", r.AnnotatedField)
	assert.Equal(t, "", r.UnannotatedStringField) // Check that it didnt touch our not-annotated field
	assert.Equal(t, 0, r.UnannotatedIntField)
	assert.Equal(t, float32(0), r.UnanntoatedFloat32Field)
	assert.Equal(t, float64(0), r.UnannotatedFloat64Field)
}

type meArrayElement struct {
	SomeData string
}

type testUnaccisbleFields struct {
	MeIsAccessible                     string `bin:":1"`
	meIsNotAccessibleWithoutAnnotation string
	MeArray                            []meArrayElement
}

// TEST: reprduces a bug that crashed unexported fields
func TestInaccesibleFields(t *testing.T) {
	data := "1234567"

	var r testUnaccisbleFields
	err := Unmarshal([]byte(data), &r, EncodingUTF8, TimezoneUTC, "")

	// suppress warning
	_ = r.meIsNotAccessibleWithoutAnnotation

	assert.Nil(t, err)
	assert.Equal(t, "1", r.MeIsAccessible)
}

type testInnerArrayUnmarshal struct {
	JustAField string `bin:":2"`
	AndAnOther int    `bin:":1"`
}
type testFixedLengthArrayUnmarshal struct {
	StructSlice    []testInnerArrayUnmarshal `bin:"array:3"`
	PrimitiveSlice []int                     `bin:"array:4,:1"`
}

func TestUnmarshalArrayWithFixedLength(t *testing.T) {

	var dataExactSize = "AA0BB1CC20123"

	var readExactSize testFixedLengthArrayUnmarshal
	err := Unmarshal([]byte(dataExactSize), &readExactSize, EncodingUTF8, TimezoneUTC, "")
	assert.Nil(t, err)

	var expectedExactSizeInner = []testInnerArrayUnmarshal{
		{JustAField: "AA", AndAnOther: 0},
		{JustAField: "BB", AndAnOther: 1},
		{JustAField: "CC", AndAnOther: 2},
	}
	assert.Equal(t, 3, len(readExactSize.StructSlice))
	assert.Equal(t, reflect.DeepEqual(readExactSize.StructSlice, expectedExactSizeInner), true)
	assert.Equal(t, 4, len(readExactSize.PrimitiveSlice))
	assert.Equal(t, reflect.DeepEqual(readExactSize.PrimitiveSlice, []int{0, 1, 2, 3}), true)

	//-Zero Padded-------------------------------------------------------------
	var dataZeroPadded = "AA0BB1\x00\x00\x0001\x00\x00"

	var readZeroPadded testFixedLengthArrayUnmarshal
	err = Unmarshal([]byte(dataZeroPadded), &readZeroPadded, EncodingUTF8, TimezoneUTC, "")
	assert.Nil(t, err)

	var expectedZeroPaddedInner = []testInnerArrayUnmarshal{
		{JustAField: "AA", AndAnOther: 0},
		{JustAField: "BB", AndAnOther: 1},
	}
	assert.Equal(t, 2, len(readZeroPadded.StructSlice))
	assert.Equal(t, reflect.DeepEqual(readZeroPadded.StructSlice, expectedZeroPaddedInner), true)
	assert.Equal(t, 2, len(readZeroPadded.PrimitiveSlice))
	assert.Equal(t, reflect.DeepEqual(readZeroPadded.PrimitiveSlice, []int{0, 1}), true)

	//-With Terminator---------------------------------------------------------
	var dataWithTerminator = "AA0BB1CC2\u000D0123"

	var readWithTerminator testFixedLengthArrayUnmarshal
	err = Unmarshal([]byte(dataWithTerminator), &readWithTerminator, EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	var expectedWithTerminatorInner = []testInnerArrayUnmarshal{
		{JustAField: "AA", AndAnOther: 0},
		{JustAField: "BB", AndAnOther: 1},
		{JustAField: "CC", AndAnOther: 2},
	}
	assert.Equal(t, 3, len(readWithTerminator.StructSlice))
	assert.Equal(t, reflect.DeepEqual(readWithTerminator.StructSlice, expectedWithTerminatorInner), true)
	assert.Equal(t, 4, len(readWithTerminator.PrimitiveSlice))
	assert.Equal(t, reflect.DeepEqual(readWithTerminator.PrimitiveSlice, []int{0, 1, 2, 3}), true)

}

type testDynamicArrayUnmarshalUnknownField struct {
	NotHere   string `bin:":1"`
	TheArray1 []int  `bin:"array:Hacaca,:1"`
}

type testDynamicArrayUnmarshalWrongType struct {
	Nope      string `bin:":1"`
	TheArray1 []int  `bin:"array:Nope,:1"`
}

type testDynamicArrayUnmarshalWrongValue struct {
	IncorrectSize int   `bin:":2"`
	TheArray1     []int `bin:"array:IncorrectSize,:1"`
}

type testDynamicArrayUnmarshal struct {
	Nope          string `bin:":1"`
	CorrectSize   int    `bin:":1"`
	IncorrectSize int    `bin:":2"`
	TheArray1     []int  `bin:"array:CorrectSize,:1"`
}

func TestUnmarshalDynamicArray(t *testing.T) {

	var err error

	var data = "A3-1012"
	var result testDynamicArrayUnmarshal

	err = Unmarshal([]byte(data), &result, EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, "A", result.Nope)
	assert.Equal(t, 3, result.CorrectSize)
	assert.Equal(t, -1, result.IncorrectSize)
	assert.Equal(t, reflect.DeepEqual(result.TheArray1, []int{0, 1, 2}), true)

	var dataWrongType = "A012"
	var resultWrongType testDynamicArrayUnmarshalWrongType

	err = Unmarshal([]byte(dataWrongType), &resultWrongType, EncodingUTF8, TimezoneUTC, "\r")
	assert.EqualError(t, err, "invalid type for array size field 'Nope' - should be int")

	var dataWrongValue = "-1012"
	var resultWrongValue testDynamicArrayUnmarshalWrongValue

	err = Unmarshal([]byte(dataWrongValue), &resultWrongValue, EncodingUTF8, TimezoneUTC, "\r")
	assert.EqualError(t, err, "invalid size for array size field 'IncorrectSize'")

	var dataNoField = "A012"
	var resultNoField testDynamicArrayUnmarshalUnknownField

	err = Unmarshal([]byte(dataNoField), &resultNoField, EncodingUTF8, TimezoneUTC, "")
	assert.EqualError(t, err, "unknown field name for array size 'Hacaca'")
}
