package binfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalBinary1(t *testing.T) {

	var r TestBinaryStructure1
	r.RecordType = "D "
	r.UnitNo = 3
	r.RackNumber = 1165
	r.CupPosition = 6
	r.SampleType = ""
	r.SampleNo = "0447"
	r.SampleId = "60722905768"
	r.Dummy = ""
	r.BlockIdentification = "E"
	r.TestResults = []TestResultsStructure{
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
	}

	data, err := Marshal(r, ' ', EncodingUTF8, TimezoneEuropeBerlin, "\r")

	assert.Nil(t, err)

	assert.Equal(t, "D 03116506 044760722905768    E61     6.40  62      935  \r", string(data))

}

func TestMarshalBinary2(t *testing.T) {

	var r TestBinaryStructure2
	r.DMessage = []TestBinaryStructure1{
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
			TestResults: []TestResultsStructure{
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
			TestResults: []TestResultsStructure{
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
	}

	data, err := Marshal(r, ' ', EncodingUTF8, TimezoneEuropeBerlin, "\r")

	assert.Nil(t, err)

	assert.Equal(t, "D 03116506 044760722905768    E61     6.40  62      935  \u000DD 03116507 044860722905758    E61     6.86  62      883  \r\r", string(data))

}

type testStringMarshal struct {
	SomeField1 string `bin:":2"`
	SomeField2 string `bin:":4"`
	SomeField3 string `bin:"8:2"` // add a "jump" of two charachters to force to fill up
	SomeField4 string `bin:":4"`
}

func TestMarshalString(t *testing.T) {

	var r testStringMarshal
	r.SomeField1 = "12"
	r.SomeField2 = "3456"
	r.SomeField3 = "aa"
	r.SomeField4 = "bb"
	data, err := Marshal(r, 'x', EncodingUTF8, TimezoneEuropeBerlin, "\r")

	assert.Nil(t, err)

	assert.Equal(t, "123456xxaa  bb", string(data))
}

type testIntMarshal struct {
	Length1        int `bin:":2"`
	Length2        int `bin:":4"`
	AbsPos         int `bin:"8:2"`
	Padded         int `bin:":4"`
	Negative       int `bin:":2"`
	PaddedNegative int `bin:":4"`
}

func TestMarshalInt(t *testing.T) {

	var r testIntMarshal
	r.Length1 = 12
	r.Length2 = 1234
	r.AbsPos = 12
	r.Padded = 12
	r.Negative = -2
	r.PaddedNegative = -2

	var data, err = Marshal(r, 'x', EncodingUTF8, TimezoneEuropeBerlin, "\r")

	assert.Nil(t, err)

	assert.Equal(t, "121234xx120012-2-002", string(data))
}

type testFloatMarshal struct {
	Length1        float32 `bin:":1"`
	Length2        float32 `bin:":2"`
	Length3        float32 `bin:":3"`
	AbsPos         float32 `bin:"8:4"`
	Padded         float32 `bin:":6"`
	Negative       float32 `bin:":5"`
	PaddedNegative float32 `bin:":6"`
	BigNum         float64 `bin:":4"`
}

func TestMarshalFloat(t *testing.T) {

	var r testFloatMarshal
	r.Length1 = 1  // TODO: is '1' can be a float?
	r.Length2 = 1. // TODO: is '1.' can be a float?
	r.Length3 = 1.2
	r.AbsPos = 1.23
	r.Padded = 1.23
	r.Negative = -1.23
	r.PaddedNegative = -1.2
	// TODO: scientific notation?
	//r.BigNum = math.MaxFloat32 + 1

	var data, err = Marshal(r, 'x', EncodingUTF8, TimezoneEuropeBerlin, "\r")

	assert.Nil(t, err)

	assert.Equal(t, "11.1.2xx1.231.2300-1.23-1.2000.00", string(data))
}

type testNestedMarshal struct {
	Anything     int `bin:"2"`
	NestedStruct struct {
		SomethingNested int `bin:"2"`
	}
}

func TestMarsalNested(t *testing.T) {

	var r testNestedMarshal
	r.Anything = 1
	r.NestedStruct.SomethingNested = 2

	var data, err = Marshal(r, 'x', EncodingUTF8, TimezoneEuropeBerlin, "\r")

	assert.Nil(t, err)

	assert.Equal(t, "0102", string(data))
}

type testSliceMarshal struct {
	MoreElements []testSliceInnerMarshal
}
type testSliceInnerMarshal struct {
	InnerData1 int `bin:":1"`
	InnerData2 int `bin:":1"`
}

func TestMarshalSlices(t *testing.T) {

	var r testSliceMarshal
	r.MoreElements = []testSliceInnerMarshal{
		{
			InnerData1: 1,
			InnerData2: 2,
		},
		{
			InnerData1: 3,
			InnerData2: 4,
		},
	}

	var data, err = Marshal(r, 'x', EncodingUTF8, TimezoneEuropeBerlin, "\r")

	assert.Nil(t, err)

	assert.Equal(t, "1234\r", string(data))
}

/*
func TestMarshalArray(t *testing.T) {
	r := make([]testGenericMarshal, 0)
	r = append(r, testGenericMarshal{someField1: "AB", someField2: "CDEF"})
	r = append(r, testGenericMarshal{someField1: "12", someField2: "3456"})

	data, err := Marshal(r, 'x', EncodingUTF8, TimezoneEuropeBerlin, "\r")

	assert.Nil(t, err)
	assert.Equal(t, "ABCDEFxx  \r123456xx  \r", data)
}*/
