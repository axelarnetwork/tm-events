package types

import "encoding/json"

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `jsons:"data"`
}

type Result struct {
	Query  string          `json:query`
	Data   json.RawMessage `json:"data"`
	Events json.RawMessage `json:"cmd"`
}

type Request struct {
	Version string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// RPC Message
type Message struct {
	Version string  `json:"jsonrpc"`
	ID      string  `json:"id"`
	Error   *Error  `json:"error"`
	Result  *Result `json:"result"`
}
