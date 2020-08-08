package main

/*
#include "aac_decoder.h"
#include "opus_encoder.h"
*/
import "C"

import (
    "strconv"
    "context"
    "log"
    ts "github.com/asticode/go-astits"
    janus "github.com/notedit/janus-go"
    webrtc "github.com/pion/webrtc/v2"
    rtp "github.com/pion/rtp"
)

type srtConnection struct {
    m webrtc.MediaEngine
    streamid string
    handle *janus.Handle
    videoPID uint16
    audioPID uint16
    videoTrack *webrtc.Track
    audioTrack *webrtc.Track
    opusCodec *webrtc.RTPCodec
    h264Codec *webrtc.RTPCodec
    audioHandler MediaHandler
    videoHandler MediaHandler
}


// runs in a thread just reading and posting data
func pumpData(dmx *ts.Demuxer, c chan *ts.Data, q chan int) {
    for {
        d, err := dmx.NextData()
        if err != nil {
            d = nil
        }
        select {
            case c <- d:
                break
            case <-q:
                log.Println("Data Pumper received quit, ending pump")
                return
        }
        if d == nil {
            return
        }
    }
}

func (conn *srtConnection) attachJanus(c chan *janus.Handle, session *janus.Session, roomId uint64) {

    config := webrtc.Configuration{
        ICEServers: []webrtc.ICEServer{
            {
                URLs: []string{"stun:stun.l.google.com:19302"},
            },
        },
        SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
    }

    // Create a new RTCPeerConnection
    pc, err := webrtc.NewAPI(webrtc.WithMediaEngine(conn.m)).NewPeerConnection(config)
    if err != nil {
        log.Println("Failed to create peerconnection", err)
        c <- nil
        return
    }

    pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
        log.Println("Connection State has changed", connectionState.String())
    })

    if conn.audioPID != 0 {
        ssrc := RandUint32()
        id := MathRandAlpha(16)
        label := MathRandAlpha(16)

        audioPacketizer := NewTSPacketizer(
            1200,
            webrtc.DefaultPayloadTypeOpus,
            ssrc,
            conn.opusCodec.Payloader,
            rtp.NewRandomSequencer(),
            conn.opusCodec.ClockRate,
        )

        conn.audioHandler.SetPacketizer(audioPacketizer)

        // Create audio track, creates an extra packetizer but oh well
        audioTrack, err := pc.NewTrack(webrtc.DefaultPayloadTypeOpus, ssrc, id, label)
        if err != nil {
            log.Println("Failed to create audiotrack", err)
            c <- nil
            return
        }

        _, err = pc.AddTrack(audioTrack)
        if err != nil {
            log.Println("Failed to add audiotrack", err)
            c <- nil
            return
        }

        conn.audioHandler.SetTrack(audioTrack)
    }

    if conn.videoPID != 0 {
        ssrc := RandUint32()
        id := MathRandAlpha(16)
        label := MathRandAlpha(16)

        videoPacketizer := NewTSPacketizer(
            1200,
            webrtc.DefaultPayloadTypeH264,
            ssrc,
            conn.h264Codec.Payloader,
            rtp.NewRandomSequencer(),
            conn.h264Codec.ClockRate,
        )

        conn.videoHandler.SetPacketizer(videoPacketizer)

        // Create video track
        videoTrack, err := pc.NewTrack(webrtc.DefaultPayloadTypeH264, ssrc, id, label)
        if err != nil {
            log.Println("Failed to create videotrack", err)
            c <- nil
            return
        }
        _, err = pc.AddTrack(videoTrack)
        if err != nil {
            log.Println("Failed to add videotrack", err)
            c <- nil
            return
        }
        conn.videoHandler.SetTrack(videoTrack)

    }

    offer, err := pc.CreateOffer(nil)
    if err != nil {
        log.Println("Failed to create offer", err)
        c <- nil
        return
    }

    err = pc.SetLocalDescription(offer)
    if err != nil {
        log.Println("Failed to setlocaldescription",err)
        c <- nil
        return
    }

    handle, err := session.Attach("janus.plugin.videoroom")
    if err != nil {
        log.Println("Failed to attach",err)
        c <- nil
        return
    }

    _, err = handle.Message(map[string]interface{}{
        "request": "join",
        "ptype":   "publisher",
        "room":    roomId,
        "id":      RandUint32(),
    }, nil)
    if err != nil {
        log.Println("Failed to send handle join", err)
        c <- nil
        return
    }

    pub_request := map[string]interface{}{
        "request": "publish",
        "data":    false,
    }

    if conn.videoPID != 0 {
        pub_request["video"] = true
        pub_request["videocodec"] = "h264"
    }

    if conn.audioPID != 0 {
        pub_request["audio"] = true
        pub_request["audiocodec"] = "opus"
    }


    msg, err := handle.Message(pub_request, map[string]interface{}{
        "type":    "offer",
        "sdp":     offer.SDP,
        "trickle": false,
    })
    if err != nil {
        log.Println("Failed to send handle publish", err)
        c <- nil
        return
    }

    if msg.Jsep != nil {
        err = pc.SetRemoteDescription(webrtc.SessionDescription{
            Type: webrtc.SDPTypeAnswer,
            SDP:  msg.Jsep["sdp"].(string),
        })
        if err != nil {
            log.Println("Failed to setremotedescription", err)
            c <- nil
            return
        }
    }

    // send the handle back to the main thread
    c <- handle
    return
}

