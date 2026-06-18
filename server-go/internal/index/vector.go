package index

import (
	"strconv"
	"strings"
)

// encodeVector renders a float slice as a pgvector text literal: "[1,2,3]".
// It is inserted with an explicit ::vector cast so the dimension is flexible.
func encodeVector(vec []float32) string {
	if len(vec) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}
