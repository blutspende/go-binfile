package binfile

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

//
//-General example structure---------------------------------------------------

type testGeneralStructureMarshal struct {
	RecordType          string `bin:":2"`      // 0
	UnitNo              int    `bin:":2"`      // 2
	RackNumber          int    `bin:":4"`      // 4
	CupPosition         int    `bin:":2"`      // 8
	SampleType          string `bin:":1,trim"` // 10
	SampleNo            string `bin:":4"`      // 11
	SampleId            string `bin:":11"`     // 15
	Dummy               string `bin:":4,trim"` // 26
	BlockIdentification string `bin:":1,trim"` // 30

	TestResults []testTestResultStructureMarshal `bin:"array:terminator"`
}

type testTestResultStructureMarshal struct {
	TestCode   string `bin:":2"`      // 1. 30  2. 43
	TestResult string `bin:":9,trim"` // 1. 32  2. 45
	Flags      string `bin:":2,trim"` // 1. 41  2. 54
}

func TestMarshalGeneralStructure(t *testing.T) {

	var inputData = testGeneralStructureMarshal{
		RecordType:          "D ",
		UnitNo:              3,
		RackNumber:          1165,
		CupPosition:         6,
		SampleType:          "",
		SampleNo:            "0447",
		SampleId:            "60722905768",
		Dummy:               "",
		BlockIdentification: "E",
		TestResults: []testTestResultStructureMarshal{
			{
				TestCode:   "61",
				TestResult: "6.40",
				Flags:      "",
			},
			{
				TestCode:   "62",
				TestResult: "935",
				Flags:      "",
			},
		},
	}

	result, err := Marshal(inputData, ' ', EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, []byte("D 03116506 044760722905768    E61     6.40  62      935  \r"), result)
}

//
//-Multiple example structure in array-----------------------------------------

type testMultipleRecordsMarshal struct {
	DMessage []testGeneralStructureMarshal `bin:"array:terminator"`
}

func TestMarshalMultipleRecords(t *testing.T) {

	var inputData = testMultipleRecordsMarshal{
		DMessage: []testGeneralStructureMarshal{
			{
				RecordType:          "D ",
				UnitNo:              3,
				RackNumber:          1165,
				CupPosition:         6,
				SampleType:          "",
				SampleNo:            "0447",
				SampleId:            "60722905768",
				Dummy:               "",
				BlockIdentification: "E",
				TestResults: []testTestResultStructureMarshal{
					{
						TestCode:   "61",
						TestResult: "6.40",
						Flags:      "",
					},
					{
						TestCode:   "62",
						TestResult: "935",
						Flags:      "",
					},
				},
			},
			{
				RecordType:          "D ",
				UnitNo:              3,
				RackNumber:          1165,
				CupPosition:         7,
				SampleType:          "",
				SampleNo:            "0448",
				SampleId:            "60722905758",
				Dummy:               "",
				BlockIdentification: "E",
				TestResults: []testTestResultStructureMarshal{
					{
						TestCode:   "61",
						TestResult: "6.86",
						Flags:      "",
					},
					{
						TestCode:   "62",
						TestResult: "883",
						Flags:      "",
					},
				},
			},
		},
	}

	result, err := Marshal(inputData, ' ', EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, []byte("D 03116506 044760722905768    E61     6.40  62      935  \u000DD 03116507 044860722905758    E61     6.86  62      883  \r\r"), result)
}

//
//-Top-level Array-------------------------------------------------------------

type testTopLevelArrayInnerMarshal struct {
	SomeField1 string `bin:":2"`
	SomeField2 string `bin:":4"`
	SomeField3 string `bin:"8:2"`
}

func TestMarshalTopLevelArray(t *testing.T) {

	var inputData = []testTopLevelArrayInnerMarshal{
		{SomeField1: "AB", SomeField2: "CDEF"},
		{SomeField1: "12", SomeField2: "3456"},
	}

	result, err := Marshal(inputData, 'x', EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, []byte("ABCDEFxx  123456xx  \r"), result)
}

//
//-Annotation presence---------------------------------------------------------

type testUnannotatedValidMarshal struct {
	UnannotatedForeignStructre uuid.UUID
	UnannotatedStringField     string
	AnnotatedField             string `bin:":2"`
	unannotatedUnexportedield  int
	UnannotatedStruct          testUnannotatedValidInnerMarshal
}

type testUnannotatedValidInnerMarshal struct {
	JustAField string `bin:":2"`
}

type testUnannotatedInvalidMarshal struct {
	unexportedAnnotatedField string `bin:":2"`
}

