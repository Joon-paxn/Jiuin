import { useEffect, useMemo, useRef, useState } from 'react'
import { mediaApi } from '../../services/api/media'
import type { AudioQuality, MusicTrack } from '../../services/api/types'
import { classNames } from '../../utils/classNames'

type PlayerMode = 'hidden' | 'album' | 'full'

function trackSource(track: MusicTrack | undefined, quality: AudioQuality | undefined) {
  return quality?.sourceUrl || track?.sourceUrl
}

function toSafeDuration(value: number | undefined) {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : 0
}

function formatTime(value: number) {
  const wholeSeconds = Math.max(0, Math.floor(value))
  const minutes = Math.floor(wholeSeconds / 60)
  const seconds = String(wholeSeconds % 60).padStart(2, '0')
  return `${minutes}:${seconds}`
}

export function MusicPlayer() {
  const audioRef = useRef<HTMLAudioElement>(null)
  const [mode, setMode] = useState<PlayerMode>('album')
  const [tracks, setTracks] = useState<MusicTrack[]>([])
  const [trackIndex, setTrackIndex] = useState(0)
  const [qualityId, setQualityId] = useState<string>()
  const [isPlaying, setIsPlaying] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [currentTime, setCurrentTime] = useState(0)
  const [duration, setDuration] = useState(0)

  const track = tracks[trackIndex]
  const quality = useMemo(
    () => track?.qualities?.find((item) => item.id === qualityId) ?? track?.qualities?.[0],
    [qualityId, track],
  )
  const source = trackSource(track, quality)

  useEffect(() => {
    let active = true

    void mediaApi.listMusic()
      .then((items) => {
        if (active) {
          setTracks(items)
        }
      })
      .catch(() => {
        // The site remains usable without an API during local visual development.
      })
      .finally(() => {
        if (active) {
          setIsLoading(false)
        }
      })

    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    setQualityId(track?.qualities?.[0]?.id)
    setIsPlaying(false)
  }, [track?.id])

  useEffect(() => {
    const audio = audioRef.current
    if (!audio) {
      return
    }

    audio.pause()
    audio.load()
    setIsPlaying(false)
    setCurrentTime(0)
    setDuration(toSafeDuration(track?.durationSeconds))
  }, [source, track?.durationSeconds])

  const togglePlayback = async () => {
    const audio = audioRef.current
    if (!audio || !source) {
      return
    }

    if (audio.paused) {
      try {
        await audio.play()
        setIsPlaying(true)
      } catch {
        setIsPlaying(false)
      }
      return
    }

    audio.pause()
    setIsPlaying(false)
  }

  const changeTrack = (direction: -1 | 1) => {
    if (tracks.length < 2) {
      return
    }
    setTrackIndex((index) => (index + direction + tracks.length) % tracks.length)
  }

  const hidePlayer = () => {
    audioRef.current?.pause()
    setIsPlaying(false)
    setMode('hidden')
  }

  const seekTo = (nextTime: number) => {
    const audio = audioRef.current
    if (!audio || !duration) {
      return
    }

    const targetTime = Math.min(duration, Math.max(0, nextTime))
    audio.currentTime = targetTime
    setCurrentTime(targetTime)
  }

  if (mode === 'hidden') {
    return null
  }

  return (
    <aside className={classNames('music-player', `music-player--${mode}`)} aria-label="音乐播放器">
      <audio
        ref={audioRef}
        preload="none"
        src={source}
        onEnded={() => {
          setIsPlaying(false)
          changeTrack(1)
        }}
        onLoadedMetadata={(event) => setDuration(toSafeDuration(event.currentTarget.duration) || toSafeDuration(track?.durationSeconds))}
        onDurationChange={(event) => setDuration(toSafeDuration(event.currentTarget.duration) || toSafeDuration(track?.durationSeconds))}
        onTimeUpdate={(event) => setCurrentTime(event.currentTarget.currentTime)}
        onPlay={() => setIsPlaying(true)}
        onPause={() => setIsPlaying(false)}
      />
      <button
        className="music-player__cover"
        type="button"
        onClick={() => setMode(mode === 'album' ? 'full' : 'album')}
        aria-label={mode === 'album' ? '展开完整播放器' : '收起至专辑模式'}
      >
        {track?.artworkUrl ? <img src={track.artworkUrl} alt="" /> : <span aria-hidden="true">J</span>}
      </button>

      {mode === 'album' && (
        <button className="music-player__expand" type="button" onClick={() => setMode('full')} aria-label="展开完整播放器">
          <span aria-hidden="true">⌃</span>
          <span>播放</span>
        </button>
      )}

      {mode === 'full' && (
        <div className="music-player__body">
          <div className="music-player__meta">
            <span>{isLoading ? '正在连接音乐库…' : track?.title ?? '音乐库待接入'}</span>
            <small>{track?.artist ?? 'Jiuin Media'}</small>
          </div>

          <div className="music-player__actions">
            <button type="button" onClick={() => setMode('album')} aria-label="收起至专辑模式">−</button>
            <button type="button" onClick={hidePlayer} aria-label="隐藏播放器">×</button>
          </div>

          <label className="music-player__progress">
            <span className="music-player__time">{formatTime(currentTime)}</span>
            <input
              type="range"
              min="0"
              max={duration || 0}
              step="0.1"
              value={Math.min(currentTime, duration || 0)}
              onChange={(event) => seekTo(Number(event.target.value))}
              disabled={!duration || !source}
              aria-label="播放进度"
            />
            <span className="music-player__time">{formatTime(duration)}</span>
          </label>

          <div className="music-player__controls" aria-label="播放控制">
            <button type="button" onClick={() => changeTrack(-1)} disabled={tracks.length < 2} aria-label="上一首">‹</button>
            <button className="music-player__play-toggle" type="button" onClick={() => void togglePlayback()} disabled={!source} aria-label={isPlaying ? '暂停' : '播放'}>
              {isPlaying ? 'Ⅱ' : '▶'}
            </button>
            <button type="button" onClick={() => changeTrack(1)} disabled={tracks.length < 2} aria-label="下一首">›</button>
          </div>

          <label className="music-player__quality">
            <span>音质</span>
            <select value={quality?.id ?? ''} onChange={(event) => setQualityId(event.target.value)} disabled={!track?.qualities?.length}>
              {track?.qualities?.map((item) => <option key={item.id} value={item.id}>{item.label}</option>) ?? <option>待提供</option>}
            </select>
          </label>
        </div>
      )}
    </aside>
  )
}
