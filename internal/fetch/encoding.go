package fetch

import (
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

// encodingMap maps encoding names to encoding.Encoding instances. It is
// sourced from the golang.org/x/text/encoding/charmap package (see
// charmap.All).
var encodingMap = map[string]encoding.Encoding{
	"CodePage037":       charmap.CodePage037,
	"CodePage437":       charmap.CodePage437,
	"CodePage850":       charmap.CodePage850,
	"CodePage852":       charmap.CodePage852,
	"CodePage855":       charmap.CodePage855,
	"CodePage858":       charmap.CodePage858,
	"CodePage860":       charmap.CodePage860,
	"CodePage862":       charmap.CodePage862,
	"CodePage863":       charmap.CodePage863,
	"CodePage865":       charmap.CodePage865,
	"CodePage866":       charmap.CodePage866,
	"CodePage1047":      charmap.CodePage1047,
	"CodePage1140":      charmap.CodePage1140,
	"ISO8859_1":         charmap.ISO8859_1,
	"ISO8859_2":         charmap.ISO8859_2,
	"ISO8859_3":         charmap.ISO8859_3,
	"ISO8859_4":         charmap.ISO8859_4,
	"ISO8859_5":         charmap.ISO8859_5,
	"ISO8859_6":         charmap.ISO8859_6,
	"ISO8859_6E":        charmap.ISO8859_6E,
	"ISO8859_6I":        charmap.ISO8859_6I,
	"ISO8859_7":         charmap.ISO8859_7,
	"ISO8859_8":         charmap.ISO8859_8,
	"ISO8859_8E":        charmap.ISO8859_8E,
	"ISO8859_8I":        charmap.ISO8859_8I,
	"ISO8859_9":         charmap.ISO8859_9,
	"ISO8859_10":        charmap.ISO8859_10,
	"ISO8859_13":        charmap.ISO8859_13,
	"ISO8859_14":        charmap.ISO8859_14,
	"ISO8859_15":        charmap.ISO8859_15,
	"ISO8859_16":        charmap.ISO8859_16,
	"KOI8R":             charmap.KOI8R,
	"KOI8U":             charmap.KOI8U,
	"Macintosh":         charmap.Macintosh,
	"MacintoshCyrillic": charmap.MacintoshCyrillic,
	"Windows874":        charmap.Windows874,
	"Windows1250":       charmap.Windows1250,
	"Windows1251":       charmap.Windows1251,
	"Windows1252":       charmap.Windows1252,
	"Windows1253":       charmap.Windows1253,
	"Windows1254":       charmap.Windows1254,
	"Windows1255":       charmap.Windows1255,
	"Windows1256":       charmap.Windows1256,
	"Windows1257":       charmap.Windows1257,
	"Windows1258":       charmap.Windows1258,
	"XUserDefined":      charmap.XUserDefined,
}
