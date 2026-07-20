// Package sei exposes Thingino's embedded SEI OSD metadata to WebSocket
// clients, so the browser player can render it as a live overlay without
// any server-side transcoding. See pkg/thingino for the wire format and
// extraction logic.
package sei

import (
	"encoding/json"
	"errors"

	"github.com/AlexxIT/go2rtc/internal/api"
	"github.com/AlexxIT/go2rtc/internal/api/ws"
	"github.com/AlexxIT/go2rtc/internal/app"
	"github.com/AlexxIT/go2rtc/internal/streams"
	"github.com/AlexxIT/go2rtc/pkg/thingino"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func Init() {
	log = app.GetLogger("sei")

	ws.HandleFunc("sei", handlerWSSEI)
}

func handlerWSSEI(tr *ws.Transport, _ *ws.Message) error {
	stream, _ := streams.GetOrPatch(tr.Request.URL.Query())
	if stream == nil {
		return errors.New(api.StreamNotFound)
	}

	log.Trace().Msg("[sei] new WS/SEI consumer")

	cons := thingino.NewConsumer(func(payload json.RawMessage) {
		tr.Write(&ws.Message{Type: "sei", Value: payload})
	})

	if err := stream.AddConsumer(cons); err != nil {
		log.Debug().Err(err).Msg("[sei] add consumer")
		return err
	}

	tr.OnClose(func() {
		stream.RemoveConsumer(cons)
	})

	return nil
}