func newSrtConnection(m webrtc.MediaEngine, opusCodec *webrtc.RTPCodec, h264Codec *webrtc.RTPCodec) *srtConnection {
    conn := new(srtConnection)
    conn.m = m
    conn.handle = nil
    conn.videoPID = 0
    conn.audioPID = 0
    conn.opusCodec = opusCodec
    conn.h264Codec = h264Codec

    return conn
}

func (conn *srtConnection) Run(reader *srtReader, session *janus.Session) {
    defer reader.Close()

    var err error
    var d *ts.Data = nil
    var i int = 0
    var roomId uint64 = 0

    data_chan := make(chan *ts.Data)
    quit_chan := make(chan int)
    handle_chan := make(chan *janus.Handle)

    defer close(data_chan)
    defer close(quit_chan)
    defer close(handle_chan)

    if roomId, err = strconv.ParseUint(reader.GetStreamID(),10,64); err != nil {
        log.Println("error parsing room id", err)
        return
    }

    ctx, _ := context.WithCancel(context.Background())
    dmx := ts.New(ctx,reader)

    go pumpData(dmx,data_chan,quit_chan)

    for {
        /* try to find a PMT within the first 1000 packets, quit if not found */
        if i == 1000 {
            d = nil
            quit_chan <- 0
            break
        }
        d = <-data_chan
        if d == nil {
            break
        }
        if d.PMT != nil {
            break
        }
        i++
    }

    if d == nil {
        return
    }

    /* find our audio and video PIDs */
    for _, es := range d.PMT.ElementaryStreams {
        log.Println("StreamType:",es.StreamType)
        if es.StreamType == 0x1B { // H.264
            conn.videoPID = es.ElementaryPID
            conn.videoHandler = NewMediaHandlerH264()
            defer conn.videoHandler.Close()
        } else if es.StreamType == 0x0F { // AAC
            conn.audioPID = es.ElementaryPID
            conn.audioHandler = NewMediaHandlerAAC()
            defer conn.audioHandler.Close()
        } else if es.StreamType == 0x06 { // Private, maybe Opus
            for _, desc := range es.ElementaryStreamDescriptors {
                if desc.Tag == 0x05 {
                    // 0x4F707573 == 'Opus'
                    if desc.Registration.FormatIdentifier == 0x4F707573 {
                        conn.audioPID = es.ElementaryPID
                        conn.audioHandler = NewMediaHandlerOpus()
                        defer conn.audioHandler.Close()
                    }
                }
            }
        }
    }

    if conn.videoPID == 0 && conn.audioPID == 0 {
        log.Println("Unable to find video and audio PIDs in PMT table")
        return
    }

    go conn.attachJanus(handle_chan, session, roomId)

AttachJanusLoop:
    for {
        select {
            case d = <-data_chan:
                if d == nil {
                    break AttachJanusLoop
                }
            case conn.handle = <-handle_chan:
                break AttachJanusLoop
        }
    }

    if conn.handle == nil {
        quit_chan <- 0
        return
    }
    defer conn.handle.Detach()

    log.Println("Connected, forwarding audio/video")

ForwardPacketsLoop:
    for {
        select {
            case msg := <-conn.handle.Events:
                switch msg := msg.(type) {
                case *janus.SlowLinkMsg:
                    log.Println("SlowLinkMsg type ", conn.handle.ID)
                case *janus.MediaMsg:
                    log.Println("MediaEvent type", msg.Type, " receiving ", msg.Receiving)
                case *janus.WebRTCUpMsg:
                    log.Println("WebRTCUp type ", conn.handle.ID)
                case *janus.HangupMsg:
                    log.Println("HangupEvent type ", conn.handle.ID)
                case *janus.EventMsg:
                    log.Printf("EventMsg %+v", msg.Plugindata.Data)
                }

                break
            case d = <-data_chan:
                if d == nil {
                    break ForwardPacketsLoop
                }
                if d.PES != nil {
                    if d.PID == conn.audioPID {
                        conn.audioHandler.SendMedia(d.PES)
                    } else if d.PID == conn.videoPID {
                        conn.videoHandler.SendMedia(d.PES)
                    }
                }
        }
    }

    _, err = conn.handle.Message(map[string]interface{}{
        "request": "leave",
    }, nil)
    if err != nil {
        log.Println("Failed to send handle join", err)
        return
    }

    quit_chan <- 0
}
