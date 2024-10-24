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

type ErrInvalidURL struct {
	msg string
}

func NewErrInvalidURL(msg string) error {
	return ErrInvalidURL{
		msg: msg,
	}
}

func (e ErrInvalidURL) Error() string {
	return e.msg
}
