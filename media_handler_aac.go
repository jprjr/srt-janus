package main

/*
#include "aac_decoder.h"
#include "opus_encoder.h"
*/
import "C"

import (
    ts "github.com/asticode/go-astits"
    webrtc "github.com/pion/webrtc/v2"
    "errors"
    "log"
    "unsafe"
)

type mediaHandlerAAC struct {
    audioTrack *webrtc.Track
    audioPacketizer TSPacketizer
    decoder *C.aac_decoder_t
    encoder *C.opus_encoder_t
    frames uint32
}

func NewMediaHandlerAAC() MediaHandler {
    return &mediaHandlerAAC{
        decoder:         C.aac_decoder_new(),
        encoder:         C.opus_encoder_new(),
        frames:          0,
    }
}

func (h *mediaHandlerAAC) SetPacketizer(audioPacketizer TSPacketizer) {
    h.audioPacketizer = audioPacketizer
}

func (h *mediaHandlerAAC) SetTrack(audioTrack *webrtc.Track) {
    h.audioTrack = audioTrack
}

func (h *mediaHandlerAAC) SendMedia(data *ts.PESData) error {
    // OK so, AAC media uses 1024 frames in a single packet,
    // but Opus uses 960 frames.

    // So I can't just use the PESData PTS (presentation time stamp)
    // since I'm only going to send part of this packet of data.

    // in my RTMP -> Janus gateway I just increase the timestamp by
    // 960 samples every time, which works since RTMP is over TCP,
    // I should be getting all the packets (though as I type this,
    // maybe that's not true?)

    // I think the right solution is:
    //   * On the first packet, just the PTS timestamp
    //   * Increase timestamp by 960 as I encode + send opus
    //   * every 15360 frames (least common multiple of 960 and 1024),
    //     reset the timestamp

    decoded_frames := C.aac_decoder_decode(h.decoder,(*C.uint8_t)(&data.Data[0]),C.size_t(len(data.Data)))
    if decoded_frames == 0 {
        log.Println("Error decoding AAC media")
        return errors.New("Error decoding AAC packet")
    }

    if h.frames % 15360 == 0 {
        h.frames = 0
    }

    if h.frames == 0 {
        var ts uint64 = uint64(data.Header.OptionalHeader.PTS.Base)
        ts *= 48000
        ts /= 90000

        h.audioPacketizer.SetTimestamp(uint32(ts))
    }


    for C.aac_decoder_size(h.decoder) >= 960 {
        raw_frame := C.aac_decoder_read(h.decoder,960);
        packet := C.opus_encoder_encode(h.encoder,raw_frame)
        if packet == nil {
            log.Println("Error encoding opus")
            return errors.New("Error encoding opus")
        }

        sample := make([]byte,packet.size)
        C.memcpy(
          unsafe.Pointer(&sample[0]),
          unsafe.Pointer(packet.data),
          C.size_t(packet.size))

        packets := h.audioPacketizer.Packetize(sample,960)

        for _, p := range packets {
            err := h.audioTrack.WriteRTP(p)
            if err != nil {
                return err
            }
        }

        h.frames += 960
    }

    return nil

}

func (h *mediaHandlerAAC) Close() {
    C.aac_decoder_close(h.decoder)
    C.opus_encoder_close(h.encoder)
}
