'use client'

import * as React from 'react'
import { cn } from '../utils/index'

export interface VideoPlayerProps {
  src: string
  posterUrl?: string
  /** Seek to this position (seconds) on mount. */
  resumePositionSeconds?: number
  /**
   * Called every ~10 seconds of playback (sampled from the native timeupdate event)
   * and on pause. Consumers wire this to `PUT /api/content/:id/position`.
   */
  onPositionChange?: (seconds: number) => void
  /** Sample interval in seconds. Defaults to 10. */
  sampleIntervalSeconds?: number
  className?: string
  /** Forwarded autoplay attribute; browsers may ignore unless muted. */
  autoPlay?: boolean
  /** Track list passed through to native <video><track>. */
  tracks?: Array<{
    src: string
    kind?: 'subtitles' | 'captions' | 'descriptions' | 'chapters' | 'metadata'
    srcLang?: string
    label?: string
    default?: boolean
  }>
}

/**
 * Native HTML5 <video> wrapper for Slate. Seeks to `resumePositionSeconds` on
 * mount and emits `onPositionChange(sec)` every `sampleIntervalSeconds` of
 * playback. Does NOT bundle HLS.js — browsers that support HLS natively
 * (Safari) will play m3u8 URLs; other formats play via whatever codec the
 * browser supports (MP4 is the expected default).
 */
export const VideoPlayer = React.forwardRef<HTMLVideoElement, VideoPlayerProps>(
  function VideoPlayer(
    {
      src,
      posterUrl,
      resumePositionSeconds = 0,
      onPositionChange,
      sampleIntervalSeconds = 10,
      className,
      autoPlay,
      tracks,
    },
    forwardedRef
  ) {
    const internalRef = React.useRef<HTMLVideoElement | null>(null)
    const lastSampledRef = React.useRef<number>(-Infinity)

    const setRefs = React.useCallback(
      (node: HTMLVideoElement | null) => {
        internalRef.current = node
        if (typeof forwardedRef === 'function') forwardedRef(node)
        else if (forwardedRef)
          (forwardedRef as React.MutableRefObject<HTMLVideoElement | null>).current = node
      },
      [forwardedRef]
    )

    React.useEffect(() => {
      const el = internalRef.current
      if (!el) return
      if (resumePositionSeconds > 0) {
        const onReady = () => {
          try {
            el.currentTime = resumePositionSeconds
          } catch {
            // seek may fail on some MSE sources; silently ignore.
          }
          el.removeEventListener('loadedmetadata', onReady)
        }
        if (el.readyState >= 1) onReady()
        else el.addEventListener('loadedmetadata', onReady)
        return () => el.removeEventListener('loadedmetadata', onReady)
      }
      return
      // only run on mount / when the src/resume position meaningfully changes
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [src, resumePositionSeconds])

    const handleTimeUpdate = (e: React.SyntheticEvent<HTMLVideoElement>) => {
      const t = e.currentTarget.currentTime
      if (t - lastSampledRef.current >= sampleIntervalSeconds) {
        lastSampledRef.current = t
        onPositionChange?.(t)
      }
    }

    const handlePause = (e: React.SyntheticEvent<HTMLVideoElement>) => {
      onPositionChange?.(e.currentTarget.currentTime)
    }

    const handleKeyDown = (e: React.KeyboardEvent<HTMLVideoElement>) => {
      const el = internalRef.current
      if (!el) return
      if (e.key === ' ' || e.key === 'k') {
        e.preventDefault()
        if (el.paused) void el.play()
        else el.pause()
      } else if (e.key === 'f') {
        e.preventDefault()
        if (document.fullscreenElement) void document.exitFullscreen()
        else void el.requestFullscreen?.()
      }
    }

    return (
      <video
        ref={setRefs}
        data-testid="video-player"
        className={cn(
          'w-full rounded-card bg-black outline-none focus-visible:ring-2 focus-visible:ring-forest-500',
          className
        )}
        src={src}
        poster={posterUrl}
        controls
        playsInline
        autoPlay={autoPlay}
        tabIndex={0}
        onTimeUpdate={handleTimeUpdate}
        onPause={handlePause}
        onKeyDown={handleKeyDown}
      >
        {tracks?.map((t, i) => (
          <track
            key={`${t.src}-${i}`}
            src={t.src}
            kind={t.kind}
            srcLang={t.srcLang}
            label={t.label}
            default={t.default}
          />
        ))}
      </video>
    )
  }
)

VideoPlayer.displayName = 'VideoPlayer'
