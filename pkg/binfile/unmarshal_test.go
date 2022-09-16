package binfile

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

//
//-General example structure---------------------------------------------------

type testGeneralStructureUnmarshal struct {
	RecordType          string `bin:":2"`      // 0
	UnitNo              int    `bin:":2"`      // 2
	RackNumber          int    `bin:":4"`      // 4
	CupPosition         int    `bin:":2"`      // 8
	SampleType          string `bin:":1,trim"` // 10
	SampleNo            string `bin:":4"`      // 11
	SampleId            string `bin:":11"`     // 15
	Dummy               string `bin:":4,trim"` // 26
	BlockIdentification string `bin:":1,trim"` // 30

	TestResults []testTestResultStructureUnmarshal `bin:"array:terminator"`
}

type testTestResultStructureUnmarshal struct {
	TestCode   string `bin:":2"`      // 1. 30  2. 43
	TestResult string `bin:":9,trim"` // 1. 32  2. 45
	Flags      string `bin:":2,trim"` // 1. 41  2. 54
}

func TestUnmarshalGeneralStructure(t *testing.T) {

	inputData := []byte("D 03116506 044760722905768    E61     6.40  62      935  ")

	var result testGeneralStructureUnmarshal
	err := Unmarshal(inputData, &result, EncodingUTF8, TimezoneUTC, "\r")

	assert.Nil(t, err)
	assert.Equal(t, "D ", result.RecordType)
	assert.Equal(t, 3, result.UnitNo)
	assert.Equal(t, 1165, result.RackNumber)
	assert.Equal(t, 6, result.CupPosition)
	assert.Equal(t, "", result.SampleType)
	assert.Equal(t, "0447", result.SampleNo)
	assert.Equal(t, "60722905768", result.SampleId)
	assert.Equal(t, "", result.Dummy)
	assert.Equal(t, "E", result.BlockIdentification)

	assert.Equal(t, 2, len(result.TestResults))

	assert.Equal(t, "61", result.TestResults[0].TestCode)
	assert.Equal(t, "6.40", result.TestResults[0].TestResult)
	assert.Equal(t, "", result.TestResults[0].Flags)

	assert.Equal(t, "62", result.TestResults[1].TestCode)
	assert.Equal(t, "935", result.TestResults[1].TestResult)
	assert.Equal(t, "", result.TestResults[1].Flags)
}

//
//-Multiple example structure in array-----------------------------------------

type testMultipleRecordsUnmarshal struct {
	DMessage []testGeneralStructureUnmarshal `bin:"array:terminator"`
}

