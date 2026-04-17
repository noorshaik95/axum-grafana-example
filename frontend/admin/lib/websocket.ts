import { io, Socket } from 'socket.io-client'

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080'

class WebSocketClient {
  private socket: Socket | null = null
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5

  connect(token: string): Socket {
    if (this.socket?.connected) {
      return this.socket
    }

    this.socket = io(WS_URL, {
      auth: {
        token,
      },
      transports: ['websocket', 'polling'],
      reconnection: true,
      reconnectionDelay: 1000,
      reconnectionDelayMax: 5000,
      reconnectionAttempts: this.maxReconnectAttempts,
    })

    this.socket.on('connect', () => {
      console.log('WebSocket connected')
      this.reconnectAttempts = 0
    })

    this.socket.on('disconnect', (reason) => {
      console.log('WebSocket disconnected:', reason)
    })

    this.socket.on('connect_error', (error) => {
      console.error('WebSocket connection error:', error)
      this.reconnectAttempts++
      if (this.reconnectAttempts >= this.maxReconnectAttempts) {
        console.error('Max reconnection attempts reached')
        this.disconnect()
      }
    })

    return this.socket
  }

  disconnect(): void {
    if (this.socket) {
      this.socket.disconnect()
      this.socket = null
    }
  }

  subscribeToJobProgress(
    jobId: string,
    callback: (data: {
      job_id: string
      status: string
      progress: number
      processed: number
      total: number
      errors?: string[]
    }) => void
  ): void {
    if (!this.socket) {
      console.error('WebSocket not connected')
      return
    }

    this.socket.emit('subscribe:job', { job_id: jobId })
    this.socket.on(`job:progress:${jobId}`, callback)
  }

  unsubscribeFromJobProgress(jobId: string): void {
    if (!this.socket) return

    this.socket.emit('unsubscribe:job', { job_id: jobId })
    this.socket.off(`job:progress:${jobId}`)
  }

  subscribeToUsageMetrics(
    callback: (data: {
      timestamp: string
      metrics: {
        active_users: number
        storage_used: number
        bandwidth_used: number
        ai_credits_used: number
      }
    }) => void
  ): void {
    if (!this.socket) {
      console.error('WebSocket not connected')
      return
    }

    this.socket.emit('subscribe:metrics')
    this.socket.on('metrics:update', callback)
  }

  unsubscribeFromUsageMetrics(): void {
    if (!this.socket) return

    this.socket.emit('unsubscribe:metrics')
    this.socket.off('metrics:update')
  }

  isConnected(): boolean {
    return this.socket?.connected || false
  }
}

export const websocketClient = new WebSocketClient()
export default websocketClient
