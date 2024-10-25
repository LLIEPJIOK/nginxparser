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

type ErrBadStatus struct {
	msg string
}

func NewErrBadStatus(msg string) error {
	return ErrBadStatus{
		msg: msg,
	}
}

func (e ErrBadStatus) Error() string {
	return e.msg
}

type ErrNoFiles struct {
	msg string
}

func NewErrNoFiles(msg string) error {
	return ErrNoFiles{
		msg: msg,
	}
}

func (e ErrNoFiles) Error() string {
	return e.msg
}
