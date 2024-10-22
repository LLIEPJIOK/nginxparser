package parser

type line struct {
	text   string
	number int
}

func newLine(text string, number int) line {
	return line{
		text:   text,
		number: number,
	}
}
