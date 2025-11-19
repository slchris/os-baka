import { writable } from 'svelte/store'
import { api } from './api'

export interface User {
  id: number
  username: string
  email: string
  full_name?: string
  is_active: boolean
  is_superuser: boolean
}

export interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  isLoading: boolean
}

const createAuthStore = () => {
  const { subscribe, set, update } = writable<AuthState>({
    user: null,
    token: localStorage.getItem('access_token'),
    isAuthenticated: false,
    isLoading: true,
  })

  return {
    subscribe,
    
    // Initialize auth state
    async init() {
      const token = localStorage.getItem('access_token')
      
      if (!token) {
        update(state => ({ ...state, isLoading: false }))
        return false
      }
      
      try {
        const response = await api.get('/auth/me')
        update(state => ({
          ...state,
          user: response.data,
          token,
          isAuthenticated: true,
          isLoading: false,
        }))
        return true
      } catch (error) {
        localStorage.removeItem('access_token')
        update(state => ({
          ...state,
          user: null,
          token: null,
          isAuthenticated: false,
          isLoading: false,
        }))
        return false
      }
    },
    
    // Login
    async login(username: string, password: string) {
      try {
        const response = await api.post('/auth/login', { username, password })
        const token = response.data.access_token
        
        localStorage.setItem('access_token', token)
        
        // Fetch user info
        const userResponse = await api.get('/auth/me')
        
        update(state => ({
          ...state,
          user: userResponse.data,
          token,
          isAuthenticated: true,
        }))
        
        return { success: true }
      } catch (error: any) {
        return {
          success: false,
          error: error.response?.data?.detail || '登录失败'
        }
      }
    },
    
    // Logout
    logout() {
      localStorage.removeItem('access_token')
      set({
        user: null,
        token: null,
        isAuthenticated: false,
        isLoading: false,
      })
    },
  }
}

export const authStore = createAuthStore()
