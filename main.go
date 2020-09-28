package main

/*
#cgo pkg-config: srt libavcodec libavutil libswresample
#include <srt/srt.h>
#include <libavcodec/avcodec.h>
#include "aac_decoder.h"
#include "opus_encoder.h"

extern int goSrtCallback(void *, SRTSOCKET ns, int hsversion, struct sockaddr* peeraddr, char *streamid);

int local_srt_callback(void* opaq, SRTSOCKET ns, int hsversion, const struct sockaddr* peeraddr, const char* streamid) {
    return goSrtCallback(opaq,ns,hsversion,(struct sockaddr *)peeraddr,(char *)streamid);
}

*/
import "C"

import (
    "strconv"
    "fmt"
    "time"
    "os"
    srtgo "github.com/haivision/srtgo"
    janus "github.com/notedit/janus-go"
    webrtc "github.com/pion/webrtc/v2"
    "github.com/pion/rtp/codecs"
    "strings"
    "flag"
)


func setupMediaEngine(opusCodec *webrtc.RTPCodec, h264Codec *webrtc.RTPCodec) webrtc.MediaEngine {

    mediaEngine := webrtc.MediaEngine{}
    mediaEngine.RegisterCodec(opusCodec)
    mediaEngine.RegisterCodec(h264Codec)

    return mediaEngine
}

func main() {
    var port uint64 = 0
    var ip string
    var err error

    janus_options := make(map[string]interface{})

    // first handle flags
    displayPtr := flag.String("display", "external", "Display name to set in Janus VideoRoom")

    flag.Parse()
    args := flag.Args()

    if len(args) < 2 {
        fmt.Printf("Usage: %s -display=external <listen-addr> <url>\n",os.Args[0])
        os.Exit(1)
    }

    janus_options["display"] = *displayPtr

    listen_parts := strings.Split(args[0],":")

    if len(listen_parts) == 1 {
        //assuming just specified a port
        ip = "0.0.0.0"
        port, err = strconv.ParseUint(listen_parts[0],10,16)
        if err != nil {
            fmt.Printf("Invalid port\n")
            os.Exit(1)
        }
    } else {
        if len(listen_parts[0]) == 0 {
            ip = "0.0.0.0"
        } else {
            ip = listen_parts[0]
        }
        port, err = strconv.ParseUint(listen_parts[1],10,16)
        if err != nil {
            fmt.Printf("Invalid port\n")
            os.Exit(1)
        }
    }

    srtgo.InitSRT()
    C.avcodec_register_all()
    C.srtjanus_aac_decoder_init()
    C.srtjanus_opus_encoder_init()
    defer srtgo.CleanupSRT()

    gateway, err := janus.Connect(args[1])
    if err != nil {
        fmt.Printf("Failed to connect to janus: %s\n",err)
        return
    }
    defer gateway.Close()

    // create a session
    session, err := gateway.Create()
    if err != nil {
        fmt.Printf("Failed to create session: %s\n",err)
        return
    }
    defer session.Destroy()

    // start a keepliave timer for the session
    go func() {
        for {
            if _, keepAliveErr := session.KeepAlive(); err != nil {
                panic(keepAliveErr)
            }

            time.Sleep(5 * time.Second)
        }
    }()

    // opus
    opusCodec := webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000)

    // custom h264 codec - need to indicate constrained baseline profile
    // with "profile-level-id=42e01f" (default is 42001f - unconstrained baseline)
    h264Codec := webrtc.NewRTPCodec(webrtc.RTPCodecTypeVideo,
          webrtc.H264,
          90000,
          0,
          "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f",
          webrtc.DefaultPayloadTypeH264,
          &codecs.H264Payloader{})


    mediaEngine := setupMediaEngine(opusCodec, h264Codec)

    options := make(map[string]string)
    options["blocking"] = "1"
    options["transtype"] = "live"

    socket := srtgo.NewSrtSocket(ip,uint16(port),options)
    C.srt_listen_callback(C.SRTSOCKET(getInternalSocket(socket)),(*C.srt_listen_callback_fn)(C.local_srt_callback),nil)

    socket.Listen(20)
    defer socket.Close()

    for {
        s, err := socket.Accept()
        if err != nil {
            panic("Error on accept")
            break
        }

        conn := newSrtConnection(mediaEngine,opusCodec,h264Codec)
        go conn.Run(newSrtReader(s),session,janus_options)
    }
}
