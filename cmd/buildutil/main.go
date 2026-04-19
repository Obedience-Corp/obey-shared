// Command buildutil runs the shared build/test dashboard for obey-shared.
//
// obey-shared is a library (no binary), so it uses buildutil in library mode:
// only test/lint/coverage/clean/all are supported.
package main

import (
	"os"

	"github.com/Obedience-Corp/obey-shared/buildutil"
)

func main() {
	buildutil.Run(os.Args[1:], buildutil.BuildConfig{
		SectionName: "obey-shared",
	})
}
