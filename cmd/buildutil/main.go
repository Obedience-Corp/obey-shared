// Command buildutil runs the live test/lint dashboard for obey-shared.
//
// The dashboard implementation lives in github.com/Obedience-Corp/build-util.
// obey-shared is a library (no binary), so this wrapper uses library mode:
// only test/lint/coverage/clean/all are supported.
package main

import (
	"os"

	buildutil "github.com/Obedience-Corp/build-util"
)

func main() {
	buildutil.Run(os.Args[1:], buildutil.BuildConfig{
		SectionName: "obey-shared",
	})
}
