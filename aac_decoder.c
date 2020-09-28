#include <stddef.h>
#include <libavutil/mem.h>
#include "aac_decoder.h"
#include "audio_fifo.h"

#define SAMPLERATE 48000
#define CHANNELS 2

#ifdef DEBUG
#include <stdio.h>
#define DEBUG_LOG(...) fprintf(stderr,__VA_ARGS__)
#else
#define DEBUG_LOG(...)
#endif

static AVCodec *codec = NULL;

int
srtjanus_aac_decoder_init(void) {
    codec = avcodec_find_decoder(AV_CODEC_ID_AAC);
    return codec == NULL;
}

srtjanus_aac_decoder_t *
srtjanus_aac_decoder_new(void) {
    int r;
    srtjanus_aac_decoder_t *d = NULL;
    if(codec == NULL) return NULL;

    d = (srtjanus_aac_decoder_t *)av_mallocz(sizeof(srtjanus_aac_decoder_t));
    if(d == NULL) return NULL;

    d->f = av_frame_alloc();
    if(d->f == NULL) {
        av_free(d);
        return NULL;
    }

    d->ctx = avcodec_alloc_context3(codec);
    if(d->ctx == NULL) {
        av_frame_free(&d->f);
        av_free(d);
        return NULL;
    }

    d->fifo = srtjanus_audio_fifo_new();
    if(d->fifo == NULL) {
        avcodec_free_context(&d->ctx);
        av_frame_free(&d->f);
        av_free(d);
        return NULL;
    }

    d->ctx->extradata = NULL;
    d->ctx->extradata_size = 0;
    d->ctx->sample_rate = SAMPLERATE;
    d->ctx->channels    = CHANNELS;

    if((r = avcodec_open2(d->ctx,codec,NULL)) < 0) {
        srtjanus_aac_decoder_close(d);
        return NULL;
    }

    return d;
}


void
srtjanus_aac_decoder_close(srtjanus_aac_decoder_t *d) {
    avcodec_free_context(&d->ctx);
    av_frame_free(&d->f);
    srtjanus_audio_fifo_close(d->fifo);
    av_free(d);
}

int
srtjanus_aac_decoder_decode(srtjanus_aac_decoder_t *d, uint8_t *data, size_t len) {
    AVPacket packet;
    int total;
    int got;
    int read;
    uint8_t *cpy;
    av_init_packet(&packet);

    cpy = av_mallocz((len+15) & ~0x07);
    if(cpy == NULL) return -1;
    memcpy(cpy,data,len);
    packet.data = cpy;

    read = 0;
    total = 0;

    while(len > 0) {
        packet.size = len;
        packet.data = &packet.data[read];

        read = avcodec_decode_audio4(d->ctx,d->f,&got,&packet);

        if(read < 0) {
            break;
        }

        if(!got || !read) break;
        DEBUG_LOG("read %d/%u bytes\n",read,len);

        srtjanus_audio_fifo_load(d->fifo,d->f);

        len -= read;
        total++;
    }

    av_free(cpy);
    return total;
}

AVFrame *
srtjanus_aac_decoder_read(srtjanus_aac_decoder_t *d, uint32_t samples) {
    return srtjanus_audio_fifo_read(d->fifo,samples);
}

uint32_t
srtjanus_aac_decoder_size(srtjanus_aac_decoder_t *d) {
    return srtjanus_audio_fifo_size(d->fifo);
}

void
srtjanus_aac_decoder_reset(srtjanus_aac_decoder_t *d) {
    avcodec_flush_buffers(d->ctx);
    srtjanus_audio_fifo_reset(d->fifo);
}
