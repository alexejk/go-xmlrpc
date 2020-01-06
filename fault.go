package xmlrpc

import "fmt"

type Fault struct {
	Code   int
	String string
}

func (f *Fault) Error() string {
	return fmt.Sprintf("%d: %s", f.Code, f.String)
}
