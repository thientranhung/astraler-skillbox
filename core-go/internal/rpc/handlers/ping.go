package handlers

import "time"

type PingResult struct {
	Pong bool   `json:"pong"`
	TS   string `json:"ts"`
}

func Ping() PingResult {
	return PingResult{
		Pong: true,
		TS:   time.Now().UTC().Format(time.RFC3339),
	}
}
