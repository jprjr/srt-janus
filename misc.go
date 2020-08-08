package main

/*
#include <srt/srt.h>
*/
import "C"

import (
    srtgo "github.com/haivision/srtgo"
    "reflect"
    "unsafe"
)

// see https://stackoverflow.com/questions/42664837/how-to-access-unexported-struct-fields-in-golang
// see https://stackoverflow.com/questions/17981651/is-there-any-way-to-access-private-fields-of-a-struct-from-another-package
// see https://play.golang.org/p/DXXHuIvRLI
// need to access underlying "socket" field to pass to srt_listen_callback, getsockflag, etc

func getInternalSocket (socket *srtgo.SrtSocket) int64 {
    v := reflect.ValueOf(*socket)
    internal_s := v.FieldByName("socket")
    return internal_s.Int()
}

/* used for srt_listen_callback to verify incoming connection has a streamid */

//export goSrtCallback
func goSrtCallback(user_data unsafe.Pointer, ns C.SRTSOCKET, hsversion C.int, peeraddr *C.struct_sockaddr, streamid *C.char) int {
    goStreamid := C.GoString(streamid)
    if len(goStreamid) == 0 {
        // no stream id, rejecting
        return -1
    }
    return 0
}


