import axios, { AxiosError, AxiosInstance } from 'axios';

// Base API URL, aligned with backend FastAPI prefix
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8000/api/v1';

export interface ApiErrorShape {
  message: string;
  status?: number;
  details?: unknown;
}

// Lightweight HTTP client following Google-style guidelines: small, typed, centralized
class HttpClient {
  private instance: AxiosInstance;

  constructor(baseURL: string) {
    this.instance = axios.create({
      baseURL,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    this.instance.interceptors.request.use(
      (config) => {
        const token = localStorage.getItem('access_token');
        if (token && config.headers) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => Promise.reject(error),
    );

    this.instance.interceptors.response.use(
      (response) => response,
      (error: AxiosError) => {
        if (error.response?.status === 401) {
          localStorage.removeItem('access_token');
          if (!window.location.pathname.includes('/login')) {
            window.location.href = '/login';
          }
        }
        return Promise.reject(error);
      },
    );
  }

  get client() {
    return this.instance;
  }

  static toApiError(error: unknown): ApiErrorShape {
    if (axios.isAxiosError(error)) {
      const axiosError = error as AxiosError;
      const status = axiosError.response?.status;
      const data = axiosError.response?.data as any;
      // Backend returns { "error": "message" }, check both common patterns
      const detail = data?.error || data?.detail || data?.message;
      return {
        message: typeof detail === 'string' ? detail : (axiosError.message || 'Request failed'),
        status,
        details: axiosError.response?.data,
      };
    }
    if (error instanceof Error) {
      return { message: error.message, details: error };
    }
    return { message: 'Unexpected error', details: error };
  }
}

export const httpClient = new HttpClient(API_BASE_URL).client;
export const mapApiError = HttpClient.toApiError;

// Auth API
export interface LoginResponse {
  access_token: string;
  token_type: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface MeResponse {
  id: number;
  username: string;
  email: string;
  full_name?: string;
  is_active: boolean;
  is_superuser: boolean;
}

export const AuthApi = {
  async login(payload: LoginRequest): Promise<LoginResponse> {
    const res = await httpClient.post<LoginResponse>('/auth/login', payload);
    return res.data;
  },

  async me(): Promise<MeResponse> {
    const res = await httpClient.get<MeResponse>('/auth/me');
    return res.data;
  },
};

// Minimal Node view mapped from backend Assets for the new UI
export interface NodeView {
  id: number;
  hostname: string;
  ip_address: string;
  mac_address: string;
  asset_tag: string;
  status: 'active' | 'inactive' | 'installing' | 'maintenance' | 'error';
  os_type?: string;
  os_version?: string;
  mirror_url?: string;
  timezone?: string;
  ssh_enabled?: boolean;
  ssh_root_login?: boolean;
  encryption_enabled: boolean;
  tpm_enabled?: boolean;
  usb_key_required?: boolean;
  pcr_binding?: string;
  last_seen?: string;
  created_at: string;
  updated_at?: string;
}

export interface NodeListResponse {
  items: NodeView[];
  total: number;
}

export interface NodeCreateRequest {
  hostname: string;
  ip_address: string;
  mac_address: string;
  asset_tag: string;
  status?: string;
  os_type?: string;
  os_version?: string;
  mirror_url?: string;
  encryption_enabled?: boolean;
  encryption_passphrase?: string;
  tpm_enabled?: boolean;
  usb_key_required?: boolean;
  pcr_binding?: string;
}

export interface NodeUpdateRequest {
  hostname?: string;
  ip_address?: string;
  mac_address?: string;
  asset_tag?: string;
  status?: string;
  os_type?: string;
  os_version?: string;
  mirror_url?: string;
  encryption_enabled?: boolean;
  encryption_passphrase?: string;
  tpm_enabled?: boolean;
  usb_key_required?: boolean;
  pcr_binding?: string;
}

export const NodesApi = {
  async list(params?: { skip?: number; limit?: number; status?: string }): Promise<NodeListResponse> {
    const res = await httpClient.get<NodeListResponse>('/nodes', { params });
    return res.data;
  },

  async get(id: number): Promise<NodeView> {
    const res = await httpClient.get<NodeView>(`/nodes/${id}`);
    return res.data;
  },

  async create(data: NodeCreateRequest): Promise<NodeView> {
    const res = await httpClient.post<NodeView>('/nodes', data);
    return res.data;
  },

  async update(id: number, data: NodeUpdateRequest): Promise<NodeView> {
    const res = await httpClient.put<NodeView>(`/nodes/${id}`, data);
    return res.data;
  },

  async getPassphrase(id: number): Promise<{ passphrase: string }> {
    const res = await httpClient.get<{ passphrase: string }>(`/nodes/${id}/passphrase`);
    return res.data;
  },

  async rebuild(id: number): Promise<{ success: boolean; message: string; status: string }> {
    const res = await httpClient.post<{ success: boolean; message: string; status: string }>(`/nodes/${id}/rebuild`);
    return res.data;
  },

  async delete(id: number): Promise<void> {
    await httpClient.delete(`/nodes/${id}`);
  },
};

// Dashboard & Notifications (to be added to backend)
export interface DashboardSummary {
  nodes_total: number;
  nodes_active: number;
  nodes_error: number;
  nodes_installing: number;
  deployments_total: number;
  deployments_pending: number;
  deployments_in_progress: number;
  deployments_completed: number;
  deployments_failed: number;
  pxe_service_running: boolean;
}

export interface NotificationItem {
  id: string;
  title: string;
  message: string;
  type: 'info' | 'warning' | 'error' | 'success';
  timestamp: string;
  read: boolean;
}

export const DashboardApi = {
  async summary(): Promise<DashboardSummary> {
    const res = await httpClient.get<DashboardSummary>('/dashboard/summary');
    return res.data;
  },
};

export const NotificationsApi = {
  async list(): Promise<NotificationItem[]> {
    const res = await httpClient.get<{ items: NotificationItem[]; total: number }>('/notifications');
    return res.data.items;
  },

  async markRead(id: string): Promise<void> {
    await httpClient.post(`/notifications/${id}/read`);
  },
};

// DHCP Types
export interface DHCPConfig {
  id: number;
  name: string;
  interface: string;
  range_start: string;
  range_end: string;
  subnet_mask: string;
  gateway: string;
  dns_server: string;
  lease_time: string;
  domain: string;
  tftp_server: string;
  boot_file: string;
  next_server?: string;
  boot_server_ip?: string;
  mirror_url?: string;
  kernel_params?: string;
  is_active: boolean;
  enable_pxe: boolean;
  created_at: string;
  updated_at: string;
}

export interface DHCPReservation {
  id: number;
  mac_address: string;
  ip_address: string;
  hostname: string;
  description: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface DHCPConfigRequest {
  name: string;
  interface?: string;
  range_start: string;
  range_end: string;
  subnet_mask?: string;
  gateway?: string;
  dns_server?: string;
  lease_time?: string;
  domain?: string;
  tftp_server?: string;
  boot_file?: string;
  next_server?: string;
  boot_server_ip?: string;
  mirror_url?: string;
  kernel_params?: string;
  is_active?: boolean;
  enable_pxe?: boolean;
}

export interface DHCPReservationRequest {
  mac_address: string;
  ip_address: string;
  hostname: string;
  description?: string;
  is_active?: boolean;
}

export const DHCPApi = {
  // Configurations
  async listConfigs(): Promise<{ items: DHCPConfig[]; total: number }> {
    const res = await httpClient.get<{ items: DHCPConfig[]; total: number }>('/dhcp/configs');
    return res.data;
  },

  async getConfig(id: number): Promise<DHCPConfig> {
    const res = await httpClient.get<DHCPConfig>(`/dhcp/configs/${id}`);
    return res.data;
  },

  async getActiveConfig(): Promise<DHCPConfig> {
    const res = await httpClient.get<DHCPConfig>('/dhcp/config/active');
    return res.data;
  },

  async createConfig(data: DHCPConfigRequest): Promise<DHCPConfig> {
    const res = await httpClient.post<DHCPConfig>('/dhcp/configs', data);
    return res.data;
  },

  async updateConfig(id: number, data: DHCPConfigRequest): Promise<DHCPConfig> {
    const res = await httpClient.put<DHCPConfig>(`/dhcp/configs/${id}`, data);
    return res.data;
  },

  async deleteConfig(id: number): Promise<void> {
    await httpClient.delete(`/dhcp/configs/${id}`);
  },

  async restartService(): Promise<{ success: boolean; message: string }> {
    const res = await httpClient.post<{ success: boolean; message: string }>('/dhcp/service/restart');
    return res.data;
  },

  // Reservations
  async listReservations(): Promise<{ items: DHCPReservation[]; total: number }> {
    const res = await httpClient.get<{ items: DHCPReservation[]; total: number }>('/dhcp/reservations');
    return res.data;
  },

  async createReservation(data: DHCPReservationRequest): Promise<DHCPReservation> {
    const res = await httpClient.post<DHCPReservation>('/dhcp/reservations', data);
    return res.data;
  },

  async updateReservation(id: number, data: DHCPReservationRequest): Promise<DHCPReservation> {
    const res = await httpClient.put<DHCPReservation>(`/dhcp/reservations/${id}`, data);
    return res.data;
  },

  async deleteReservation(id: number): Promise<void> {
    await httpClient.delete(`/dhcp/reservations/${id}`);
  },

  async syncFromNodes(): Promise<{ success: boolean; created: number; updated: number }> {
    const res = await httpClient.post<{ success: boolean; created: number; updated: number }>('/dhcp/reservations/sync');
    return res.data;
  },
};

// System API for hardware info
export interface NetworkInterface {
  name: string;
  mac_address: string;
  ip_addresses: string[];
  flags: string[];
}

export const SystemApi = {
  async listInterfaces(): Promise<NetworkInterface[]> {
    const res = await httpClient.get<NetworkInterface[]>('/system/interfaces');
    return res.data;
  },
};

// Boot Asset API
export interface BootAsset {
  id: number;
  name: string;
  file_name: string;
  type: string;
  size: number;
  path: string;
  checksum: string;
  created_at: string;
}

export const AssetApi = {
  async listBootAssets(): Promise<{ items: BootAsset[]; total: number }> {
    const res = await httpClient.get<{ items: BootAsset[]; total: number }>('/assets/boot');
    return res.data;
  },

  async uploadBootAsset(file: File, type: string, name?: string): Promise<BootAsset> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('type', type);
    if (name) formData.append('name', name);

    const res = await httpClient.post<BootAsset>('/assets/boot', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    return res.data;
  },

  async deleteBootAsset(id: number): Promise<{ success: boolean }> {
    const res = await httpClient.delete<{ success: boolean }>(`/assets/boot/${id}`);
    return res.data;
  },
};
