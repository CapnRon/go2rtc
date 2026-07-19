package thingino

import (
	"encoding/json"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/h264"
	"github.com/pion/rtp"
)

// Consumer taps a stream's H264 video track and calls onSEI whenever a new
// (changed) Thingino SEI payload is seen. It's a sniffer, not a real media
// consumer - it doesn't forward or store any video itself.
type Consumer struct {
	core.Connection

	onSEI func(json.RawMessage)
	last  string
}

func NewConsumer(onSEI func(json.RawMessage)) *Consumer {
	return &Consumer{
		Connection: core.Connection{
			ID:         core.NewID(),
			FormatName: "thingino/sei",
			Medias: []*core.Media{
				{
					Kind:      core.KindVideo,
					Direction: core.DirectionSendonly,
					Codecs:    []*core.Codec{{Name: core.CodecH264}},
				},
			},
		},
		onSEI: onSEI,
	}
}

func (c *Consumer) AddTrack(media *core.Media, _ *core.Codec, track *core.Receiver) error {
	handler := core.NewSender(media, track.Codec.Clone())

	handler.Handler = func(packet *rtp.Packet) {
		payload, ok := ExtractSEI(packet.Payload)
		if !ok {
			return
		}
		s := string(payload)
		if s == c.last {
			return
		}
		c.last = s
		c.onSEI(json.RawMessage(payload))
	}

	if track.Codec.IsRTP() {
		handler.Handler = h264.RTPDepay(track.Codec, handler.Handler)
	} else {
		handler.Handler = h264.RepairAVCC(track.Codec, handler.Handler)
	}

	handler.HandleRTP(track)
	c.Senders = append(c.Senders, handler)

	return nil
}
