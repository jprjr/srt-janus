#include "audio_fifo.h"
#include <libavcodec/avcodec.h>
#include <libavutil/avutil.h>
#include <libavutil/channel_layout.h>
#include <libavutil/opt.h>
#include <libavutil/mem.h>

#define BUFFER_SAMPLES 2048

#define SAMPLERATE 48000
#define CHANNELS 2

#ifdef DEBUG
#include <stdio.h>
#define DEBUG_LOG(...) fprintf(stderr,...)
#else
#define DEBUG_LOG(...)
#endif

audio_fifo_t *
audio_fifo_new(void) {
    audio_fifo_t *f = NULL;

    f = (audio_fifo_t *)av_mallocz(sizeof(audio_fifo_t));
    if(f == NULL) return NULL;

    f->frame = av_frame_alloc();
    if(f->frame == NULL) {
        av_free(f);
        return NULL;
    }

    f->frame->sample_rate = SAMPLERATE;
    f->frame->format = AV_SAMPLE_FMT_S16;
    f->frame->channel_layout = av_get_default_channel_layout(CHANNELS);
    f->frame->channels = CHANNELS;

    f->fifo = av_audio_fifo_alloc(AV_SAMPLE_FMT_S16, CHANNELS, BUFFER_SAMPLES);
    if(f->fifo == NULL) {
        av_frame_free(&f->frame);
        av_free(f);
        return NULL;
    }

    av_samples_alloc(&f->buffer,NULL,CHANNELS,BUFFER_SAMPLES,AV_SAMPLE_FMT_S16,0);
    if(f->buffer == NULL) {
        av_audio_fifo_free(f->fifo);
        av_frame_free(&f->frame);
        av_free(f);
        return NULL;
    }
    f->bufferSize = BUFFER_SAMPLES;
    memset(f->buffer,0,av_get_bytes_per_sample(AV_SAMPLE_FMT_S16) * CHANNELS * f->bufferSize);

    f->resampler = swr_alloc();
    if(f->resampler == NULL) {
        av_freep(&f->buffer);
        av_audio_fifo_free(f->fifo);
        av_frame_free(&f->frame);
        av_free(f);
        return NULL;
    }

    av_opt_set_channel_layout(f->resampler, "out_channel_layout", av_get_default_channel_layout(CHANNELS), 0);
    av_opt_set_int(f->resampler,"out_sample_rate", SAMPLERATE, 0);
    av_opt_set_sample_fmt(f->resampler,"out_sample_fmt", AV_SAMPLE_FMT_S16, 0);

    av_opt_set_int(f->resampler,"in_sample_rate", SAMPLERATE, 0);
    av_opt_set_sample_fmt(f->resampler,"in_sample_fmt", AV_SAMPLE_FMT_FLTP, 0);
    av_opt_set_channel_layout(f->resampler, "in_channel_layout", av_get_default_channel_layout(CHANNELS), 0);

    if(swr_init(f->resampler) < 0) {
        audio_fifo_close(f);
        return NULL;
    }

    return f;
}


void audio_fifo_close(audio_fifo_t *f) {
    if(f->frame != NULL) {
        av_frame_free(&f->frame);
    }
    if(f->buffer != NULL) {
        av_freep(&f->buffer);
    }
    if(f->fifo != NULL) {
        av_audio_fifo_free(f->fifo);
    }
    if(f->resampler != NULL) {
        swr_free(&f->resampler);
    }
    av_free(f);
}

static int
audio_fifo_realloc_buffer(audio_fifo_t *f, uint32_t size) {
    uint8_t *newBuffer = NULL;
    uint32_t newBufferSize = f->bufferSize;

    if(size > newBufferSize) {
        while(newBufferSize < (uint32_t)size) {
            newBufferSize *= 2;
        }
        av_samples_alloc(&newBuffer,NULL,CHANNELS,newBufferSize,AV_SAMPLE_FMT_FLTP,0);
        if(newBuffer == NULL) {
            return 1;
        }
        f->bufferSize = newBufferSize;
        memset(f->buffer,0,av_get_bytes_per_sample(AV_SAMPLE_FMT_FLTP * CHANNELS * f->bufferSize));
        av_freep(&f->buffer);
        f->buffer = newBuffer;
    }

    return 0;
}

