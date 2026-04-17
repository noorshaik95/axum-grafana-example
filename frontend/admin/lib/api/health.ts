import apiClient from './client'
import type { ServiceHealth, KafkaConsumerLag, PlatformStats } from './types'

export const healthApi = {
  async getServiceHealth(): Promise<ServiceHealth[]> {
    const response = await apiClient.get('/system/health')
    return response.data
  },

  async getPlatformStats(): Promise<PlatformStats> {
    const response = await apiClient.get('/system/stats')
    return response.data
  },

  async getKafkaLag(): Promise<KafkaConsumerLag[]> {
    const response = await apiClient.get('/system/kafka/lag')
    return response.data
  },
}

export default healthApi
