import client from './client'

export interface LoginInput {
  username: string
  password: string
}

export interface LoginResult {
  token: string
  username: string
  role: string
  tenant_id: string
  expires_in: number
}

export const authApi = {
  login: (data: LoginInput) => client.post('/auth/login', data),
  getProfile: () => client.get('/auth/profile'),
}
