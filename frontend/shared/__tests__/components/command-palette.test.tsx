import * as React from 'react'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { CommandPalette, type CommandPaletteResult } from '../../components/command-palette'

const sample: CommandPaletteResult[] = [
  { id: '1', label: 'CS 101', route: '/courses/cs-101', group: 'Courses' },
  { id: '2', label: 'CS 201', route: '/courses/cs-201', group: 'Courses' },
  { id: '3', label: 'Alice', route: '/people/alice', group: 'People' },
]

describe('<CommandPalette />', () => {
  it('is hidden when not open', () => {
    render(<CommandPalette onSearch={async () => []} open={false} />)
    expect(screen.queryByTestId('command-palette')).toBeNull()
  })

  it('renders when open and focuses the input', async () => {
    render(<CommandPalette onSearch={async () => []} open />)
    expect(screen.getByTestId('command-palette')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByTestId('command-palette-input')).toHaveFocus()
    })
  })

  it('debounces and calls onSearch with the query', async () => {
    const onSearch = jest.fn(async (q: string) => sample.filter((r) => r.label.includes(q)))
    render(<CommandPalette onSearch={onSearch} open debounceMs={10} />)

    const input = screen.getByTestId('command-palette-input')
    fireEvent.change(input, { target: { value: 'CS' } })

    await waitFor(() => expect(onSearch).toHaveBeenCalledWith('CS'))
    await waitFor(() => expect(screen.getByText('CS 101')).toBeInTheDocument())
  })

  it('navigates results with arrow keys and selects with Enter', async () => {
    const onSelect = jest.fn()
    render(<CommandPalette onSearch={async () => sample} onSelect={onSelect} open debounceMs={0} />)
    const input = screen.getByTestId('command-palette-input')
    fireEvent.change(input, { target: { value: 'a' } })
    await waitFor(() => expect(screen.getByText('CS 101')).toBeInTheDocument())

    fireEvent.keyDown(input, { key: 'ArrowDown' })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onSelect).toHaveBeenCalledWith(sample[1])
  })

  it('dismisses on Escape', async () => {
    const onOpenChange = jest.fn()
    render(
      <CommandPalette onSearch={async () => []} open onOpenChange={onOpenChange} debounceMs={0} />
    )
    fireEvent.keyDown(screen.getByTestId('command-palette-input'), { key: 'Escape' })
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('falls back to routerPush when no onSelect given', async () => {
    const routerPush = jest.fn()
    render(
      <CommandPalette onSearch={async () => sample} routerPush={routerPush} open debounceMs={0} />
    )
    const input = screen.getByTestId('command-palette-input')
    fireEvent.change(input, { target: { value: 'a' } })
    await waitFor(() => expect(screen.getByText('CS 101')).toBeInTheDocument())
    fireEvent.click(screen.getByText('CS 101'))
    expect(routerPush).toHaveBeenCalledWith('/courses/cs-101')
  })

  it('opens on ⌘K when uncontrolled', () => {
    render(<CommandPalette onSearch={async () => []} />)
    expect(screen.queryByTestId('command-palette')).toBeNull()
    act(() => {
      const e = new KeyboardEvent('keydown', { key: 'k', metaKey: true })
      window.dispatchEvent(e)
    })
    expect(screen.getByTestId('command-palette')).toBeInTheDocument()
  })
})