// TEST: bug: unannotated fields in struct did cause an erroresult.
// Unannotated fields should just be skipped
func TestMarshalUnanotatedFieldsAreSkipped(t *testing.T) {

	var result []byte
	var err error

	var inputData testUnannotatedValidMarshal
	inputData.AnnotatedField = "12"
	inputData.UnannotatedStruct.JustAField = "34"
	_ = inputData.unannotatedUnexportedield // suppress warning

	result, err = Marshal(inputData, ' ', EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, []byte("1234"), result)

	//-------------------------------------------------------------------------

	var inputInvalid testUnannotatedInvalidMarshal
	_ = inputInvalid.unexportedAnnotatedField
	result, err = Marshal(inputInvalid, ' ', EncodingUTF8, TimezoneUTC, "\r")
	_ = result
	assert.EqualError(t, err, "field 'unexportedAnnotatedField' is not exported but annotated")
}

//
//-Addressing------------------------------------------------------------------

type testInvalidAbsolutePositionMarshal struct {
	Field1       string `bin:":2"`
	FieldInvalid string `bin:"1:2"` // points backwards
}

type testValidAddressingMarshal struct {
	SomeField1 string `bin:":2"`
	SomeField2 string `bin:":4"`
	SomeField3 string `bin:"8:2"` // add a "jump" of two charachters to force to fill up
	IntField   int    `bin:":3"`
	//IntFieldPadding int     `bin:":3,padzero"` TODO
	//DecimalField    float32 `bin:":5,decimal(2)"` TODO
}

func TestMarshalAddressing(t *testing.T) {

	var inputData = testValidAddressingMarshal{
		SomeField1: "12",
		SomeField2: "3456",
		SomeField3: "aa",
		IntField:   7,
	}

	resultValid, err := Marshal(inputData, 'x', EncodingUTF8, TimezoneEuropeBerlin, "\r")
	assert.Nil(t, err)

	assert.Equal(t, []byte("123456xxaa007"), resultValid)

	//-------------------------------------------------------------------------

	var inputDataInvalid testInvalidAbsolutePositionMarshal
	resultInvalid, err := Marshal(inputDataInvalid, 'x', EncodingUTF8, TimezoneEuropeBerlin, "\r")
	_ = resultInvalid
	assert.EqualError(t, err, "absoulute position points backwards 'FieldInvalid' `1:2`")
}

//
//-String----------------------------------------------------------------------

type testStringMarshal struct {
	SomeField1 string `bin:":2"`
	SomeField2 string `bin:":4"`
	SomeField3 string `bin:"8:2"` // add a "jump" of two charachters to force to fill up
	SomeField4 string `bin:":4"`
}

func TestMarshalString(t *testing.T) {

	var inputData = testStringMarshal{
		SomeField1: "12",
		SomeField2: "3456",
		SomeField3: "aa",
		SomeField4: "bb",
	}

	result, err := Marshal(inputData, 'x', EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, []byte("123456xxaa  bb"), result)
}

//
//-Integer---------------------------------------------------------------------

type testIntMarshal struct {
	Length1        int `bin:":2"`
	Length2        int `bin:":4"`
	AbsPos         int `bin:"8:2"`
	Padded         int `bin:":4"`
	Negative       int `bin:":2"`
	PaddedNegative int `bin:":4"`
}

func TestMarshalInt(t *testing.T) {

	var inputData = testIntMarshal{
		Length1:        12,
		Length2:        1234,
		AbsPos:         12,
		Padded:         12,
		Negative:       -2,
		PaddedNegative: -2,
	}

	result, err := Marshal(inputData, 'x', EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, []byte("121234xx120012-2-002"), result)
}

//
//-Float32 / Float64-----------------------------------------------------------

type testFloatMarshal struct {
	Length1        float32 `bin:":1"`
	Length2        float32 `bin:":2"`
	Length3        float32 `bin:":3"`
	AbsPos         float32 `bin:"8:4"`
	Padded         float32 `bin:":6"`
	Negative       float32 `bin:":5"`
	PaddedNegative float32 `bin:":6"`
	//BigNum         float64 `bin:":4"` // TODO
}

func TestMarshalFloat(t *testing.T) {

	var inputData = testFloatMarshal{
		Length1:        1,  // TODO: is '1' can be a float?
		Length2:        1., // TODO: is '1.' can be a float?
		Length3:        1.2,
		AbsPos:         1.23,
		Padded:         1.23,
		Negative:       -1.23,
		PaddedNegative: -1.2,
		//BigNum:         math.MaxFloat32 + 1, // TODO: scientific notation?
	}

	var result, err = Marshal(inputData, 'x', EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, []byte("11.1.2xx1.231.2300-1.23-1.200"), result)
}

//
//-Nested Struct---------------------------------------------------------------

type testNestedStructMarshal struct {
	Anything     int `bin:":2"`
	NestedStruct struct {
		SomethingNested int `bin:":2"`
	}
}

func TestMarsalNestedStruct(t *testing.T) {

	var inputData testNestedStructMarshal
	inputData.Anything = 1
	inputData.NestedStruct.SomethingNested = 2

	var data, err = Marshal(inputData, 'x', EncodingUTF8, TimezoneEuropeBerlin, "\r")
	assert.Nil(t, err)

	assert.Equal(t, []byte("0102"), data)
}

//
//-Terminated Array------------------------------------------------------------

