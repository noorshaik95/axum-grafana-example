import apiClient from './client'
import type { ServiceHealth, KafkaConsumerLag, PlatformStats } from './types'

export const healthApi = {
  async getServiceHealth(): Promise<ServiceHealth[]> {
    const response = await apiClient.get('/metrics/system')
    // TODO: align types — response shape is SystemMetricsResponse, not ServiceHealth[]
    return response.data as any
  },

  async getPlatformStats(): Promise<PlatformStats> {
    const response = await apiClient.get('/metrics/platform')
    // TODO: align types — response shape is PlatformMetricsResponse, not PlatformStats
    return response.data as any
  },

  async getKafkaLag(): Promise<KafkaConsumerLag[]> {
    // TODO: no Kafka lag RPC in metrics-service
    return Promise.resolve([])
  },
}

export default healthApi
