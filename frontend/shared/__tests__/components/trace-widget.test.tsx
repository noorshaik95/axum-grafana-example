import * as React from 'react'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { TraceWidget } from '../../components/trace-widget'
import { traceStore } from '../../lib/trace/store'

describe('<TraceWidget />', () => {
  beforeEach(() => traceStore.clear())

  it('renders nothing when no request has been recorded yet', () => {
    const { container } = render(<TraceWidget enabled />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders nothing when explicitly disabled (production build)', () => {
    traceStore.record({
      requestId: 'req-abc',
      traceparent: '00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01',
      route: 'GET /api/test',
      status: 200,
      ts: Date.now(),
      durationMs: 3,
    })
    const { container } = render(<TraceWidget enabled={false} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('surfaces the current request id and status tone', () => {
    act(() => {
      traceStore.record({
        requestId: 'req-abcdef1234',
        traceparent: '00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01',
        route: 'GET /api/test',
        status: 500,
        ts: Date.now(),
        durationMs: 42,
      })
    })
    render(<TraceWidget enabled />)

    expect(screen.getByTestId('trace-widget')).toBeInTheDocument()
    expect(screen.getByText('500')).toBeInTheDocument()
    // id appears inside a <span>; truncated to the first 8 chars + ellipsis
    expect(screen.getByText(/^req-abcd/)).toBeInTheDocument()
  })

  it('links to Grafana Tempo when the traceparent is valid', () => {
    act(() => {
      traceStore.record({
        requestId: 'req-1',
        traceparent: '00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01',
        route: 'GET /api/test',
        status: 200,
        ts: Date.now(),
        durationMs: 10,
      })
    })
    render(<TraceWidget enabled grafanaUrl="http://localhost:3200" />)

    const link = screen.getByRole('link', { name: /tempo/i })
    expect(link).toHaveAttribute(
      'href',
      expect.stringContaining('4bf92f3577b34da6a3ce929d0e0e4736')
    )
  })

  it('omits the Tempo link when traceparent is invalid', () => {
    act(() => {
      traceStore.record({
        requestId: 'req-1',
        traceparent: null,
        route: 'GET /api/test',
        status: 200,
        ts: Date.now(),
        durationMs: 10,
      })
    })
    render(<TraceWidget enabled />)
    expect(screen.queryByRole('link', { name: /tempo/i })).toBeNull()
  })

  it('expands history when the + counter is clicked', () => {
    act(() => {
      traceStore.record({
        requestId: 'req-1',
        traceparent: null,
        route: 'GET /api/first',
        status: 200,
        ts: Date.now(),
        durationMs: 10,
      })
      traceStore.record({
        requestId: 'req-2',
        traceparent: null,
        route: 'GET /api/second',
        status: 200,
        ts: Date.now(),
        durationMs: 20,
      })
    })
    render(<TraceWidget enabled />)

    const expander = screen.getByRole('button', { name: /\+2/ })
    fireEvent.click(expander)
    expect(screen.getByText('Recent requests')).toBeInTheDocument()
    expect(screen.getByText('GET /api/first')).toBeInTheDocument()
    expect(screen.getByText('GET /api/second')).toBeInTheDocument()
  })
})
