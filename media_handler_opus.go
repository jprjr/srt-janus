package main

import (
    ts "github.com/asticode/go-astits"
    webrtc "github.com/pion/webrtc/v2"
    "errors"
)

type mediaHandlerOpus struct {
    audioTrack *webrtc.Track
    audioPacketizer TSPacketizer
}

func NewMediaHandlerOpus() MediaHandler {
    return &mediaHandlerOpus{ }
}

func (h *mediaHandlerOpus) SetPacketizer(audioPacketizer TSPacketizer) {
    h.audioPacketizer = audioPacketizer
}

func (h *mediaHandlerOpus) SetTrack(audioTrack *webrtc.Track) {
    h.audioTrack = audioTrack
}

// https://tools.ietf.org/html/rfc6716#section-3.1
// converting ms values to 48kHz # of samples
var sample_lengths = [32]uint32 {
    480, 960, 1920, 2880,
    480, 960, 1920, 2880,
    480, 960, 1920, 2880,
    480, 960,
    480, 960,
    120, 240, 480, 960,
    120, 240, 480, 960,
    120, 240, 480, 960,
    120, 240, 480, 960,
}

func (h *mediaHandlerOpus) SendMedia(data *ts.PESData) error {
    /* loop through, extract opus packets and send */
    var pos uint32 = 0
    var start_trim_flag bool = false
    var end_trim_flag bool = false
    var control_extension_flag bool = false
    var reserved uint8 = 0x00
    var header uint16 = 0x0000
    var au_size uint32 = 0
    var start_trim uint16 = 0
    var end_trim uint16 = 0
    var control_extension_length uint8 = 0
    var sample_length_index uint8 = 0
    var stereo bool = false
    var config uint8 = 0

    var ts uint64 = uint64(data.Header.OptionalHeader.PTS.Base)
    packetLength := len(data.Data)

    ts *= 48000
    ts /= 90000

    h.audioPacketizer.SetTimestamp(uint32(ts))

    for int(pos) < packetLength  {

        header = (uint16(data.Data[pos+0]) << 8) + uint16(data.Data[pos+1])
        pos += 2
        if header & 0xFFE0 != 0x7FE0 {
            return errors.New("Invalid Opus Access Unit")
        }

        start_trim_flag        =      (header & 0x0010) == 0x0010
        end_trim_flag          =      (header & 0x0008) == 0x0008
        control_extension_flag =      (header & 0x0004) == 0x0004
        reserved               = uint8(header & 0x0003)

        au_size = 0
        for data.Data[pos] == 0xFF {
            au_size += 255
            pos++
        }

        au_size += uint32(data.Data[pos])
        pos++

        if start_trim_flag {
            start_trim = ((uint16(data.Data[pos+0]) << 8) + uint16(data.Data[pos+1])) & 0x1FFF
            pos += 2
        }

        if end_trim_flag {
            end_trim = ((uint16(data.Data[pos+0]) << 8) + uint16(data.Data[pos+1])) & 0x1FFF
            pos += 2
        }

        if control_extension_flag {
            control_extension_length = uint8(data.Data[pos])
            pos++
            for control_extension_length > 0 {
                control_extension_length--
                pos++
            }
        }

        _ = reserved
        _ = start_trim
        _ = end_trim

        sample_length_index = data.Data[pos] >> 3
        stereo = data.Data[pos] & 0x04 == 0x04
        config = data.Data[pos] & 0x03

        _ = config
        _ = stereo

        slice := data.Data[pos:pos+au_size]

        packets := h.audioPacketizer.Packetize(slice, sample_lengths[sample_length_index])
        for _, p := range packets {
            err := h.audioTrack.WriteRTP(p)
            if err != nil {
                return err
            }
        }

        pos += au_size
    }

    return nil

}

func (h *mediaHandlerOpus) Close() {
}


