import { useEffect, useMemo, useRef, useState } from 'react'
import { mediaApi } from '../../services/api/media'
import type { AudioQuality, MusicTrack } from '../../services/api/types'
import { classNames } from '../../utils/classNames'

type PlayerMode = 'hidden' | 'compact' | 'expanded'

function trackSource(track: MusicTrack | undefined, quality: AudioQuality | undefined) {
  return quality?.sourceUrl || track?.sourceUrl
}

export function MusicPlayer() {
  const audioRef = useRef<HTMLAudioElement>(null)
  const [mode, setMode] = useState<PlayerMode>('compact')
  const [tracks, setTracks] = useState<MusicTrack[]>([])
  const [trackIndex, setTrackIndex] = useState(0)
  const [qualityId, setQualityId] = useState<string>()
  const [isPlaying, setIsPlaying] = useState(false)
  const [isLoading, setIsLoading] = useState(true)

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
  }, [source])

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

  if (mode === 'hidden') {
    return (
      <button className="music-player__launcher" type="button" onClick={() => setMode('compact')} aria-label="打开音乐播放器">
        ♫
      </button>
    )
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
      />
      <button
        className="music-player__cover"
        type="button"
        onClick={() => setMode(mode === 'compact' ? 'expanded' : 'compact')}
        aria-label={mode === 'compact' ? '展开播放器' : '收起播放器'}
      >
        {track?.artworkUrl ? <img src={track.artworkUrl} alt="" /> : <span aria-hidden="true">J</span>}
      </button>

      <div className="music-player__body">
        <div className="music-player__meta">
          <span>{isLoading ? '正在连接音乐库…' : track?.title ?? '音乐库待接入'}</span>
          <small>{track?.artist ?? 'Jiuin Media'}</small>
        </div>
        {mode === 'expanded' && (
          <>
            <div className="music-player__controls" aria-label="播放控制">
              <button type="button" onClick={() => changeTrack(-1)} disabled={tracks.length < 2} aria-label="上一首">‹</button>
              <button type="button" onClick={() => void togglePlayback()} disabled={!source} aria-label={isPlaying ? '暂停' : '播放'}>
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
          </>
        )}
      </div>

      <div className="music-player__actions">
        {mode === 'expanded' && <button type="button" onClick={() => setMode('compact')} aria-label="收起播放器">−</button>}
        <button type="button" onClick={() => setMode('hidden')} aria-label="隐藏播放器">×</button>
      </div>
    </aside>
  )
}
