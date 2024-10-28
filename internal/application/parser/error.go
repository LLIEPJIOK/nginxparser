package parser

type ErrEmptyLogPath struct{}

func (e ErrEmptyLogPath) Error() string {
	return "log path is empty"
}

type ErrFlag struct {
	msg string
}

func NewErrFlag(msg string) ErrFlag {
	return ErrFlag{
		msg: msg,
	}
}

func (e ErrFlag) Error() string {
	return e.msg
}
