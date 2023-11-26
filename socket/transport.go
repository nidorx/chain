package socket

import "github.com/nidorx/chain"

type Transport interface {
	Configure(h *Handler, r *chain.Router, endpoint string)
}
