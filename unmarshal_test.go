package binfile

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

type TestBinaryStructure1 struct {
	RecordType          string `bin:",2"`      // 0
	UnitNo              int    `bin:",2"`      // 2
	RackNumber          int    `bin:",4"`      // 4
	CupPosition         int    `bin:",2"`      // 8
	SampleType          string `bin:",1,trim"` // 10
	SampleNo            string `bin:",4"`      // 11
	SampleId            string `bin:",11"`     // 15
	Dummy               string `bin:",4,trim"` // 26
	BlockIdentification string `bin:",1,trim"` // 30

	TestResults []struct {
		TestCode   string `bin:",2"`            // 1. 30  2. 43
		TestResult string `bin:",9,trim"`       // 1. 32  2. 45
		Flags      string `bin:",2,trim"`       // 1. 41  2. 54
		Terminator string `bin:",1,terminator"` // 1.     2. 56
	} `bin:",,array"`

	DoNotSkipThisOneTest string `bin:",1"` // 44 (with 1 run)
}

// POC Unmarshal binary protocol generic simple example
/*func TestUnmarshalBinary1(t *testing.T) {
	data := "D 03116506 044760722905768    E61     6.40  62      935  "
	var r TestBinaryStructure1
	err := Unmarshal([]byte(data), &r, CR, "\n")
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
	assert.Equal(t, "6.40", r.TestResults[0].TestResult)
	assert.Equal(t, "", r.TestResults[0].Flags)

	assert.Equal(t, "62", r.TestResults[1].TestCode)
	assert.Equal(t, "935", r.TestResults[1].TestResult)
	assert.Equal(t, "", r.TestResults[1].Flags)

}
*/
type TestBinaryStructure2 struct {
	DMessage []TestBinaryStructure1
}

func TestUnmarshalBinary2(t *testing.T) {
	data := "D 03116506 044760722905768    E61     6.40  62      935  \u000DD 03116507 044860722905758    E61     6.86  62      883  "

	var r TestBinaryStructure2
	err := Unmarshal([]byte(data), &r, CR, "\r")

	assert.Equal(t, nil, err)

	assert.Equal(t, 2, len(r.DMessage))
}
