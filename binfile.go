package binfile

type LineBreak int

const (
	CR   LineBreak = 0x0D
	LF   LineBreak = 0x0A
	CRLF LineBreak = 0x0D0A
)

func unmarshal(d []byte, v interface{}, linebreak LineBreak) {

}

func marshal(v interface{}) []byte {

	return []byte{}
}
