package rpc

import (
	"io"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
)

// New creates a jrpc2 Server with NDJSON framing on r/w and push notifications enabled.
func New(assigner jrpc2.Assigner, r io.Reader, w io.WriteCloser) *jrpc2.Server {
	srv := jrpc2.NewServer(assigner, &jrpc2.ServerOptions{AllowPush: true})
	ch := channel.Line(r, w)
	srv.Start(ch)
	return srv
}
