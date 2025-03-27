package bus

import (
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
