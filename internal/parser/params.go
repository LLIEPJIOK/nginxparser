package parser

import "time"

type Params struct {
	Path        string
	From        *time.Time
	To          *time.Time
	FilterField string
	FilterValue string
}
