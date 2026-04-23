import * as React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { RubricEditor, type RubricRow } from '../../components/rubric-editor'

function seed(): RubricRow[] {
  return [
    { id: 'a', title: 'Clarity', maxPoints: 10, sortOrder: 0 },
    { id: 'b', title: 'Accuracy', maxPoints: 15, sortOrder: 1 },
  ]
}

describe('<RubricEditor />', () => {
  it('renders existing rows as editable inputs', () => {
    render(<RubricEditor rows={seed()} onChange={() => {}} />)
    expect(screen.getByDisplayValue('Clarity')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Accuracy')).toBeInTheDocument()
  })

  it('adds a new row with sortOrder set correctly', () => {
    const onChange = jest.fn()
    render(<RubricEditor rows={seed()} onChange={onChange} />)
    fireEvent.click(screen.getByRole('button', { name: '+ Add row' }))
    const next = onChange.mock.calls[0][0] as RubricRow[]
    expect(next).toHaveLength(3)
    expect(next[2].sortOrder).toBe(2)
  })

  it('removes a row and renumbers sortOrder', () => {
    const onChange = jest.fn()
    render(<RubricEditor rows={seed()} onChange={onChange} />)
    fireEvent.click(screen.getByRole('button', { name: 'Remove row 1' }))
    const next = onChange.mock.calls[0][0] as RubricRow[]
    expect(next).toHaveLength(1)
    expect(next[0].sortOrder).toBe(0)
    expect(next[0].id).toBe('b')
  })

  it('updates a row title', () => {
    const onChange = jest.fn()
    render(<RubricEditor rows={seed()} onChange={onChange} />)
    fireEvent.change(screen.getByDisplayValue('Clarity'), {
      target: { value: 'Clarity!' },
    })
    const next = onChange.mock.calls[0][0] as RubricRow[]
    expect(next[0].title).toBe('Clarity!')
  })

  it('swaps rows with the down button', () => {
    const onChange = jest.fn()
    render(<RubricEditor rows={seed()} onChange={onChange} />)
    fireEvent.click(screen.getByRole('button', { name: 'Move row 1 down' }))
    const next = onChange.mock.calls[0][0] as RubricRow[]
    expect(next[0].id).toBe('b')
    expect(next[1].id).toBe('a')
    expect(next[0].sortOrder).toBe(0)
    expect(next[1].sortOrder).toBe(1)
  })

  it('disables controls when editable=false', () => {
    render(<RubricEditor rows={seed()} onChange={() => {}} editable={false} />)
    expect(screen.queryByRole('button', { name: '+ Add row' })).toBeNull()
    expect(screen.getByDisplayValue('Clarity')).toBeDisabled()
  })

  it('renders an empty placeholder when rows is empty', () => {
    render(<RubricEditor rows={[]} onChange={() => {}} />)
    expect(screen.getByText(/No rubric rows yet/i)).toBeInTheDocument()
  })
})
