package logger

import (
	"fmt"
	"os"
)

var logFile *os.File

func init() {
	logFile, _ = os.Create("/tmp/sun.log")
}

func Log(line string, params ...interface{}) {
	if logFile != nil {
		_, _ = fmt.Fprintf(logFile, line+"\n", params...)
	}
}
