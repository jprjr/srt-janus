package main

import (
    ts "github.com/asticode/go-astits"
    webrtc "github.com/pion/webrtc/v2"
)

type mediaHandlerH264 struct {
    videoTrack *webrtc.Track
    videoPacketizer TSPacketizer
}

func NewMediaHandlerH264() MediaHandler {
    return &mediaHandlerH264{ }
}

func (h *mediaHandlerH264) SetPacketizer(videoPacketizer TSPacketizer) {
    h.videoPacketizer = videoPacketizer
}

func (h *mediaHandlerH264) SetTrack(videoTrack *webrtc.Track) {
    h.videoTrack = videoTrack
}

func (h *mediaHandlerH264) SendMedia(data *ts.PESData) error {
    h.videoPacketizer.SetTimestamp(uint32(data.Header.OptionalHeader.PTS.Base))
    packets := h.videoPacketizer.Packetize(data.Data,0)

    for _, p := range packets {
        err := h.videoTrack.WriteRTP(p)
        if err != nil {
            return err
        }
    }

    return nil

}

func (h *mediaHandlerH264) Close() {
}

