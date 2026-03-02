/**
 * BackendService Tests
 * Tests for the service layer that bridges apiClient ↔ frontend types
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock apiClient before importing backendService
vi.mock('../apiClient', () => ({
    httpClient: {
        get: vi.fn(),
        post: vi.fn(),
        put: vi.fn(),
        delete: vi.fn(),
    },
    AuthApi: {
        login: vi.fn(),
        me: vi.fn(),
    },
    NodesApi: {
        list: vi.fn(),
        get: vi.fn(),
        create: vi.fn(),
        update: vi.fn(),
        getPassphrase: vi.fn(),
        rebuild: vi.fn(),
        delete: vi.fn(),
        rotatePassphrase: vi.fn(),
        testIpmi: vi.fn(),
        powerAction: vi.fn(),
    },
    NotificationsApi: {
        list: vi.fn(),
        markRead: vi.fn(),
    },
    DashboardApi: {
        summary: vi.fn(),
    },
    UsersApi: {
        list: vi.fn(),
        create: vi.fn(),
        update: vi.fn(),
        changePassword: vi.fn(),
        delete: vi.fn(),
    },
    mapApiError: (e: unknown) => {
        if (e instanceof Error) return { message: e.message };
        return { message: 'Unexpected error' };
    },
}));

import { BackendService } from '../backendService';
import { AuthApi, NodesApi, NotificationsApi, UsersApi } from '../apiClient';

describe('BackendService', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.clear();
    });

    // ==================== Auth ====================
    describe('Auth', () => {
        it('should login and store token', async () => {
            vi.mocked(AuthApi.login).mockResolvedValue({
                access_token: 'test-jwt-token',
                token_type: 'bearer',
            });
            vi.mocked(AuthApi.me).mockResolvedValue({
                id: 1,
                username: 'admin',
                email: 'admin@test.com',
                is_active: true,
                is_superuser: true,
            });

            const user = await BackendService.login('admin', 'password');

            expect(AuthApi.login).toHaveBeenCalledWith({ username: 'admin', password: 'password' });
            expect(localStorage.getItem('access_token')).toBe('test-jwt-token');
            expect(user.name).toBe('admin');
            expect(user.role).toBe('Administrator');
        });

        it('should map non-superuser to operator role', async () => {
            vi.mocked(AuthApi.login).mockResolvedValue({
                access_token: 'token',
                token_type: 'bearer',
            });
            vi.mocked(AuthApi.me).mockResolvedValue({
                id: 2,
                username: 'operator',
                email: 'op@test.com',
                is_active: true,
                is_superuser: false,
            });

            const user = await BackendService.login('operator', 'pass');
            expect(user.role).toBe('Operator');
        });

        it('should clear token on logout', () => {
            localStorage.setItem('access_token', 'some-token');
            BackendService.logout();
            expect(localStorage.getItem('access_token')).toBeNull();
        });

        it('should check authentication status', () => {
            expect(BackendService.isAuthenticated()).toBe(false);
            localStorage.setItem('access_token', 'token');
            expect(BackendService.isAuthenticated()).toBe(true);
        });
    });

    // ==================== Nodes ====================
    describe('Nodes', () => {
        it('should map backend nodes to frontend NodeConfig', async () => {
            vi.mocked(NodesApi.list).mockResolvedValue({
                items: [
                    {
                        id: 1,
                        hostname: 'worker-01',
                        ip_address: '10.0.0.1',
                        mac_address: 'AA:BB:CC:DD:EE:FF',
                        asset_tag: 'ASSET-001',
                        status: 'active',
                        os_type: 'debian',
                        os_version: '12',
                        encryption_enabled: true,
                        tpm_enabled: true,
                        usb_key_required: false,
                        pcr_binding: '0,7',
                        created_at: '2025-01-01T00:00:00Z',
                    },
                ],
                total: 1,
            });

            const nodes = await BackendService.getNodes();
            expect(nodes).toHaveLength(1);
            expect(nodes[0].hostname).toBe('worker-01');
            expect(nodes[0].ipAddress).toBe('10.0.0.1');
            expect(nodes[0].encryption.enabled).toBe(true);
            expect(nodes[0].encryption.tpmEnabled).toBe(true);
            expect(nodes[0].encryption.pcrBinding).toEqual([0, 7]);
        });

        it('should handle nodes with missing optional fields', async () => {
            vi.mocked(NodesApi.list).mockResolvedValue({
                items: [
                    {
                        id: 2,
                        hostname: 'bare-node',
                        ip_address: '10.0.0.2',
                        mac_address: '11:22:33:44:55:66',
                        asset_tag: '',
                        status: 'installing',
                        os_type: '',
                        encryption_enabled: false,
                        created_at: '2025-01-01T00:00:00Z',
                    },
                ],
                total: 1,
            });

            const nodes = await BackendService.getNodes();
            expect(nodes[0].osType).toBe(''); // preserves backend value as-is
            expect(nodes[0].encryption.enabled).toBe(false);
        });

        it('should return empty array on error', async () => {
            vi.mocked(NodesApi.list).mockRejectedValue(new Error('Network error'));
            const nodes = await BackendService.getNodes();
            expect(nodes).toEqual([]);
        });
    });

    // ==================== Users ====================
    describe('Users', () => {
        it('should fetch and map users', async () => {
            vi.mocked(UsersApi.list).mockResolvedValue({
                items: [
                    { id: 1, username: 'admin', email: 'admin@test.com', full_name: 'Admin', role: 'admin' },
                    { id: 2, username: 'op', email: 'op@test.com', full_name: '', role: 'operator' },
                ],
                total: 2,
            });

            const users = await BackendService.getUsers();
            expect(users).toHaveLength(2);
            expect(users[0].role).toBe('Administrator');
            expect(users[1].name).toBe('op');
        });
    });

    // ==================== Notifications ====================
    describe('Notifications', () => {
        it('should return empty array on error (no mock fallback)', async () => {
            vi.mocked(NotificationsApi.list).mockRejectedValue(new Error('fail'));
            const notifs = await BackendService.getNotifications();
            expect(notifs).toEqual([]);
        });

        it('should return notifications from backend', async () => {
            vi.mocked(NotificationsApi.list).mockResolvedValue([
                {
                    id: '1',
                    title: 'Test',
                    message: 'Hello',
                    type: 'info',
                    timestamp: '5 mins ago',
                    read: false,
                },
            ]);

            const notifs = await BackendService.getNotifications();
            expect(notifs).toHaveLength(1);
            expect(notifs[0].title).toBe('Test');
        });
    });

    // ==================== Shell History ====================
    describe('Shell History', () => {
        it('should persist and retrieve shell history', () => {
            BackendService.saveShellHistory(['ls', 'pwd', 'cat /etc/os-release']);
            const history = BackendService.getShellHistory();
            expect(history).toEqual(['ls', 'pwd', 'cat /etc/os-release']);
        });

        it('should return empty array when no history exists', () => {
            const history = BackendService.getShellHistory();
            expect(history).toEqual([]);
        });
    });

    // ==================== Utility Functions ====================
    describe('Utilities', () => {
        it('should generate a valid dnsmasq config blob', async () => {
            vi.mocked(NodesApi.list).mockResolvedValue({
                items: [
                    {
                        id: 1,
                        hostname: 'node1',
                        ip_address: '10.0.0.1',
                        mac_address: 'AA:BB:CC:DD:EE:FF',
                        asset_tag: '',
                        status: 'active',
                        os_type: 'debian',
                        encryption_enabled: false,
                        created_at: '2025-01-01T00:00:00Z',
                    },
                ],
                total: 1,
            });

            const blob = await BackendService.generateDnsmasqConfig();
            expect(blob).toBeInstanceOf(Blob);

            const text = await blob.text();
            expect(text).toContain('AA:BB:CC:DD:EE:FF');
            expect(text).toContain('10.0.0.1');
            expect(text).toContain('node1');
        });

        it('should generate LUKS passphrase blob', () => {
            const blob = BackendService.generateLuksPassphrase('test-host');
            expect(blob).toBeInstanceOf(Blob);
        });

        it('should generate LUKS header blob', () => {
            const blob = BackendService.generateLuksHeader('test-host');
            expect(blob).toBeInstanceOf(Blob);
        });
    });
});
