#ifndef OPUS_ENCODER_H
#define OPUS_ENCODER_H

#include <libavcodec/avcodec.h>
#include <libavutil/frame.h>
#include <stdint.h>

typedef struct {
    AVCodecContext *ctx;
    AVDictionary *opts;
    AVPacket *packet;
} srtjanus_opus_encoder_t;

#ifdef __cplusplus
extern "C" {
#endif

int
srtjanus_opus_encoder_init(void);

srtjanus_opus_encoder_t *
srtjanus_opus_encoder_new(void);

void
srtjanus_opus_encoder_close(srtjanus_opus_encoder_t *e);

AVPacket *
srtjanus_opus_encoder_encode(srtjanus_opus_encoder_t *e, AVFrame *frame);

#ifdef __cplusplus
}
#endif

#endif
