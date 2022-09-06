package binfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testGenericMarshal struct {
	someField1 string `bin:",2"`
	someField2 string `bin:",4"`
	someField3 string `bin:"8,2"` // add a "jump" of two charachters to force to fill up
}

func TestMarhalGeneric(t *testing.T) {

	var r testGenericMarshal
	r.someField1 = "12"
	r.someField2 = "3456"
	r.someField3 = "aa"
	data, err := Marshal(r, 'x', EncodingUTF8, TimezoneEuropeBerlin, "\r")

	assert.Nil(t, err)

	assert.Equal(t, "123456xxaa", data)
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
