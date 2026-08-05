import client from './client'

export interface NodeInfo {
  id: string
  name: string
  hostname: string
  ip: string
  os: string
  arch: string
  agent_version: string
  status: 'online' | 'offline'
  cpu_usage: number
  mem_usage: number
  last_seen: string
  service_count: number
}

export interface SubnetScanResult {
  subnet: string
  scanned_ips: number
  scanned_ports: number
  discovered: Array<{
    ip: string
    port: number
    reachable: boolean
    latency: number
    service: string
  }>
  total: number
}

export const nodeApi = {
  list() {
    return client.get<any, NodeInfo[]>('/nodes')
  },
  scanSubnet(subnet: string, ports?: number[]) {
    return client.post<any, SubnetScanResult>('/nodes/scan-subnet', { subnet, ports: ports || [] })
  },
}
