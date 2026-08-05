import client from './client'

export interface LoginParams {
  username: string
  password: string
}

export interface UserInfo {
  id: number
  username: string
  email: string
  phone: string
  status: number
  created_at: string
  updated_at: string
}

export interface CreateUserParams {
  username: string
  email: string
  phone: string
  password: string
  status?: number
}

export interface UpdateUserParams {
  email?: string
  phone?: string
  status?: number
}

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export const userApi = {
  login(params: LoginParams) {
    return client.post<any, { token: string; user: UserInfo }>('/auth/login', params)
  },

  listUsers(params: { page?: number; page_size?: number; username?: string; status?: number }) {
    return client.get<any, PageResult<UserInfo>>('/users', { params })
  },

  createUser(params: CreateUserParams) {
    return client.post<any, UserInfo>('/users', params)
  },

  updateUser(id: number, params: UpdateUserParams) {
    return client.put<any, UserInfo>(`/users/${id}`, params)
  },

  deleteUser(id: number) {
    return client.delete(`/users/${id}`)
  },
}
