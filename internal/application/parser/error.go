package parser

type ErrEmptyLogPath struct{}

func (e ErrEmptyLogPath) Error() string {
	return "log path is empty"
}
