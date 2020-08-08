package main

/*
#include <srt/srt.h>
*/
import "C"

import (
   srtgo "github.com/haivision/srtgo"
   "bytes"
   "unsafe"
)

// buffered reader for SRT connections, provides io.Reader interface

type srtReader struct {
    socket *srtgo.SrtSocket
    buffer bytes.Buffer
    tmp []byte
}

func newSrtReader(socket *srtgo.SrtSocket) *srtReader {
    reader := new(srtReader)
    reader.socket = socket

    var payloadSize C.int32_t = 0
    var payloadLen  C.int = 0
    C.srt_getsockflag(C.SRTSOCKET(getInternalSocket(reader.socket)),C.SRTO_PAYLOADSIZE,unsafe.Pointer(&payloadSize),(*C.int)(unsafe.Pointer(&payloadLen)))
    reader.buffer.Grow(int(payloadSize))
    reader.tmp = make([]byte,int(payloadSize))

    return reader
}

func (reader *srtReader) GetStreamID() string {
    id_buffer := make([]byte,512)
    var id_bufferLen C.int = 512
    C.srt_getsockflag(C.SRTSOCKET(getInternalSocket(reader.socket)),C.SRTO_STREAMID,unsafe.Pointer(&id_buffer[0]),(*C.int)(unsafe.Pointer(&id_bufferLen)))

    return string(id_buffer[0:id_bufferLen])
}


func (reader *srtReader) Read(p []byte) (int, error) {
    var n int = 0
    var err error = nil
    var rem int = len(p) - reader.buffer.Len()

    for rem > 0 {
        n, err = reader.socket.Read(reader.tmp,100)
        if err != nil {
            return 0, err
        }

        if n == 0 {
            break
        }

        n, err = reader.buffer.Write(reader.tmp[0:n])
        if err != nil {
            return 0, err
        }

        if n == 0 {
            break
        }

        rem = rem - n
    }

    return reader.buffer.Read(p)
}

func (reader *srtReader) Close() {
    reader.socket.Close()
}

