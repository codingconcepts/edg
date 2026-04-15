package env

import (
	"fmt"
	"os"

	"github.com/codingconcepts/edg/pkg/convert"
)

func environ(name string) (string, error) {
	val, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("missing environment variable: %q", name)
	}

	return val, nil
}

func (e *Env) global(name string) any {
	return e.request.Globals[name]
}

func (e *Env) sep() convert.RawSQL {
	switch e.driver {
	case "mysql", "mssql":
		return convert.RawSQL("CHAR(31)")
	default:
		return convert.RawSQL("chr(31)")
	}
}
