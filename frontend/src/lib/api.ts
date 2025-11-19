import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8000/api/v1'

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor - add auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('access_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor - handle 401 errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Clear token and redirect to login
      localStorage.removeItem('access_token')
      if (!window.location.pathname.includes('/login')) {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

export interface Asset {
  id: number
  hostname: string
  ip_address: string
  mac_address: string
  asset_tag: string
  usb_key?: string
  status: 'active' | 'inactive' | 'installing' | 'maintenance' | 'error'
  os_type?: 'macos' | 'linux' | 'windows'
  encryption_enabled: boolean
  last_seen?: string
  created_at: string
  updated_at?: string
}

export interface AssetCreate {
  hostname: string
  ip_address: string
  mac_address: string
  asset_tag: string
  usb_key?: string
  status?: string
  os_type?: string
  encryption_enabled?: boolean
}

export interface AssetUpdate extends Partial<AssetCreate> {}

export interface AssetListResponse {
  total: number
  items: Asset[]
}

// Asset API functions
export const assetApi = {
  // List assets with pagination and filtering
  list: async (params?: {
    skip?: number
    limit?: number
    status?: string
  }): Promise<AssetListResponse> => {
    const response = await api.get('/assets', { params })
    return response.data
  },

  // Get single asset by ID
  get: async (id: number): Promise<Asset> => {
    const response = await api.get(`/assets/${id}`)
    return response.data
  },

  // Create new asset
  create: async (data: AssetCreate): Promise<Asset> => {
    const response = await api.post('/assets', data)
    return response.data
  },

  // Update existing asset
  update: async (id: number, data: AssetUpdate): Promise<Asset> => {
    const response = await api.put(`/assets/${id}`, data)
    return response.data
  },

  // Delete asset
  delete: async (id: number): Promise<void> => {
    await api.delete(`/assets/${id}`)
  },
}

// PXE Configuration interfaces
export interface PXEConfig {
  id: number
  hostname: string
  mac_address: string
  ip_address: string
  netmask: string
  gateway?: string
  dns_servers?: string
  boot_image?: string
  boot_params?: string
  os_type: string
  enabled: boolean
  deployed: boolean
  last_boot?: string
  description?: string
  created_at: string
  updated_at: string
  asset_id?: number
}

export interface PXEConfigCreate {
  hostname: string
  mac_address: string
  ip_address: string
  netmask?: string
  gateway?: string
  dns_servers?: string
  boot_image?: string
  boot_params?: string
  os_type?: string
  enabled?: boolean
  description?: string
  asset_id?: number
}

export interface PXEConfigUpdate extends Partial<PXEConfigCreate> {}

export interface PXEConfigListResponse {
  items: PXEConfig[]
  total: number
  page: number
  page_size: number
}

export interface PXEDeployment {
  id: number
  pxe_config_id: number
  started_at: string
  completed_at?: string
  status: string
  log_output?: string
  error_message?: string
  initiated_by?: number
}

export interface ServiceStatus {
  service: string
  running: boolean
  status: string
  output: string
}

// PXE API functions
export const pxeApi = {
  // List PXE configurations
  list: async (params?: {
    skip?: number
    limit?: number
  }): Promise<PXEConfigListResponse> => {
    const response = await api.get('/pxe', { params })
    return response.data
  },

  // Get single PXE configuration
  get: async (id: number): Promise<PXEConfig> => {
    const response = await api.get(`/pxe/${id}`)
    return response.data
  },

  // Create new PXE configuration
  create: async (data: PXEConfigCreate): Promise<PXEConfig> => {
    const response = await api.post('/pxe', data)
    return response.data
  },

  // Update PXE configuration
  update: async (id: number, data: PXEConfigUpdate): Promise<PXEConfig> => {
    const response = await api.put(`/pxe/${id}`, data)
    return response.data
  },

  // Delete PXE configuration
  delete: async (id: number): Promise<void> => {
    await api.delete(`/pxe/${id}`)
  },

  // Generate all dnsmasq configs
  generateAll: async (): Promise<{ success: boolean; message: string; details: string[] }> => {
    const response = await api.post('/pxe/generate-all')
    return response.data
  },

  // Validate dnsmasq configuration
  validate: async (): Promise<{ valid: boolean; errors: string[] }> => {
    const response = await api.post('/pxe/validate')
    return response.data
  },

  // Restart dnsmasq service
  restartService: async (): Promise<{ success: boolean; message: string }> => {
    const response = await api.post('/pxe/restart-service')
    return response.data
  },

  // Get service status
  getServiceStatus: async (): Promise<ServiceStatus> => {
    const response = await api.get('/pxe/service-status')
    return response.data
  },

  // List deployments for a config
  listDeployments: async (configId: number): Promise<PXEDeployment[]> => {
    const response = await api.get(`/pxe/${configId}/deployments`)
    return response.data
  },

  // Create deployment
  createDeployment: async (configId: number): Promise<PXEDeployment> => {
    const response = await api.post(`/pxe/${configId}/deploy`)
    return response.data
  },
}

// Deployment interfaces
export interface DeploymentStats {
  total: number
  pending: number
  in_progress: number
  completed: number
  failed: number
}

// Deployment API functions
export const deploymentApi = {
  // List deployments
  list: async (params?: {
    skip?: number
    limit?: number
    status?: string
  }): Promise<PXEDeployment[]> => {
    const response = await api.get('/deployments', { params })
    return response.data
  },

  // Get deployment by ID
  get: async (id: number): Promise<PXEDeployment> => {
    const response = await api.get(`/deployments/${id}`)
    return response.data
  },

  // Get deployment statistics
  getStats: async (): Promise<DeploymentStats> => {
    const response = await api.get('/deployments/stats/summary')
    return response.data
  },
}

// Settings interfaces
export interface SystemSetting {
  id: number
  key: string
  value: string | null
  value_type: string
  category: string
  description: string | null
  is_public: boolean
  created_at: string
  updated_at: string
}

export interface SettingCreate {
  key: string
  value?: string
  value_type?: string
  category?: string
  description?: string
  is_public?: boolean
}

export interface SettingUpdate {
  value?: string
  description?: string
  is_public?: boolean
}

// Settings API functions
export const settingsApi = {
  // List settings
  list: async (params?: {
    category?: string
    skip?: number
    limit?: number
  }): Promise<{ items: SystemSetting[]; total: number }> => {
    const response = await api.get('/settings', { params })
    return response.data
  },

  // Get setting by key
  get: async (key: string): Promise<SystemSetting> => {
    const response = await api.get(`/settings/${key}`)
    return response.data
  },

  // Create setting
  create: async (data: SettingCreate): Promise<SystemSetting> => {
    const response = await api.post('/settings', data)
    return response.data
  },

  // Update setting
  update: async (key: string, data: SettingUpdate): Promise<SystemSetting> => {
    const response = await api.put(`/settings/${key}`, data)
    return response.data
  },

  // Delete setting
  delete: async (key: string): Promise<void> => {
    await api.delete(`/settings/${key}`)
  },

  // List categories
  listCategories: async (): Promise<string[]> => {
    const response = await api.get('/settings/categories/list')
    return response.data
  },
}

// PXE Service interfaces (Singleton Configuration)
export interface PXEServiceConfig {
  id: number
  server_ip: string
  dhcp_range_start: string
  dhcp_range_end: string
  netmask: string
  gateway?: string
  dns_servers?: string
  tftp_root: string
  enable_bios: boolean
  enable_uefi: boolean
  bios_boot_file: string
  uefi_boot_file: string
  os_type: string
  debian_version: string
  debian_mirror: string
  enable_encrypted: boolean
  enable_unencrypted: boolean
  default_encryption: boolean
  default_username: string
  container_name: string
  container_image: string
  container_status: string
  container_id?: string
  service_enabled: boolean
  last_started?: string
  last_stopped?: string
  created_at: string
  updated_at: string
}

export interface PXEServiceConfigUpdate {
  server_ip?: string
  dhcp_range_start?: string
  dhcp_range_end?: string
  netmask?: string
  gateway?: string
  dns_servers?: string
  tftp_root?: string
  enable_bios?: boolean
  enable_uefi?: boolean
  bios_boot_file?: string
  uefi_boot_file?: string
  debian_version?: string
  debian_mirror?: string
  enable_encrypted?: boolean
  enable_unencrypted?: boolean
  default_encryption?: boolean
  luks_password?: string
  preseed_template?: string
  root_password?: string
  default_username?: string
  default_user_password?: string
  container_image?: string
}

export interface PXEServiceStatus {
  status: string
  container_exists: boolean
  container_running: boolean
  enabled: boolean
  last_started?: string
  last_stopped?: string
}

export interface ServiceActionResponse {
  success: boolean
  message: string
  status: string
}

export interface LogsResponse {
  success: boolean
  logs: string
  message?: string
}

// PXE Service API functions
export const pxeServiceApi = {
  // Get configuration (singleton)
  getConfig: async (): Promise<PXEServiceConfig> => {
    const response = await api.get('/pxe-service/config')
    return response.data
  },

  // Update configuration (admin only)
  updateConfig: async (data: PXEServiceConfigUpdate): Promise<PXEServiceConfig> => {
    const response = await api.put('/pxe-service/config', data)
    return response.data
  },

  // Get service status
  getStatus: async (): Promise<PXEServiceStatus> => {
    const response = await api.get('/pxe-service/status')
    return response.data
  },

  // Start service (admin only)
  start: async (): Promise<ServiceActionResponse> => {
    const response = await api.post('/pxe-service/start')
    return response.data
  },

  // Stop service (admin only)
  stop: async (): Promise<ServiceActionResponse> => {
    const response = await api.post('/pxe-service/stop')
    return response.data
  },

  // Restart service (admin only)
  restart: async (): Promise<ServiceActionResponse> => {
    const response = await api.post('/pxe-service/restart')
    return response.data
  },

  // Get service logs
  getLogs: async (tail: number = 100): Promise<LogsResponse> => {
    const response = await api.get('/pxe-service/logs', {
      params: { tail }
    })
    return response.data
  },

  // Remove container (admin only)
  removeContainer: async (): Promise<ServiceActionResponse> => {
    const response = await api.delete('/pxe-service/container')
    return response.data
  },
}

// USB Key interfaces
export interface USBKey {
  id: number
  uuid: string
  label: string
  serial_number?: string
  status: string
  is_initialized: boolean
  asset_id?: number
  bound_at?: string
  last_used_at?: string
  use_count: number
  failed_attempts: number
  backup_count: number
  last_backup_at?: string
  description?: string
  created_at: string
  updated_at: string
}

export interface USBKeyCreate {
  label: string
  description?: string
}

export interface USBKeyInitResponse {
  usb_key: USBKey
  key_material: string
  recovery_code: string
  uuid: string
  warning: string
}

export interface BackupResponse {
  backup_id: number
  backup_uuid: string
  backup_password: string
  checksum: string
  created_at: string
  expires_at?: string
  warning: string
}

export interface USBKeyBackup {
  id: number
  backup_uuid: string
  backup_size?: number
  checksum: string
  is_recoverable: boolean
  recovered_count: number
  created_at: string
  expires_at?: string
}

// USB Key API functions
export const usbKeyApi = {
  // List USB keys
  list: async (params?: {
    status?: string
    asset_id?: number
    skip?: number
    limit?: number
  }): Promise<{ items: USBKey[]; total: number }> => {
    const response = await api.get('/usbkeys', { params })
    return response.data
  },

  // Get USB key by ID
  get: async (id: number): Promise<USBKey> => {
    const response = await api.get(`/usbkeys/${id}`)
    return response.data
  },

  // Initialize new USB key (admin only)
  initialize: async (data: USBKeyCreate): Promise<USBKeyInitResponse> => {
    const response = await api.post('/usbkeys', data)
    return response.data
  },

  // Bind to asset
  bind: async (id: number, assetId: number): Promise<USBKey> => {
    const response = await api.post(`/usbkeys/${id}/bind`, { asset_id: assetId })
    return response.data
  },

  // Unbind from asset
  unbind: async (id: number): Promise<USBKey> => {
    const response = await api.post(`/usbkeys/${id}/unbind`)
    return response.data
  },

  // Create backup
  createBackup: async (id: number): Promise<BackupResponse> => {
    const response = await api.post(`/usbkeys/${id}/backup`)
    return response.data
  },

  // Download backup file
  downloadBackup: async (id: number): Promise<Blob> => {
    const response = await api.get(`/usbkeys/${id}/backup/download`, {
      responseType: 'blob'
    })
    return response.data
  },

  // Restore from backup
  restore: async (backupUuid: string, backupPassword: string, newLabel?: string): Promise<USBKey> => {
    const response = await api.post('/usbkeys/restore', {
      backup_uuid: backupUuid,
      backup_password: backupPassword,
      new_label: newLabel
    })
    return response.data
  },

  // Rebuild key
  rebuild: async (id: number, recoveryCode: string, newLabel?: string): Promise<USBKeyInitResponse> => {
    const response = await api.post(`/usbkeys/${id}/rebuild`, {
      recovery_code: recoveryCode,
      new_label: newLabel
    })
    return response.data
  },

  // Revoke key
  revoke: async (id: number, reason?: string): Promise<USBKey> => {
    const response = await api.post(`/usbkeys/${id}/revoke`, { reason })
    return response.data
  },

  // Delete key
  delete: async (id: number): Promise<void> => {
    await api.delete(`/usbkeys/${id}`)
  },

  // List backups
  listBackups: async (id: number): Promise<USBKeyBackup[]> => {
    const response = await api.get(`/usbkeys/${id}/backups`)
    return response.data
  },
}

// Export named api instance
export { api }

export default api
