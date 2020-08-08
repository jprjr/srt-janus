#include "opus_encoder.h"
#include <stddef.h>
#include <libavutil/mem.h>

#define SAMPLERATE 48000

#define CHANNELS 2

#ifdef DEBUG
#include <stdio.h>
#define DEBUG_LOG(...) fprintf(stderr,...)
#else
#define DEBUG_LOG(...)
#endif

static AVCodec *codec = NULL;

int
opus_encoder_init(void) {
    codec = avcodec_find_encoder_by_name("libopus");
    return codec == NULL;
}

opus_encoder_t *
opus_encoder_new(void) {
    int r;
    opus_encoder_t *e = NULL;

    if(codec == NULL) return NULL;

    e = (opus_encoder_t *)av_mallocz(sizeof(opus_encoder_t));
    if(e == NULL) return NULL;

    e->opts = NULL;
    e->ctx  = avcodec_alloc_context3(codec);
    if(e->ctx == NULL) {
        av_free(e);
        return NULL;
    }

    e->packet = (AVPacket *)av_mallocz(sizeof(AVPacket));
    if(e->packet == NULL) {
        avcodec_free_context(&e->ctx);
        av_free(e);
        return NULL;
    }

    av_init_packet(e->packet);
    e->packet->data = NULL;

    e->ctx->sample_rate = SAMPLERATE;
    e->ctx->sample_fmt = AV_SAMPLE_FMT_S16;
    e->ctx->channel_layout = av_get_default_channel_layout(CHANNELS);
    e->ctx->channels = CHANNELS;

    av_dict_set(&e->opts,"framesize","20",0);
    av_dict_set(&e->opts,"vbr","off",0);
    av_dict_set(&e->opts,"b","96k",0);

    r = avcodec_open2(e->ctx,codec,&e->opts);
    if(r != 0) {
        opus_encoder_close(e);
        return NULL;
    }

    return e;
}

void
opus_encoder_close(opus_encoder_t *e) {
    if(e->ctx != NULL) avcodec_free_context(&e->ctx);
    if(e->opts != NULL) av_dict_free(&e->opts);
    if(e->packet != NULL) av_free_packet(e->packet);
    av_free(e->packet);
    av_free(e);

}


AVPacket *
opus_encoder_encode(opus_encoder_t *e, AVFrame *frame) {
    int got = 0;
    int r;

    r = avcodec_encode_audio2(e->ctx,e->packet,frame,&got);
    if(r < 0) {
        DEBUG_LOG("error encoding frame: %d\n",r);
        return NULL;
    }

    if(got) {
        return e->packet;
    }

    DEBUG_LOG("got = 0, returning NULL\n");


    return NULL;
}