// This test inspired by the AU-600 protocol:
// Read two consecutive Data messages
func TestUnmarshalMultipleRecords(t *testing.T) {

	inputData := []byte("D 03116506 044760722905768    E61     6.40  62      935  \u000DD 03116507 044860722905758    E61     6.86  62      883  ")

	var result testMultipleRecordsUnmarshal
	err := Unmarshal(inputData, &result, EncodingUTF8, TimezoneUTC, "\r")

	assert.Equal(t, nil, err)

	assert.Equal(t, 2, len(result.DMessage))

	assert.Equal(t, "D ", result.DMessage[0].RecordType)
	assert.Equal(t, 3, result.DMessage[0].UnitNo)
	assert.Equal(t, 1165, result.DMessage[0].RackNumber)
	assert.Equal(t, 6, result.DMessage[0].CupPosition)
	assert.Equal(t, "", result.DMessage[0].SampleType)
	assert.Equal(t, "0447", result.DMessage[0].SampleNo)
	assert.Equal(t, "60722905768", result.DMessage[0].SampleId)
	assert.Equal(t, "", result.DMessage[0].Dummy)
	assert.Equal(t, "E", result.DMessage[0].BlockIdentification)

	assert.Equal(t, 2, len(result.DMessage[0].TestResults))

	assert.Equal(t, "61", result.DMessage[0].TestResults[0].TestCode)
	assert.Equal(t, "6.40", result.DMessage[0].TestResults[0].TestResult)
	assert.Equal(t, "", result.DMessage[0].TestResults[0].Flags)

	assert.Equal(t, "62", result.DMessage[0].TestResults[1].TestCode)
	assert.Equal(t, "935", result.DMessage[0].TestResults[1].TestResult)
	assert.Equal(t, "", result.DMessage[0].TestResults[1].Flags)

	//-2nd message block-------------------------------------------------------
	assert.Equal(t, "D ", result.DMessage[1].RecordType)
	assert.Equal(t, 3, result.DMessage[1].UnitNo)
	assert.Equal(t, 1165, result.DMessage[1].RackNumber)
	assert.Equal(t, 7, result.DMessage[1].CupPosition)
	assert.Equal(t, "", result.DMessage[1].SampleType)
	assert.Equal(t, "0448", result.DMessage[1].SampleNo)
	assert.Equal(t, "60722905758", result.DMessage[1].SampleId)
	assert.Equal(t, "", result.DMessage[1].Dummy)
	assert.Equal(t, "E", result.DMessage[1].BlockIdentification)

	assert.Equal(t, 2, len(result.DMessage[1].TestResults))

	assert.Equal(t, "61", result.DMessage[1].TestResults[0].TestCode)
	assert.Equal(t, "6.86", result.DMessage[1].TestResults[0].TestResult)
	assert.Equal(t, "", result.DMessage[1].TestResults[0].Flags)

	assert.Equal(t, "62", result.DMessage[1].TestResults[1].TestCode)
	assert.Equal(t, "883", result.DMessage[1].TestResults[1].TestResult)
	assert.Equal(t, "", result.DMessage[1].TestResults[1].Flags)
}

//
//-Top-level Array-------------------------------------------------------------

type testTopLevelArrayInnerUnmarshal struct {
	SomeField1 string `bin:":2"`
	SomeField2 string `bin:":4"`
	SomeField3 string `bin:"8:2"`
}

func TestUnmarshalTopLevelArray(t *testing.T) {

	var inputData = []byte("ABCDEFxx  \r123456xx  \r")
	var err error

	var result []testTopLevelArrayInnerUnmarshal
	err = Unmarshal(inputData, &result, EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, 2, len(result))

	assert.Equal(t, "AB", result[0].SomeField1)
	assert.Equal(t, "CDEF", result[0].SomeField2)
	assert.Equal(t, "  ", result[0].SomeField3)

	assert.Equal(t, "12", result[1].SomeField1)
	assert.Equal(t, "3456", result[1].SomeField2)
	assert.Equal(t, "  ", result[1].SomeField3)
}

//
//-Annotation presence---------------------------------------------------------

type testUnannotatedValidUnmarshal struct {
	UnannotatedForeignStructre uuid.UUID
	UnannotatedStringField     string
	AnnotatedField             string `bin:":2"`
	unannotatedUnexportedield  int
	UnannotatedStruct          testUnannotatedValidInnerUnmarshal
}

type testUnannotatedValidInnerUnmarshal struct {
	JustAField string `bin:":2"`
}

type testUnannotatedInvalidUnmarshal struct {
	unexportedAnnotatedField string `bin:":2"`
}

// TEST: bug: unannotated fields in struct did cause an erroresult.
// Unannotated fields should just be skipped
func TestUnmarshalUnanotatedFieldsAreSkipped(t *testing.T) {

	var inputData = []byte("1234")
	var err error

	var resultValid testUnannotatedValidUnmarshal
	err = Unmarshal(inputData, &resultValid, EncodingUTF8, TimezoneUTC, "")
	assert.Nil(t, err)

	// Check that it didnt touch our not-annotated field
	assert.Equal(t, uuid.UUID{}, resultValid.UnannotatedForeignStructre)
	assert.Equal(t, "", resultValid.UnannotatedStringField)
	assert.Equal(t, "12", resultValid.AnnotatedField)
	assert.Equal(t, 0, resultValid.unannotatedUnexportedield)
	assert.Equal(t, "34", resultValid.UnannotatedStruct.JustAField)

	//-------------------------------------------------------------------------

	var resultInvalid testUnannotatedInvalidUnmarshal
	_ = resultInvalid.unexportedAnnotatedField // supress warning
	err = Unmarshal(inputData, &resultInvalid, EncodingUTF8, TimezoneUTC, "")
	assert.EqualError(t, err, "field 'unexportedAnnotatedField' is not exported but annotated")
}

