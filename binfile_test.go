package binfile

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

type TestBinaryStructure1 struct {
	RecordType          string `bin:",2"`
	UnitNo              int    `bin:",2"`
	RackNumber          int    `bin:",4"`
	CupPosition         int    `bin:",2"`
	SampleType          string `bin:",1,trim"`
	SampleNo            string `bin:",4"`
	SampleId            string `bin:",11"`
	Dummy               string `bin:",4,trim"`
	BlockIdentification string `bin:",1,trim"`
	TestResults         []struct {
		TestCode   string `bin:",2"`
		TestResult string `bin:",9,trim"`
		Flags      string `bin:",2"`
	}
}

// POC Unmarshal binary protocol
func TestUnmarshalBinary1(t *testing.T) {
	data := "D 03116506 044760722905768    E61     6.40  62      935  "
	var r TestBinaryStructure1
	err := Unmarshal([]byte(data), r, CR)
	assert.Equal(t, nil, err)
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
	assert.Equal(t, "6.4", r.TestResults[0].TestResult)
	assert.Equal(t, "", r.TestResults[0].Flags)
	assert.Equal(t, "62", r.TestResults[0].TestCode)
	assert.Equal(t, "935", r.TestResults[0].TestResult)
	assert.Equal(t, "", r.TestResults[0].Flags)
}

type TestBinaryStructure2 struct {
	DMessage []TestBinaryStructure1
}

func TestUnmarshalBinary2(t *testing.T) {
	data := "D 03116506 044760722905768    E61     6.40  62      935  \u000DD 03116507 044860722905758    E61     6.86  62      883  "

	var r TestBinaryStructure2
	err := Unmarshal([]byte(data), r, CR)
	assert.Equal(t, nil, err)
}
