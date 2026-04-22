package env

import (
	"fmt"
	"log"
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

func environNil(name string) any {
	val, ok := os.LookupEnv(name)
	if !ok {
		return nil
	}
	return val
}

func fail(msg string) (any, error) {
	return nil, fmt.Errorf("fail: %s", msg)
}

func fatal(msg string) any {
	log.Fatalf("fatal: %s", msg)
	return nil
}

func (e *Env) global(name string) any {
	return e.request.Globals[name]
}

func (e *Env) sep() convert.RawSQL {
	switch e.driver {
	case "mysql", "mssql":
		return convert.RawSQL("CHAR(31)")
	case "spanner":
		return convert.RawSQL("CODE_POINTS_TO_STRING([31])")
	case "oracle":
		return convert.RawSQL("codepoints-to-string(31)")
	default:
		return convert.RawSQL("chr(31)")
	}
}