//
//-Addressing------------------------------------------------------------------

type testInvalidAbsolutePositionUnmarshal struct {
	Field1       string `bin:":2"`
	FieldInvalid string `bin:"14:2"` // points out-of-bounds
}

type testInvalidRelativeLengthOOBUnmarshal struct {
	FieldInvalid string `bin:":14"` // out-of-bounds read
}

type testValidAddressingUnmarshal struct {
	Field1 string `bin:":2"`
	Field2 string `bin:"0:2"`
	Field3 string `bin:"7:2"`
	Field4 string `bin:":2"`
}

func TestUnmarshalAddressing(t *testing.T) {

	var inputData = []byte("1234xxx5678")
	var err error

	var resultValid testValidAddressingUnmarshal
	err = Unmarshal(inputData, &resultValid, EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, "12", resultValid.Field1)
	assert.Equal(t, "34", resultValid.Field2)
	assert.Equal(t, "56", resultValid.Field3)
	assert.Equal(t, "78", resultValid.Field4)

	//-------------------------------------------------------------------------

	var resultInvalidAbsolutePos testInvalidAbsolutePositionUnmarshal
	err = Unmarshal(inputData, &resultInvalidAbsolutePos, EncodingUTF8, TimezoneUTC, "\r")
	assert.EqualError(t, err, "absolute position points out-of-bounds on field 'FieldInvalid' `14:2`", err.Error())

	//-------------------------------------------------------------------------

	var resultInvalidRelativeLength testInvalidRelativeLengthOOBUnmarshal
	err = Unmarshal(inputData, &resultInvalidRelativeLength, EncodingUTF8, TimezoneUTC, "\r")
	assert.EqualError(t, err, "error processing field 'FieldInvalid' `:14`: reading out of bounds position 14 in input data of 11 bytes", err.Error())

}

//
//-String----------------------------------------------------------------------

type testStringUnmarshal struct {
	SomeField1 string `bin:":2"`
	SomeField2 string `bin:":4"`
	SomeField3 string `bin:"8:2"` // add a "jump" of two charachters to force to fill up
	SomeField4 string `bin:":4"`
}

func TestUnmarshalString(t *testing.T) {

	var inputData = []byte("123456xxaa  bb")
	var err error

	var result testStringUnmarshal
	err = Unmarshal(inputData, &result, EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, "12", result.SomeField1)
	assert.Equal(t, "3456", result.SomeField2)
	assert.Equal(t, "aa", result.SomeField3)
	assert.Equal(t, "  bb", result.SomeField4)
}

//
//-Integer---------------------------------------------------------------------

type testIntUnmarshal struct {
	Length1                 int `bin:":2"`
	Length2                 int `bin:":4"`
	AbsPos                  int `bin:"8:2"`
	Padded                  int `bin:":4"`
	Negative                int `bin:":2"`
	PaddedNegative          int `bin:":4"`
	PaddedWithSpace         int `bin:":4,padspace"`
	PaddedWithSpaceNegative int `bin:":4,padspace"`
	ForceSign               int `bin:":4,forcesign"`
}

func TestUnmarshalInt(t *testing.T) {

	var inputData = []byte("121234xx120012-2-002   3-  4+005")
	var err error

	var result testIntUnmarshal
	err = Unmarshal(inputData, &result, EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, 12, result.Length1)
	assert.Equal(t, 1234, result.Length2)
	assert.Equal(t, 12, result.AbsPos)
	assert.Equal(t, 12, result.Padded)
	assert.Equal(t, -2, result.Negative)
	assert.Equal(t, -2, result.PaddedNegative)
	assert.Equal(t, 3, result.PaddedWithSpace)
	assert.Equal(t, -4, result.PaddedWithSpaceNegative)
	assert.Equal(t, 5, result.ForceSign)
}

