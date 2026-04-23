import * as React from 'react'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { VideoPlayer } from '../../components/video-player'

/**
 * jsdom does not implement media playback, so these tests focus on the props
 * plumbing the wrapper manages: position seeking on mount, sample emission on
 * timeupdate, and emission on pause.
 */

function getVideo(): HTMLVideoElement {
  return screen.getByTestId('video-player') as HTMLVideoElement
}

describe('<VideoPlayer />', () => {
  it('forwards src and poster to the native video element', () => {
    render(<VideoPlayer src="/v.mp4" posterUrl="/p.png" />)
    const v = getVideo()
    expect(v.getAttribute('src')).toBe('/v.mp4')
    expect(v.getAttribute('poster')).toBe('/p.png')
  })

  it('seeks to resumePositionSeconds when video is ready', () => {
    render(<VideoPlayer src="/v.mp4" resumePositionSeconds={42} />)
    const v = getVideo()
    // Simulate the metadata-loaded callback.
    Object.defineProperty(v, 'readyState', { value: 1, configurable: true })
    fireEvent(v, new Event('loadedmetadata'))
    expect(v.currentTime).toBe(42)
  })

  it('emits onPositionChange on the first timeupdate and then every sampleInterval', () => {
    const onPositionChange = jest.fn()
    render(
      <VideoPlayer src="/v.mp4" onPositionChange={onPositionChange} sampleIntervalSeconds={10} />
    )
    const v = getVideo()
    Object.defineProperty(v, 'currentTime', { value: 3, configurable: true, writable: true })
    fireEvent.timeUpdate(v) // first sample always fires
    expect(onPositionChange).toHaveBeenCalledWith(3)
    ;(v as any).currentTime = 7
    fireEvent.timeUpdate(v) // within interval
    expect(onPositionChange).toHaveBeenCalledTimes(1)
    ;(v as any).currentTime = 14
    fireEvent.timeUpdate(v) // 14 - 3 = 11 >= 10 → fires
    expect(onPositionChange).toHaveBeenCalledTimes(2)
    expect(onPositionChange).toHaveBeenLastCalledWith(14)
  })

  it('emits onPositionChange on pause', () => {
    const onPositionChange = jest.fn()
    render(<VideoPlayer src="/v.mp4" onPositionChange={onPositionChange} />)
    const v = getVideo()
    ;(v as any).currentTime = 7
    fireEvent.pause(v)
    expect(onPositionChange).toHaveBeenCalledWith(7)
  })

  it('toggles play/pause when spacebar pressed', () => {
    const play = jest.fn().mockResolvedValue(undefined)
    const pause = jest.fn()
    render(<VideoPlayer src="/v.mp4" />)
    const v = getVideo()
    Object.defineProperty(v, 'paused', { value: true, configurable: true })
    ;(v as any).play = play
    ;(v as any).pause = pause

    fireEvent.keyDown(v, { key: ' ' })
    expect(play).toHaveBeenCalled()

    Object.defineProperty(v, 'paused', { value: false, configurable: true })
    fireEvent.keyDown(v, { key: ' ' })
    expect(pause).toHaveBeenCalled()
  })
})
