# SRT to Janus Videoroom

An app that forwards SRT streams into Janus videorooms.

## Warning: I am not a Go programmer

This is very much in the work-in-progress/proof-of-concept phase!

I really don't know go, I learned just enough so I could make use of the [`pion/webrtc`](https://github.com/pion/webrtc) library.

This is adapted from the Janus example in Pion's [`example-webrtc-applications`](https://github.com/pion/example-webrtc-applications) repo.

Pull requests are always welcome!

## Usage

```bash
srt-janus <listen host:port> ws://janus-host:port

# example: srt-janus :8090 ws://127.0.0.1:8188
```

## What does this do?

When launched, this app:

* Connects to Janus gateway via websocket, establishes a session.
* Starts listening for incoming [SRT](https://github.com/Haivision/srt) sessions.

When a SRT connection is received, it looks for a streamid parameter and uses that to pick a room ID.

Example: `srt://127.0.0.1:8090?streamid=1234` - the `1234` becomes the room ID.

The app then joins the janus videoroom and establishes a WebRTC session.

H264 data is forwarded as-is. AAC audio is encoded to Opus using libavcodec, Opus is forwarded as-is.

You **must** use:

  * MPEG-TS as the muxer
    * Max 1 video stream
    * Max 1 audio stream
  * H264 as the video codec
    * baseline profile
  * AAC-LC audio or Opus audio
    * 48kHz, 2-channel only

If you want to send Opus, make sure you're properly muxing it into MPEG-TS. One known bad muxer
is ffmpeg 2.8 (and older). ffmpeg 3.0 and newer will work properly.

## Example

By default, Janus has a demo videoroom with id `1234`, I believe h264 is disabled in the default config.

Edit your `janus.plugin.videoroom.jcfg' config file, find the section for room-1234, make it something like this:

```
room-1234: {
	description = "Demo Room"
	secret = "adminpwd"
	publishers = 6
	bitrate = 128000
	fir_freq = 10
	#audiocodec = "opus"
	videocodec = "h264,vp8"
	record = false
	#rec_dir = "/path/to/recordings-folder"
}
```

Restart janus, join a videoroom in your browser.

Run srt-janus:

```
./srt-janus :8090 ws://127.0.0.1:8188
```

Then fire up ffmpeg:

```
ffmpeg -re -i /path/to/some/file \
  -c:v libx264 \
  -profile:v baseline \
  -b:v 1000k \
  -bsf:v h264_mp4toannexb \
  -c:a aac \
  -b:a 96k \
  -ar 48000 \
  -ac 2 \
  -g 30 \
  -f mpegts \
  'srt://127.0.0.1:8090?streamid=1234'
```

If your ffmpeg doesn't have SRT enabled, you can use UDP instead and forward with `srt-live-transmit`:

```
srt-live-transmit udp://127.0.0.1:9999 srt://127.0.0.1:8090?streamid=1234 &
ffmpeg (arguments-from-above) udp://127.0.0.1:9999?pkt_size=1316

````

Your audio+video will appear in Janus as a member of the videoroom!

## TODO

* Figure out what events from Janus I should handle (I just blast audio/video).
* Test this with a bad connection, like something that randomly drops UDP packets.
* See if I can support VP8 as well.

## LICENSE

MIT (see `LICENSE`)

