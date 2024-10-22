package parser

type ErrRegexp struct {
	msg string
}

func NewErrRegexp(msg string) error {
	return ErrRegexp{
		msg: msg,
	}
}

func (e ErrRegexp) Error() string {
	return e.msg
}
