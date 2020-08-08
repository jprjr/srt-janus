#ifndef AAC_DECODER_H
#define AAC_DECODER_H

#include <libavcodec/avcodec.h>
#include <libavutil/frame.h>
#include <stdint.h>
#include "audio_fifo.h"

typedef struct {
    AVCodecContext *ctx;
    AVFrame *f;
    int got;
    audio_fifo_t *fifo;
} aac_decoder_t;

#ifdef __cplusplus
extern "C" {
#endif

int
aac_decoder_init(void); /* must be called very early in process */

aac_decoder_t *
aac_decoder_new(void);

void
aac_decoder_close(aac_decoder_t *d);


int
aac_decoder_decode(aac_decoder_t *d, uint8_t *data, size_t len);

AVFrame *
aac_decoder_read(aac_decoder_t *d, uint32_t samples);

uint32_t
aac_decoder_size(aac_decoder_t *d);

void
aac_decoder_reset(aac_decoder_t *d);


#ifdef __cplusplus
}
#endif

#endif
