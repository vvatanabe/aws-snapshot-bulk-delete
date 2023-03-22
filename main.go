package main

import (
	"fmt"
	"os"
)

const (
	exitCodeOK    int = 0
	exitCodeError int = iota
)

func main() {
	app := app()
	err := app.Run(os.Args)
	code := exitCodeOK
	if err != nil {
		if err.Error() != "" {
			_, _ = fmt.Fprintf(app.ErrWriter, "%v\n", err)
		}
		code = exitCodeError
	}
	os.Exit(code)
}
