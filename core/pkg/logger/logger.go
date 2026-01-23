package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

// Sets a multiwriter logger: one to `/tmp/<filepath>`
// and another to the stderr.
//
// Panics if an error occurs.
func CreateMultiWriterLogger(filepath string) {
	f, err := os.OpenFile(
		fmt.Sprintf("/tmp/%s.log", filepath),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	if err != nil {
		log.Fatal(err)
	}

	mw := io.MultiWriter(os.Stderr, f)
	log.SetOutput(mw)
}
