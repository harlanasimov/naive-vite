package common

import "fmt"

type StrError struct {
	E string
}

func (e StrError) Error() string {
	return fmt.Sprintf("%s", e.E)
}
