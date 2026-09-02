package bus

import (
	"encoding/binary"
	"strings"
	"unicode"
)

func unique(subj string) (durable string) {
	var res strings.Builder
	for _, x := range subj {
		switch {
		case unicode.IsLetter(x), unicode.IsNumber(x):
			res.WriteRune(x)
		}
	}

	return res.String()
}

// int
func IntBytes(i int) (r []byte) {
	r = make([]byte, 8)
	binary.LittleEndian.PutUint64(r, uint64(i))
	return
}

func BytesInt(b []byte) (i int) {
	if b == nil {
		return 0
	}
	return int(binary.LittleEndian.Uint64(b))
}