//
//-Float32 / Float64-----------------------------------------------------------

type testFloatUnmarshal struct {
	Length1                 float32 `bin:":1"`
	Length2                 float32 `bin:":2"`
	Length3                 float32 `bin:":3"`
	AbsPos                  float32 `bin:"8:4"`
	Padded                  float32 `bin:":6"`
	Negative                float32 `bin:":5"`
	PaddedNegative          float32 `bin:":6"`
	PaddedWithSpace         float32 `bin:":6,padspace"`
	PaddedWithSpaceNegative float32 `bin:":6,padspace"`
	Precision               float32 `bin:":6,precision:2"` // setting the decimal places shouldn't affect reading. can cause loss of data
	ForceSign               float32 `bin:":4,forcesign"`
	BigNum                  float64 `bin:":10"` // test if scientific notation works
}

func TestUnmarshalFloat(t *testing.T) {

	var inputData = []byte("11.1.2xx1.23001.23-1.23-001.2  1.23-  1.2-1.234+1.2+1.234E+10")
	var err error

	var result testFloatUnmarshal
	err = Unmarshal(inputData, &result, EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, float32(1), result.Length1)
	assert.Equal(t, float32(1), result.Length2)
	assert.Equal(t, float32(1.2), result.Length3)
	assert.Equal(t, float32(1.23), result.AbsPos)
	assert.Equal(t, float32(1.23), result.Padded)
	assert.Equal(t, float32(-1.23), result.Negative)
	assert.Equal(t, float32(-1.2), result.PaddedNegative)
	assert.Equal(t, float32(1.23), result.PaddedWithSpace)
	assert.Equal(t, float32(-1.2), result.PaddedWithSpaceNegative)
	assert.Equal(t, float32(-1.234), result.Precision)
	assert.Equal(t, float32(1.2), result.ForceSign)
	assert.Equal(t, float64(12340000000), result.BigNum)
}

//
//-Nested Struct---------------------------------------------------------------

type testNestedStructUnmarshal struct {
	Anything     int `bin:":2"`
	NestedStruct struct {
		SomethingNested int `bin:":2"`
	}
}

func TestUnmarsalNestedStruct(t *testing.T) {

	var inputData = []byte("0102")
	var err error

	var result testNestedStructUnmarshal
	err = Unmarshal(inputData, &result, EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, 1, result.Anything)
	assert.Equal(t, 2, result.NestedStruct.SomethingNested)
}

//
//-Terminated Array------------------------------------------------------------

type testArrayUnmarshal struct {
	MoreElements []testArrayInnerUnmarshal `bin:"array:terminator"`
	ALotOfInts   []int                     `bin:"array:terminator,:1"`
}

type testArrayInnerUnmarshal struct {
	InnerData1 int `bin:":1"`
	InnerData2 int `bin:":1"`
}

func TestUnmarshalTerminatedArrays(t *testing.T) {

	var inputData = []byte("1234\r0123\r")

	var result testArrayUnmarshal
	var err = Unmarshal(inputData, &result, EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, 2, len(result.MoreElements))
	assert.Equal(t, 1, result.MoreElements[0].InnerData1)
	assert.Equal(t, 2, result.MoreElements[0].InnerData2)
	assert.Equal(t, 3, result.MoreElements[1].InnerData1)
	assert.Equal(t, 4, result.MoreElements[1].InnerData2)
	assert.Equal(t, reflect.DeepEqual(result.ALotOfInts, []int{0, 1, 2, 3}), true)
}

//
//-Fixed Size Array------------------------------------------------------------

type testFixedSizeArrayInnerUnmarshal struct {
	JustAField string `bin:":2"`
	AndAnother int    `bin:":1"`
}

type testFixedSizeArrayUnmarshal struct {
	StructSlice    []testFixedSizeArrayInnerUnmarshal `bin:"array:3"`
	PrimitiveSlice []int                              `bin:"array:4,:1"`
}

