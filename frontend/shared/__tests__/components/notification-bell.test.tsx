import * as React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { NotificationBell } from '../../components/notification-bell'

describe('<NotificationBell />', () => {
  it('renders with a11y label when no unread', () => {
    render(<NotificationBell unreadCount={0} />)
    expect(screen.getByTestId('notification-bell')).toHaveAttribute('aria-label', 'Notifications')
    expect(screen.queryByTestId('notification-bell-badge')).toBeNull()
  })

  it('renders badge count when unread > 0', () => {
    render(<NotificationBell unreadCount={3} />)
    const badge = screen.getByTestId('notification-bell-badge')
    expect(badge).toHaveTextContent('3')
    expect(badge.className).toMatch(/bg-forest-600/)
  })

  it('clamps to "99+" by default when count exceeds max', () => {
    render(<NotificationBell unreadCount={1000} />)
    expect(screen.getByTestId('notification-bell-badge')).toHaveTextContent('99+')
  })

  it('honors custom max', () => {
    render(<NotificationBell unreadCount={20} max={9} />)
    expect(screen.getByTestId('notification-bell-badge')).toHaveTextContent('9+')
  })

  it('invokes onClick when pressed', () => {
    const onClick = jest.fn()
    render(<NotificationBell unreadCount={1} onClick={onClick} />)
    fireEvent.click(screen.getByTestId('notification-bell'))
    expect(onClick).toHaveBeenCalledTimes(1)
  })
})