int audio_fifo_load(audio_fifo_t *f, AVFrame *frame) {
    int samples_needed;
    int out_samples;

    samples_needed = swr_get_out_samples(f->resampler,frame->nb_samples);
    if(audio_fifo_realloc_buffer(f,(uint32_t)samples_needed) != 0) {
        return 1;
    }

    out_samples = swr_convert(f->resampler,
        &f->buffer,f->bufferSize,
        (const uint8_t **)frame->data, frame->nb_samples);

    if(out_samples > av_audio_fifo_space(f->fifo)) {
        av_audio_fifo_realloc(f->fifo,out_samples + av_audio_fifo_size(f->fifo));
    }

    av_audio_fifo_write(f->fifo,(void **)&f->buffer,out_samples);

    DEBUG_LOG("new audio_fifo_size: %d\n",av_audio_fifo_size(f->fifo));

    return 0;
}

uint32_t
audio_fifo_size(audio_fifo_t *f) {
    return (uint32_t) av_audio_fifo_size(f->fifo);
}

AVFrame *
audio_fifo_read(audio_fifo_t *f, uint32_t samples) {
    uint32_t bufferSizeBytes;
    int r;

    if( (uint32_t)av_audio_fifo_size(f->fifo) < samples) {
        DEBUG_LOG("not enough samples\n");
        return NULL;
    }

    if(audio_fifo_realloc_buffer(f,samples) != 0) {
        DEBUG_LOG("could not realloc buffer\n");
        return NULL;
    }

    DEBUG_LOG("fifo: reading %u samples\n",samples);
    r = av_audio_fifo_read(f->fifo,(void **)&f->buffer,samples);
    if(r < 0) {
        DEBUG_LOG("error reading from fifo: %d\n",r);
        return NULL;
    }
    DEBUG_LOG("fifo: read %d samples\n",r);

    f->frame->nb_samples = samples;
    bufferSizeBytes = av_get_bytes_per_sample(AV_SAMPLE_FMT_S16) * CHANNELS * f->bufferSize;

    r = avcodec_fill_audio_frame(f->frame,CHANNELS,AV_SAMPLE_FMT_S16,f->buffer,bufferSizeBytes,0);

    if(r < 0) {
        DEBUG_LOG("error filling audio frame: %d\n",r);
        return NULL;
    }

    return f->frame;
}

AVFrame *
audio_fifo_flush(audio_fifo_t *f) {
    uint32_t bufferSizeBytes;
    int r;
    uint32_t samples;

    /* flush out remaining samples */
    samples = swr_convert(f->resampler,
        &f->buffer,f->bufferSize,
        NULL, 0);

    if(samples > 0) {
        if(samples > (uint32_t)av_audio_fifo_space(f->fifo)) {
            av_audio_fifo_realloc(f->fifo,samples + av_audio_fifo_size(f->fifo));
        }
        av_audio_fifo_write(f->fifo,(void **)&f->buffer,samples);
    }

    samples = (uint32_t)av_audio_fifo_size(f->fifo);

    if(samples == 0) {
        return NULL;
    }

    if(audio_fifo_realloc_buffer(f,samples) != 0) {
        DEBUG_LOG("could not realloc buffer\n");
        return NULL;
    }

    r = av_audio_fifo_read(f->fifo,(void **)&f->buffer,samples);
    if(r < 0) {
        DEBUG_LOG("error reading from fifo: %d\n",r);
        return NULL;
    }
    f->frame->nb_samples = samples;
    bufferSizeBytes = av_get_bytes_per_sample(AV_SAMPLE_FMT_S16) * CHANNELS * f->bufferSize;

    r = avcodec_fill_audio_frame(f->frame,CHANNELS,AV_SAMPLE_FMT_S16,f->buffer,bufferSizeBytes,0);

    if(r < 0) {
        DEBUG_LOG("error filling audio frame: %d\n",r);
        return NULL;
    }

    return f->frame;
}

void
audio_fifo_reset(audio_fifo_t *f) {
    av_audio_fifo_reset(f->fifo);
}
