import { useEffect, useState, type RefObject } from 'react'

const segmentDurationSeconds = 15
const defaultSegmentBytes = 600 * 1024

type SegmentedAudioOptions = {
  audioRef: RefObject<HTMLAudioElement | null>
  sourceUrl: string | undefined
  byteLength: number | undefined
  durationSeconds: number | undefined
}

function canUseSegmentedMP3() {
  return typeof MediaSource !== 'undefined' && MediaSource.isTypeSupported('audio/mpeg')
}

// The browser's native <audio> loader is allowed to make an open-ended range
// request. This hook owns the requests instead: it appends one 15-second-ish
// MP3 byte window and fetches the next only when buffered playback drops below
// that window. Processed tracks are CBR MP3; for legacy files the file-size
// estimate still supplies a small bounded first request.
export function useSegmentedAudio({ audioRef, sourceUrl, byteLength, durationSeconds }: SegmentedAudioOptions) {
  const [mediaUrl, setMediaUrl] = useState<string>()

  useEffect(() => {
    if (!sourceUrl) {
      setMediaUrl(undefined)
      return
    }
    if (!byteLength || !canUseSegmentedMP3()) {
      setMediaUrl(sourceUrl)
      return
    }

    const mediaSource = new MediaSource()
    const objectUrl = URL.createObjectURL(mediaSource)
    let disposed = false
    let sourceBuffer: SourceBuffer | undefined
    let nextOffset = 0
    let fetching = false

    const audio = audioRef.current
    const segmentBytes = durationSeconds && Number.isFinite(durationSeconds) && durationSeconds > 0
      ? Math.max(64 * 1024, Math.ceil((byteLength / durationSeconds) * segmentDurationSeconds))
      : defaultSegmentBytes

    const bufferedAhead = () => {
      if (!audio || !sourceBuffer) {
        return 0
      }
      for (let index = 0; index < sourceBuffer.buffered.length; index += 1) {
        const start = sourceBuffer.buffered.start(index)
        const end = sourceBuffer.buffered.end(index)
        if (audio.currentTime >= start && audio.currentTime <= end) {
          return end - audio.currentTime
        }
      }
      return 0
    }

    const requestNextSegment = () => {
      if (disposed || !sourceBuffer || fetching || sourceBuffer.updating) {
        return
      }
      if (nextOffset >= byteLength) {
        if (mediaSource.readyState === 'open') {
          mediaSource.endOfStream()
        }
        return
      }
      if (nextOffset > 0 && bufferedAhead() >= segmentDurationSeconds) {
        return
      }

      const start = nextOffset
      const end = Math.min(byteLength - 1, start + segmentBytes - 1)
      fetching = true
      void fetch(sourceUrl, { headers: { Range: `bytes=${start}-${end}` } })
        .then(async (response) => {
          if (!response.ok || response.status !== 206) {
            throw new Error('segment request was not partial content')
          }
          return response.arrayBuffer()
        })
        .then((data) => {
          if (disposed || !sourceBuffer || mediaSource.readyState !== 'open') {
            return
          }
          nextOffset = end + 1
          sourceBuffer.appendBuffer(data)
        })
        .catch(() => {
          if (!disposed && mediaSource.readyState === 'open') {
            mediaSource.endOfStream('network')
          }
        })
        .finally(() => {
          fetching = false
        })
    }

    const onSourceOpen = () => {
      if (disposed) {
        return
      }
      sourceBuffer = mediaSource.addSourceBuffer('audio/mpeg')
      sourceBuffer.addEventListener('updateend', requestNextSegment)
      if (durationSeconds && Number.isFinite(durationSeconds) && durationSeconds > 0) {
        mediaSource.duration = durationSeconds
      }
      requestNextSegment()
    }

    mediaSource.addEventListener('sourceopen', onSourceOpen)
    audio?.addEventListener('timeupdate', requestNextSegment)
    audio?.addEventListener('seeking', requestNextSegment)
    setMediaUrl(objectUrl)

    return () => {
      disposed = true
      mediaSource.removeEventListener('sourceopen', onSourceOpen)
      sourceBuffer?.removeEventListener('updateend', requestNextSegment)
      audio?.removeEventListener('timeupdate', requestNextSegment)
      audio?.removeEventListener('seeking', requestNextSegment)
      URL.revokeObjectURL(objectUrl)
    }
  }, [audioRef, byteLength, durationSeconds, sourceUrl])

  return mediaUrl
}
