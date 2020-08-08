package main

import (
    ts "github.com/asticode/go-astits"
    webrtc "github.com/pion/webrtc/v2"
)

// type for processing PES audio packets
type MediaHandler interface {
    SendMedia(data *ts.PESData) error
    SetPacketizer(audioPacketizer TSPacketizer)
    SetTrack(audioTrack *webrtc.Track)
    Close()
}

