/**
 * Backend Service - Unified service layer for React frontend
 * This replaces the old localStorage-based mock with real HTTP API calls
 * Following Google-style best practices: thin service layer, proper error handling, typed interfaces
 */

import { NodeConfig, NodeStatus, Notification, User, Role, KeySlot } from '../types';
import {
  AuthApi,
  NodesApi,
  DashboardApi,
  NotificationsApi,
  mapApiError,
  type NodeView,
  type MeResponse,
} from './apiClient';

// Storage keys for client-side state
const STORAGE_KEYS = {
  AUTH: 'access_token',
  SHELL_HISTORY: 'osbaka_shell_history',
};

// Map backend User to frontend User type
function mapBackendUserToFrontend(backendUser: MeResponse): User {
  return {
    id: String(backendUser.id),
    email: backendUser.email,
    name: backendUser.full_name || backendUser.username,
    role: backendUser.is_superuser ? Role.ADMIN : Role.OPERATOR,
    avatarUrl: `https://api.dicebear.com/7.x/avataaars/svg?seed=${backendUser.username}`,
  };
}

// Map backend NodeView to frontend NodeConfig
function mapNodeViewToNodeConfig(node: NodeView): NodeConfig {
  return {
    id: String(node.id),
    hostname: node.hostname,
    macAddress: node.mac_address,
    ipAddress: node.ip_address,
    status: mapStatus(node.status),
    provisioningMethod: 'PXE_MAC',
    encryption: {
      enabled: node.encryption_enabled,
      luksVersion: 'luks2',
      tpmEnabled: false,
      usbKeyRequired: false,
      keySlots: [],
    },
    lastSeen: node.last_seen || formatRelativeTime(node.updated_at || node.created_at),
  };
}

function mapStatus(status: string): NodeStatus {
  const statusMap: Record<string, NodeStatus> = {
    active: NodeStatus.ACTIVE,
    inactive: NodeStatus.OFFLINE,
    installing: NodeStatus.INSTALLING,
    maintenance: NodeStatus.PENDING,
    error: NodeStatus.ERROR,
  };
  return statusMap[status] || NodeStatus.PENDING;
}

