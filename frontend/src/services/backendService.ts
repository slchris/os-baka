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
  UsersApi,
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
  // Parse PCR binding from comma-separated string to number array
  const pcrBinding = node.pcr_binding
    ? node.pcr_binding.split(',').map(Number).filter((n) => !isNaN(n))
    : undefined;

  return {
    id: String(node.id),
    hostname: node.hostname,
    macAddress: node.mac_address,
    ipAddress: node.ip_address,
    status: mapStatus(node.status),
    provisioningMethod: 'PXE_MAC',
    osType: node.os_type,
    osVersion: node.os_version,
    mirrorUrl: node.mirror_url,
    assetTag: node.asset_tag,
    timezone: node.timezone,
    sshEnabled: node.ssh_enabled ?? false,
    sshRootLogin: node.ssh_root_login ?? false,
    encryption: {
      enabled: node.encryption_enabled,
      luksVersion: 'luks2',
      tpmEnabled: node.tpm_enabled ?? false,
      usbKeyRequired: node.usb_key_required ?? false,
      pcrBinding,
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

  getUsers: async (): Promise<User[]> => {
    try {
      const resp = await UsersApi.list();
      return resp.items.map((u) => ({
        id: String(u.id),
        name: u.full_name || u.username,
        email: u.email,
        role: u.role === 'admin' ? Role.ADMIN : u.role === 'auditor' ? Role.AUDITOR : Role.OPERATOR,
        avatarUrl: `https://api.dicebear.com/7.x/avataaars/svg?seed=${u.username}`,
      }));
    } catch (error) {
      console.error('[BackendService] Failed to fetch users:', mapApiError(error));
      return [];
    }
  },

  updateUserRole: async (userId: string, newRole: Role): Promise<User[]> => {
    try {
      const roleStr = newRole === Role.ADMIN ? 'admin' : newRole === Role.AUDITOR ? 'auditor' : 'operator';
      await UsersApi.update(parseInt(userId, 10), { role: roleStr });
    } catch (error) {
      console.error('[BackendService] Failed to update user role:', mapApiError(error));
    }
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
      return [];
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

  generateDnsmasqConfig: async (): Promise<Blob> => {
    const nodes = await BackendService.getNodes();

    const lines = [
      '# OS-Baka Auto-Generated Dnsmasq Config',
      `# Generated at ${new Date().toISOString()}`,
      '',
      '# DHCP Static Bindings',
    ];

    for (const node of nodes) {
      lines.push(`dhcp-host=${node.macAddress},${node.ipAddress},${node.hostname}`);
    }

    lines.push('', '# Boot Options');
    lines.push('dhcp-option=option:router,192.168.10.1');
    lines.push('dhcp-option=option:dns-server,8.8.8.8,8.8.4.4');

    return new Blob([lines.join('\n')], { type: 'text/plain' });
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
    try {
      const res = await NodesApi.rotatePassphrase(parseInt(nodeId));
      return res.passphrase;
    } catch (error) {
      console.error('[BackendService] Failed to rotate passphrase:', mapApiError(error));
      throw error;
    }
  },
};
