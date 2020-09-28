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
    srtjanus_audio_fifo_t *fifo;
} srtjanus_aac_decoder_t;

#ifdef __cplusplus
extern "C" {
#endif

int
srtjanus_aac_decoder_init(void); /* must be called very early in process */

srtjanus_aac_decoder_t *
srtjanus_aac_decoder_new(void);

void
srtjanus_aac_decoder_close(srtjanus_aac_decoder_t *d);


int
srtjanus_aac_decoder_decode(srtjanus_aac_decoder_t *d, uint8_t *data, size_t len);

AVFrame *
srtjanus_aac_decoder_read(srtjanus_aac_decoder_t *d, uint32_t samples);

uint32_t
srtjanus_aac_decoder_size(srtjanus_aac_decoder_t *d);

void
srtjanus_aac_decoder_reset(srtjanus_aac_decoder_t *d);


#ifdef __cplusplus
}
#endif

#endif