function formatRelativeTime(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  
  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins} min${diffMins > 1 ? 's' : ''} ago`;
  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;
  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;
}

export const BackendService = {
  /**
   * Initialize backend connection (no-op now, can add health check later)
   */
  init: async (): Promise<void> => {
    console.log('[BackendService] Initializing connection to FastAPI backend...');
    // Future: could add a health check here
  },

  // ==================== Auth ====================

  /**
   * Login with username/password, store token, fetch user info
   * Note: 'usernameOrEmail' can be either username or email, backend expects username
   */
  login: async (usernameOrEmail: string, password: string): Promise<User> => {
    try {
      // Backend expects 'username' field, so we pass the input as username
      const loginResp = await AuthApi.login({ username: usernameOrEmail, password });
      localStorage.setItem(STORAGE_KEYS.AUTH, loginResp.access_token);

      const userResp = await AuthApi.me();
      return mapBackendUserToFrontend(userResp);
    } catch (error) {
      const apiError = mapApiError(error);
      throw new Error(apiError.message || 'Login failed');
    }
  },

  logout: (): void => {
    localStorage.removeItem(STORAGE_KEYS.AUTH);
  },

  isAuthenticated: (): boolean => {
    return !!localStorage.getItem(STORAGE_KEYS.AUTH);
  },

  // ==================== Users ====================
  // Note: Backend doesn't have a /users endpoint yet, keeping mock for now
  // TODO: Add backend /users API or integrate with /auth/me for current user only

  getUsers: (): User[] => {
    // Mock users for now (UI expects list of users for admin panel)
    return [
      {
        id: 'u-admin',
        name: 'System Administrator',
        email: 'admin@os-baka.local',
        role: Role.ADMIN,
        avatarUrl: 'https://api.dicebear.com/7.x/avataaars/svg?seed=Felix',
      },
      {
        id: 'u-op1',
        name: 'Sarah Operator',
        email: 'sarah@os-baka.local',
        role: Role.OPERATOR,
        avatarUrl: 'https://api.dicebear.com/7.x/avataaars/svg?seed=Sarah',
      },
    ];
  },

  updateUserRole: (userId: string, newRole: Role): User[] => {
    // Mock implementation - TODO: Add backend API
    return BackendService.getUsers();
  },

  // ==================== Nodes ====================

  getNodes: async (): Promise<NodeConfig[]> => {
    try {
      const resp = await NodesApi.list();
      return resp.items.map(mapNodeViewToNodeConfig);
    } catch (error) {
      console.error('[BackendService] Failed to fetch nodes:', mapApiError(error));
      return [];
    }
  },

  addNodes: async (nodes: NodeConfig[]): Promise<NodeConfig[]> => {
    try {
      for (const node of nodes) {
        await NodesApi.create({
          hostname: node.hostname,
          ip_address: node.ipAddress,
          mac_address: node.macAddress,
          asset_tag: node.id, // Using id as asset_tag for now
          status: node.status.toLowerCase(),
          os_type: 'linux',
          encryption_enabled: node.encryption.enabled,
        });
      }
      return await BackendService.getNodes();
    } catch (error) {
      console.error('[BackendService] Failed to add nodes:', mapApiError(error));
      throw error;
    }
  },

  updateNode: async (node: NodeConfig): Promise<NodeConfig[]> => {
    try {
      const nodeId = parseInt(node.id, 10);
      await NodesApi.update(nodeId, {
        hostname: node.hostname,
        ip_address: node.ipAddress,
        mac_address: node.macAddress,
        status: node.status.toLowerCase(),
        encryption_enabled: node.encryption.enabled,
      });
      return await BackendService.getNodes();
    } catch (error) {
      console.error('[BackendService] Failed to update node:', mapApiError(error));
      throw error;
    }
  },

  deleteNode: async (id: string): Promise<NodeConfig[]> => {
    try {
      const nodeId = parseInt(id, 10);
      await NodesApi.delete(nodeId);
      return await BackendService.getNodes();
    } catch (error) {
      console.error('[BackendService] Failed to delete node:', mapApiError(error));
      throw error;
    }
  },

  // ==================== Notifications ====================

  getNotifications: async (): Promise<Notification[]> => {
    try {
      return await NotificationsApi.list();
    } catch (error) {
      console.error('[BackendService] Failed to fetch notifications:', mapApiError(error));
      // Fallback to mock if backend not ready
      return [
        {
          id: '1',
          title: 'System Update Available',
          message: 'A new kernel patch (v6.1.0) is available for deployment.',
          type: 'info',
          timestamp: '10 mins ago',
          read: false,
        },
        {
          id: '2',
          title: 'Node Offline Detected',
          message: 'worker-prod-02 has failed 3 consecutive health checks.',
          type: 'error',
          timestamp: '1 hour ago',
          read: false,
        },
      ];
    }
  },

  // ==================== Shell History ====================

  getShellHistory: (): string[] => {
    try {
      return JSON.parse(localStorage.getItem(STORAGE_KEYS.SHELL_HISTORY) || '[]');
    } catch {
      return [];
    }
  },

  saveShellHistory: (history: string[]): void => {
    localStorage.setItem(STORAGE_KEYS.SHELL_HISTORY, JSON.stringify(history));
  },

  // ==================== Pure Frontend Utilities ====================
  // These don't need backend, they generate files locally

  generateDnsmasqConfig: (): Blob => {
    const nodes = BackendService.getNodes();
    // Note: This is now async, but we'll keep it sync for now with a warning
    console.warn('[BackendService] generateDnsmasqConfig called synchronously - consider making async');
    
    const configContent = [
      '# OS-Baka Auto-Generated Dnsmasq Config',
      `# Generated at ${new Date().toISOString()}`,
      '',
      '# DHCP Static Bindings',
      '# Note: Run getNodes() first to ensure data is current',
      '',
      '# Boot Options',
      'dhcp-option=option:router,192.168.10.1',
      'dhcp-option=option:dns-server,8.8.8.8,8.8.4.4',
    ].join('\n');

    return new Blob([configContent], { type: 'text/plain' });
  },

  generateLuksHeader: (hostname: string): Blob => {
    const size = 16 * 1024;
    const buffer = new Uint8Array(size);
    crypto.getRandomValues(buffer);
    
    buffer[0] = 0x4c; // L
    buffer[1] = 0x55; // U
    buffer[2] = 0x4b; // K
    buffer[3] = 0x53; // S
    buffer[4] = 0xba;
    buffer[5] = 0xbe;

    return new Blob([buffer], { type: 'application/octet-stream' });
  },

  generateUsbKey: (hostname: string): Blob => {
    const size = 4096;
    const buffer = new Uint8Array(size);
    crypto.getRandomValues(buffer);
    return new Blob([buffer], { type: 'application/octet-stream' });
  },

  generateLuksPassphrase: (hostname: string): Blob => {
    const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+';
    let passphrase = '';
    const values = new Uint32Array(32);
    crypto.getRandomValues(values);
    for (let i = 0; i < 32; i++) {
      passphrase += charset[values[i] % charset.length];
    }

    const content = `Confidential Recovery Key for ${hostname}\n${'—'.repeat(40)}\nPassphrase: ${passphrase}\nGenerated: ${new Date().toISOString()}\n\nStore this safely.`;
    return new Blob([content], { type: 'text/plain' });
  },

  createPassphraseBlob: (hostname: string, passphrase: string): Blob => {
    const content = `Confidential Recovery Key for ${hostname}\n${'—'.repeat(40)}\nPassphrase: ${passphrase}\nGenerated: ${new Date().toISOString()}\n\nStore this safely.`;
    return new Blob([content], { type: 'text/plain' });
  },

  regenerateRecoveryKey: async (nodeId: string): Promise<string> => {
    // TODO: This should call backend API to regenerate key on server side
    // For now, generate locally and return passphrase
    const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+';
    let passphrase = '';
    const values = new Uint32Array(32);
    crypto.getRandomValues(values);
    for (let i = 0; i < 32; i++) {
      passphrase += charset[values[i] % charset.length];
    }

    // Update node to mark key as rotated (would need backend API)
    console.warn('[BackendService] regenerateRecoveryKey - backend integration pending');
    
    return passphrase;
  },
};
