#ifndef AUDIO_FIFO_H
#define AUDIO_FIFO_H

#include <libavutil/audio_fifo.h>
#include <libswresample/swresample.h>
#include <stdint.h>

/* handles receiving decoded samples,
 * resample,
 * put into a fifo */

typedef struct {
    SwrContext *resampler;
    AVAudioFifo *fifo;
    AVFrame *frame;
    uint8_t *buffer;
    uint32_t bufferSize;
} srtjanus_audio_fifo_t;

#ifdef __cplusplus
extern "C" {
#endif

srtjanus_audio_fifo_t *
srtjanus_audio_fifo_new(void);

int
srtjanus_audio_fifo_load(srtjanus_audio_fifo_t *f, AVFrame *frame);

uint32_t
srtjanus_audio_fifo_size(srtjanus_audio_fifo_t *f);

AVFrame *
srtjanus_audio_fifo_read(srtjanus_audio_fifo_t *f, uint32_t samples);

AVFrame *
srtjanus_audio_fifo_flush(srtjanus_audio_fifo_t *f);

void
srtjanus_audio_fifo_close(srtjanus_audio_fifo_t *f);

void
srtjanus_audio_fifo_reset(srtjanus_audio_fifo_t *f);


#ifdef __cplusplus
}
#endif

#endif
