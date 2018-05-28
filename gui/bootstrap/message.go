package bootstrap

import (
	"encoding/json"

	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilog"
)

// MessageOut represents a message going out
type MessageOut struct {
	Name    string      `json:"name"`
	Payload interface{} `json:"payload"`
}

// MessageIn represents a message going in
type MessageIn struct {
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload"`
}

// handleMessages handles messages
func handleMessages(w *astilectron.Window, messageHandler MessageHandler) astilectron.ListenerMessage {
	return func(e *astilectron.EventMessage) (v interface{}) {
		// Unmarshal message
		var m MessageIn
		var err error
		if err = e.Unmarshal(&m); err != nil {
			astilog.Errorf("Unmarshaling message %+v failed", *e)
			return
		}

		// Handle message
		messageHandler(w, m)
		return
	}
}