func TestUnmarshalArrayWithFixedLength(t *testing.T) {

	var dataExactSize = "AA0BB1CC20123"

	var readExactSize testFixedSizeArrayUnmarshal
	err := Unmarshal([]byte(dataExactSize), &readExactSize, EncodingUTF8, TimezoneUTC, "")
	assert.Nil(t, err)

	var expectedExactSizeInner = []testFixedSizeArrayInnerUnmarshal{
		{JustAField: "AA", AndAnother: 0},
		{JustAField: "BB", AndAnother: 1},
		{JustAField: "CC", AndAnother: 2},
	}
	assert.Equal(t, 3, len(readExactSize.StructSlice))
	assert.Equal(t, reflect.DeepEqual(readExactSize.StructSlice, expectedExactSizeInner), true)
	assert.Equal(t, 4, len(readExactSize.PrimitiveSlice))
	assert.Equal(t, reflect.DeepEqual(readExactSize.PrimitiveSlice, []int{0, 1, 2, 3}), true)

	//-Zero Padded-------------------------------------------------------------
	var dataZeroPadded = "AA0BB1\x00\x00\x0001\x00\x00"

	var readZeroPadded testFixedSizeArrayUnmarshal
	err = Unmarshal([]byte(dataZeroPadded), &readZeroPadded, EncodingUTF8, TimezoneUTC, "")
	assert.Nil(t, err)

	var expectedZeroPaddedInner = []testFixedSizeArrayInnerUnmarshal{
		{JustAField: "AA", AndAnother: 0},
		{JustAField: "BB", AndAnother: 1},
	}
	assert.Equal(t, 2, len(readZeroPadded.StructSlice))
	assert.Equal(t, reflect.DeepEqual(readZeroPadded.StructSlice, expectedZeroPaddedInner), true)
	assert.Equal(t, 2, len(readZeroPadded.PrimitiveSlice))
	assert.Equal(t, reflect.DeepEqual(readZeroPadded.PrimitiveSlice, []int{0, 1}), true)

	//-With Terminator---------------------------------------------------------
	var dataWithTerminator = "AA0BB1CC2\u000D0123"

	var readWithTerminator testFixedSizeArrayUnmarshal
	err = Unmarshal([]byte(dataWithTerminator), &readWithTerminator, EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	var expectedWithTerminatorInner = []testFixedSizeArrayInnerUnmarshal{
		{JustAField: "AA", AndAnother: 0},
		{JustAField: "BB", AndAnother: 1},
		{JustAField: "CC", AndAnother: 2},
	}
	assert.Equal(t, 3, len(readWithTerminator.StructSlice))
	assert.Equal(t, reflect.DeepEqual(readWithTerminator.StructSlice, expectedWithTerminatorInner), true)
	assert.Equal(t, 4, len(readWithTerminator.PrimitiveSlice))
	assert.Equal(t, reflect.DeepEqual(readWithTerminator.PrimitiveSlice, []int{0, 1, 2, 3}), true)
}

//
//-Dynamic Array---------------------------------------------------------------

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

	//-------------------------------------------------------------------------

	var dataWrongType = "A012"
	var resultWrongType testDynamicArrayUnmarshalWrongType

	err = Unmarshal([]byte(dataWrongType), &resultWrongType, EncodingUTF8, TimezoneUTC, "\r")
	assert.EqualError(t, err, "invalid type for array size field 'Nope' - should be int")

	//-------------------------------------------------------------------------

	var dataWrongValue = "-1012"
	var resultWrongValue testDynamicArrayUnmarshalWrongValue

	err = Unmarshal([]byte(dataWrongValue), &resultWrongValue, EncodingUTF8, TimezoneUTC, "\r")
	assert.EqualError(t, err, "invalid size for array size field 'IncorrectSize'")

	//-------------------------------------------------------------------------

	var dataNoField = "A012"
	var resultNoField testDynamicArrayUnmarshalUnknownField

	err = Unmarshal([]byte(dataNoField), &resultNoField, EncodingUTF8, TimezoneUTC, "")
	assert.EqualError(t, err, "unknown field name for array size 'Hacaca'")
}
