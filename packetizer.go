package main

// modified version of https://raw.githubusercontent.com/pion/rtp/master/packetizer.go that uses timestamps directly

import (
    rtp "github.com/pion/rtp"
	"time"
)


// Packetizer packetizes a payload
type TSPacketizer interface {
	Packetize(payload []byte, samples uint32) []*rtp.Packet
	SetTimestamp(timestamp uint32)
	GetTimestamp() uint32
}

type tspacketizer struct {
	MTU              int
	PayloadType      uint8
	SSRC             uint32
	Payloader        rtp.Payloader
	Sequencer        rtp.Sequencer
	Timestamp        uint32
	ClockRate        uint32
	timegen func() time.Time
}

// NewPacketizer returns a new instance of a Packetizer for a specific payloader
func NewTSPacketizer(mtu int, pt uint8, ssrc uint32, payloader rtp.Payloader, sequencer rtp.Sequencer, clockRate uint32) TSPacketizer {
	return &tspacketizer{
		MTU:         mtu,
		PayloadType: pt,
		SSRC:        ssrc,
		Payloader:   payloader,
		Sequencer:   sequencer,
		Timestamp:   0,
		ClockRate:   clockRate,
		timegen:     time.Now,
	}
}

func (p *tspacketizer) Packetize(payload []byte, samples uint32) []*rtp.Packet {
	// Guard against an empty payload
	if len(payload) == 0 {
		return nil
	}

	payloads := p.Payloader.Payload(p.MTU-12, payload)
	packets := make([]*rtp.Packet, len(payloads))


	for i, pp := range payloads {
		packets[i] = &rtp.Packet{
			Header: rtp.Header{
				Version:        2,
				Padding:        false,
				Extension:      false,
				Marker:         i == len(payloads)-1,
				PayloadType:    p.PayloadType,
				SequenceNumber: p.Sequencer.NextSequenceNumber(),
				Timestamp:      p.Timestamp,
				SSRC:           p.SSRC,
			},
			Payload: pp,
		}
	}

    p.Timestamp += samples

	return packets
}

func (p *tspacketizer) SetTimestamp(timestamp uint32) {
    p.Timestamp = timestamp
}

func (p *tspacketizer) GetTimestamp() uint32 {
    return p.Timestamp
}
