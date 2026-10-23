//go:build container_fs

package updatecheck

import (
	"net/http"
	"time"
)

func at(t time.Time) func() time.Time { return func() time.Time { return t } }

// tty is a human terminal with an empty environment, so host CI variables
// cannot leak into a test.
var tty = SuppressInput{StderrIsTTY: true, LookupEnv: func(string) string { return "" }}

func (rs *releaseServer) serve(h http.HandlerFunc) { rs.handler.Store(h) }