type testArrayMarshal struct {
	MoreElements []testArrayInnerMarshal `bin:"array:terminator"`
	ALotOfInts   []int                   `bin:"array:terminator,:1"`
}
type testArrayInnerMarshal struct {
	InnerData1 int `bin:":1"`
	InnerData2 int `bin:":1"`
}

func TestMarshalTerminatedArrays(t *testing.T) {

	var inputData = testArrayMarshal{
		MoreElements: []testArrayInnerMarshal{
			{
				InnerData1: 1,
				InnerData2: 2,
			},
			{
				InnerData1: 3,
				InnerData2: 4,
			},
		},
		ALotOfInts: []int{0, 1, 2, 3},
	}

	var data, err = Marshal(inputData, 'x', EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, []byte("1234\r0123\r"), data)
}

//
//-Fixed Size Array------------------------------------------------------------

type testFixedSizeArrayMarshal struct {
	SomeArray  []testFixedSizeArrayInnerMarshal `bin:"array:5"`
	LotsOfInts []int                            `bin:"array:3,:1"`
}

type testFixedSizeArrayInnerMarshal struct {
	Field1 string `bin:":2"`
}

func TestMarshalArrayWithFixedLength(t *testing.T) {

	inputDataLessElements := testFixedSizeArrayMarshal{
		SomeArray: []testFixedSizeArrayInnerMarshal{
			{Field1: "N1"},
			{Field1: "N2"},
			{Field1: "N3"},
			{Field1: "N4"},
		},
		LotsOfInts: []int{1},
	}

	inputDataMoreElements := testFixedSizeArrayMarshal{
		SomeArray: []testFixedSizeArrayInnerMarshal{
			{Field1: "N1"},
			{Field1: "N2"},
			{Field1: "N3"},
			{Field1: "N4"},
			{Field1: "N5"},
			{Field1: "N6"},
		},
		LotsOfInts: []int{0, 1, 2, 3, 4},
	}

	// padding zeros for a required length of 5
	result, err := Marshal(inputDataLessElements, 'x', EncodingUTF8, TimezoneUTC, "")
	assert.Nil(t, err)

	assert.Equal(t, []byte("N1N2N3N4\x00\x001\x00\x00"), result)

	// 6 Elements in a 5 elment array expected to b cut to 5 and then terminated
	result, err = Marshal(inputDataMoreElements, 'x', EncodingUTF8, TimezoneUTC, "_")
	assert.Nil(t, err)

	assert.Equal(t, []byte("N1N2N3N4N5_012_"), result)
}

//
//-Dynamic Array---------------------------------------------------------------

type testDynamicArrayMarshalUnknownField struct {
	NotHere   string `bin:":1"`
	TheArray1 []int  `bin:"array:Hacaca,:1"`
}

type testDynamicArrayMarshalWrongType struct {
	Nope      string `bin:":1"`
	TheArray1 []int  `bin:"array:Nope,:1"`
}

type testDynamicArrayMarshalWrongValue struct {
	IncorrectSize int   `bin:":2"`
	TheArray1     []int `bin:"array:IncorrectSize,:1"`
}

type testDynamicArrayMarshal struct {
	Nope          string `bin:":1"`
	CorrectSize   int    `bin:":1"`
	IncorrectSize int    `bin:":2"`
	TheArray1     []int  `bin:"array:CorrectSize,:1"`
}

func TestMasrshalDynamicArray(t *testing.T) {

	var result []byte
	var err error

	var inputData = testDynamicArrayMarshal{
		Nope:          "A",
		CorrectSize:   3,
		IncorrectSize: -1,
		TheArray1:     []int{0, 1, 2},
	}

	result, err = Marshal(inputData, 'x', EncodingUTF8, TimezoneUTC, "\r")
	assert.Nil(t, err)

	assert.Equal(t, []byte("A3-1012"), result)

	//-------------------------------------------------------------------------

	var inputDataWrongType = testDynamicArrayMarshalWrongType{
		Nope:      "A",
		TheArray1: []int{0, 1, 2},
	}

	result, err = Marshal(inputDataWrongType, 'x', EncodingUTF8, TimezoneUTC, "\r")
	_ = result
	assert.EqualError(t, err, "invalid type for array size field 'Nope' - should be int")

	//-------------------------------------------------------------------------

	var inputDataWrongValue = testDynamicArrayMarshalWrongValue{
		IncorrectSize: -1,
		TheArray1:     []int{0, 1, 2},
	}

	result, err = Marshal(inputDataWrongValue, 'x', EncodingUTF8, TimezoneUTC, "\r")
	_ = result
	assert.EqualError(t, err, "invalid size for array size field 'IncorrectSize'")

	//-------------------------------------------------------------------------

	var inputDataNoField = testDynamicArrayMarshalUnknownField{
		NotHere:   "A",
		TheArray1: []int{0, 1, 2},
	}

	result, err = Marshal(inputDataNoField, 'x', EncodingUTF8, TimezoneUTC, "\r")
	_ = result
	assert.EqualError(t, err, "unknown field name for array size 'Hacaca'")
}
