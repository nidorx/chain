package socket

import "github.com/syntax-framework/chain"

type Transport interface {
	Configure(h *Handler, r *chain.Router, endpoint string)
}
