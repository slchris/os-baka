
export enum NodeStatus {
  PENDING = 'PENDING',
  INSTALLING = 'INSTALLING',
  ACTIVE = 'ACTIVE',
  OFFLINE = 'OFFLINE',
  ERROR = 'ERROR',
  REBUILDING = 'REBUILDING',
}

export enum Role {
  ADMIN = 'Administrator',
  OPERATOR = 'Operator',
  AUDITOR = 'Auditor',
}

export type KeySlotType = 'passphrase' | 'tpm' | 'keyfile' | 'recovery';

export interface KeySlot {
  index: number;
  active: boolean;
  type?: KeySlotType;
  label?: string;
}

export interface IpmiCredentials {
  ip: string;
  username: string;
  password?: string; // Optional in frontend for security display
  allowUntrustedCerts?: boolean;
}

export interface SshCredentials {
  username: string;
  password?: string;
  privateKey?: string;
  keyPassphrase?: string;
  port?: number;
}

export interface NodeConfig {
  id: string;
  hostname: string;
  macAddress: string;
  ipAddress: string;
  osType?: string;
  osVersion?: string;
  mirrorUrl?: string;
  timezone?: string;
  status: NodeStatus;
  provisioningMethod: 'PXE_MAC' | 'IPMI_BMC';
  ipmi?: IpmiCredentials;
  ssh?: SshCredentials;
  encryption: {
    enabled: boolean;
    luksVersion: 'luks1' | 'luks2';
    tpmEnabled: boolean;
    usbKeyRequired: boolean;
    pcrBinding?: number[];
    keySlots?: KeySlot[];
  };
  lastSeen?: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  role: Role;
  avatarUrl?: string;
}

export interface AuditLog {
  id: string;
  timestamp: string;
  action: string;
  user: string;
  details: string;
}

export interface Notification {
  id: string;
  title: string;
  message: string;
  type: 'info' | 'warning' | 'error' | 'success';
  timestamp: string;
  read: boolean;
}

export interface GeminiAnalysisResult {
  suggestions: string[];
  risks: string[];
  isValid: boolean;
}