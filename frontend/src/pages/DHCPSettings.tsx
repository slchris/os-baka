import React, { useEffect, useState } from 'react';
import { Network, Server, Plus, Trash2, Edit2, RefreshCw, Save, X, Check, AlertCircle, FileText, Upload, HardDrive } from 'lucide-react';
import { DHCPApi, DHCPConfig, DHCPReservation, DHCPConfigRequest, DHCPReservationRequest, SystemApi, NetworkInterface, AssetApi, BootAsset } from '../services/apiClient';

export const DHCPSettings: React.FC = () => {
    const [activeTab, setActiveTab] = useState<'config' | 'reservations' | 'assets'>('config');
    const [configs, setConfigs] = useState<DHCPConfig[]>([]);
    const [reservations, setReservations] = useState<DHCPReservation[]>([]);
    const [assets, setAssets] = useState<BootAsset[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [editingConfig, setEditingConfig] = useState<DHCPConfig | null>(null);
    const [editingReservation, setEditingReservation] = useState<DHCPReservation | null>(null);
    const [showNewReservation, setShowNewReservation] = useState(false);
    const [showUploadAsset, setShowUploadAsset] = useState(false);

    useEffect(() => {
        loadData();
    }, []);

    const loadData = async () => {
        setLoading(true);
        setError(null);
        try {
            const [configRes, reservationRes, assetRes] = await Promise.all([
                DHCPApi.listConfigs().catch(() => ({ items: [], total: 0 })),
                DHCPApi.listReservations().catch(() => ({ items: [], total: 0 })),
                AssetApi.listBootAssets().catch(() => ({ items: [], total: 0 })),
            ]);
            // Safely extract items, filtering out any null entries
            const configItems = Array.isArray(configRes?.items)
                ? configRes.items.filter((c): c is DHCPConfig => c !== null && c !== undefined)
                : [];
            const reservationItems = Array.isArray(reservationRes?.items)
                ? reservationRes.items.filter((r): r is DHCPReservation => r !== null && r !== undefined)
                : [];
            const assetItems = Array.isArray(assetRes?.items)
                ? assetRes.items.filter((a): a is BootAsset => a !== null && a !== undefined)
                : [];
            setConfigs(configItems);
            setReservations(reservationItems);
            setAssets(assetItems);
        } catch (e: any) {
            setError(e.message || 'Failed to load DHCP settings');
            setConfigs([]);
            setReservations([]);
        } finally {
            setLoading(false);
        }
    };

    const handleSaveConfig = async (config: DHCPConfig) => {
        if (!config) return;
        try {
            const data: DHCPConfigRequest = {
                name: config.name || '',
                interface: config.interface || '',
                range_start: config.range_start || '',
                range_end: config.range_end || '',
                subnet_mask: config.subnet_mask || '',
                gateway: config.gateway || '',
                dns_server: config.dns_server || '',
                lease_time: config.lease_time || '',
                domain: config.domain || '',
                tftp_server: config.tftp_server || '',
                boot_file: config.boot_file || '',
                next_server: config.next_server || '',
                boot_server_ip: config.boot_server_ip || '',
                kernel_params: config.kernel_params || '',
                is_active: config.is_active || false,
                enable_pxe: config.enable_pxe || false,
            };
            if (config.id && config.id > 0) {
                await DHCPApi.updateConfig(config.id, data);
            } else {
                await DHCPApi.createConfig(data);
            }
            setEditingConfig(null);
            loadData();
        } catch (e: any) {
            setError(e.message || 'Failed to save configuration');
        }
    };

    const handleSyncFromNodes = async () => {
        try {
            const result = await DHCPApi.syncFromNodes();
            alert(`Synced ${result.created} new reservations, updated ${result.updated} existing.`);
            loadData();
        } catch (e: any) {
            setError(e.message || 'Failed to sync from nodes');
        }
    };

    const handleSaveReservation = async (reservation: DHCPReservation) => {
        try {
            const data: DHCPReservationRequest = {
                mac_address: reservation.mac_address,
                ip_address: reservation.ip_address,
                hostname: reservation.hostname,
                description: reservation.description,
                is_active: reservation.is_active,
            };
            if (reservation.id) {
                await DHCPApi.updateReservation(reservation.id, data);
            } else {
                await DHCPApi.createReservation(data);
            }
            setEditingReservation(null);
            setShowNewReservation(false);
            loadData();
        } catch (e: any) {
            setError(e.message || 'Failed to save reservation');
        }
    };

    const handleDeleteReservation = async (id: number) => {
        if (!confirm('Are you sure you want to delete this reservation?')) return;
        try {
            await DHCPApi.deleteReservation(id);
            loadData();
        } catch (e: any) {
            setError(e.message || 'Failed to delete reservation');
        }
    };

    const handleDeleteAsset = async (id: number) => {
        if (!confirm('Are you sure you want to delete this asset?')) return;
        try {
            await AssetApi.deleteBootAsset(id);
            loadData();
        } catch (e: any) {
            setError(e.message || 'Failed to delete asset');
        }
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <RefreshCw className="w-8 h-8 animate-spin text-blue-500" />
            </div>
        );
    }

    return (
        <div className="space-y-6 max-w-6xl animate-fade-in pb-10">
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="text-2xl font-bold text-gray-900 dark:text-white">DHCP Settings</h2>
                    <p className="text-gray-500 dark:text-gray-400">Configure DHCP server and IP reservations for PXE boot.</p>
                </div>
                <div className="flex gap-2">
                    <button
                        onClick={async () => {
                            try {
                                await DHCPApi.restartService();
                                alert('DHCP Service restarted successfully');
                            } catch (e: any) {
                                setError(e.message || 'Failed to restart service');
                            }
                        }}
                        className="flex items-center gap-2 px-4 py-2 text-sm bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300 rounded-lg hover:bg-orange-200 dark:hover:bg-orange-900/50 transition-colors"
                    >
                        <RefreshCw className="w-4 h-4" /> Restart Service
                    </button>
                    <button
                        onClick={loadData}
                        className="flex items-center gap-2 px-4 py-2 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
                    >
                        <RefreshCw className="w-4 h-4" /> Refresh
                    </button>
                </div>
            </div>

            {error && (
                <div className="flex items-center gap-2 p-4 bg-red-50 dark:bg-red-900/20 border border-red-100 dark:border-red-900/30 rounded-xl text-red-600 dark:text-red-400">
                    <AlertCircle className="w-5 h-5" />
                    {error}
                </div>
            )}

            {/* Tabs */}
            <div className="flex gap-1 bg-gray-100 dark:bg-gray-800 p-1 rounded-xl w-fit">
                <button
                    onClick={() => setActiveTab('config')}
                    className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${activeTab === 'config'
                        ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm'
                        : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                        }`}
                >
                    <Network className="w-4 h-4 inline mr-2" />
                    Configuration
                </button>
                <button
                    onClick={() => setActiveTab('reservations')}
                    className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${activeTab === 'reservations'
                        ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm'
                        : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                        }`}
                >
                    <Server className="w-4 h-4 inline mr-2" />
                    Reservations ({reservations.length})
                </button>
                <button
                    onClick={() => setActiveTab('assets')}
                    className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${activeTab === 'assets'
                        ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm'
                        : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                        }`}
                >
                    <HardDrive className="w-4 h-4 inline mr-2" />
                    Boot Assets
                </button>
            </div>

            {/* Configuration Tab */}
            {activeTab === 'config' && (
                <div className="space-y-4">
                    <div className="flex justify-end">
                        <button
                            onClick={() => setEditingConfig({
                                id: 0,
                                name: '',
                                interface: '',
                                range_start: '',
                                range_end: '',
                                subnet_mask: '',
                                gateway: '',
                                dns_server: '',
                                lease_time: '',
                                domain: '',
                                tftp_server: '',
                                boot_file: '',
                                next_server: '',
                                boot_server_ip: '',
                                kernel_params: '',
                                is_active: false,
                                enable_pxe: false,
                                created_at: '',
                                updated_at: ''
                            })}
                            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm"
                        >
                            <Plus className="w-4 h-4" /> Add Configuration
                        </button>
                    </div>
                    {editingConfig?.id === 0 && (
                        <ConfigEditor config={editingConfig} assets={assets} onSave={handleSaveConfig} onCancel={() => setEditingConfig(null)} />
                    )}
                    {configs.length === 0 ? (
                        <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-2xl shadow-sm p-8 text-center">
                            <Network className="w-12 h-12 text-gray-300 dark:text-gray-600 mx-auto mb-4" />
                            <p className="text-gray-500 dark:text-gray-400">No DHCP configurations found.</p>
                            <p className="text-sm text-gray-400 dark:text-gray-500 mt-1">A default configuration will be created on first backend startup.</p>
                        </div>
                    ) : (
                        configs.map((config) => (
                            <div key={config.id} className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-2xl shadow-sm overflow-hidden">
                                <div className="p-6 border-b border-gray-100 dark:border-gray-700 flex items-center justify-between">
                                    <div className="flex items-center gap-3">
                                        <h3 className="font-bold text-gray-900 dark:text-white">{config.name}</h3>
                                        {config.is_active && (
                                            <span className="px-2 py-1 bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400 text-xs font-medium rounded-full flex items-center gap-1">
                                                <Check className="w-3 h-3" /> Active
                                            </span>
                                        )}
                                        {config.enable_pxe && (
                                            <span className="px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 text-xs font-medium rounded-full">
                                                PXE Enabled
                                            </span>
                                        )}
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <button
                                            onClick={() => setEditingConfig(editingConfig?.id === config.id ? null : config)}
                                            className="flex items-center gap-1 px-3 py-1.5 text-sm bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 rounded-lg hover:bg-blue-100 dark:hover:bg-blue-900/30 transition-colors"
                                        >
                                            <Edit2 className="w-4 h-4" /> Edit
                                        </button>
                                        <button
                                            onClick={async () => {
                                                if (!confirm(`Are you sure you want to delete configuration "${config.name}"?`)) return;
                                                try {
                                                    await DHCPApi.deleteConfig(config.id);
                                                    loadData();
                                                } catch (e: any) {
                                                    setError(e.message || 'Failed to delete configuration');
                                                }
                                            }}
                                            className="flex items-center gap-1 px-3 py-1.5 text-sm bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 rounded-lg hover:bg-red-100 dark:hover:bg-red-900/30 transition-colors"
                                        >
                                            <Trash2 className="w-4 h-4" />
                                        </button>
                                    </div>
                                </div>

                                {editingConfig?.id === config.id ? (
                                    <ConfigEditor config={editingConfig} assets={assets} onSave={handleSaveConfig} onCancel={() => setEditingConfig(null)} />
                                ) : (
                                    <div className="p-6 grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                                        <div>
                                            <p className="text-gray-500 dark:text-gray-400">Interface</p>
                                            <p className="font-medium text-gray-900 dark:text-white">{config.interface || '-'}</p>
                                        </div>
                                        <div>
                                            <p className="text-gray-500 dark:text-gray-400">DHCP Range</p>
                                            <p className="font-medium text-gray-900 dark:text-white">{config.range_start || '-'} - {config.range_end || '-'}</p>
                                        </div>
                                        <div>
                                            <p className="text-gray-500 dark:text-gray-400">Subnet Mask</p>
                                            <p className="font-medium text-gray-900 dark:text-white">{config.subnet_mask || '-'}</p>
                                        </div>
                                        <div>
                                            <p className="text-gray-500 dark:text-gray-400">Gateway</p>
                                            <p className="font-medium text-gray-900 dark:text-white">{config.gateway || '-'}</p>
                                        </div>
                                        <div>
                                            <p className="text-gray-500 dark:text-gray-400">DNS Server</p>
                                            <p className="font-medium text-gray-900 dark:text-white">{config.dns_server || '-'}</p>
                                        </div>
                                        <div>
                                            <p className="text-gray-500 dark:text-gray-400">Lease Time</p>
                                            <p className="font-medium text-gray-900 dark:text-white">{config.lease_time || '-'}</p>
                                        </div>
                                        <div>
                                            <p className="text-gray-500 dark:text-gray-400">Domain</p>
                                            <p className="font-medium text-gray-900 dark:text-white">{config.domain || '-'}</p>
                                        </div>
                                        <div>
                                            <p className="text-gray-500 dark:text-gray-400">Boot File</p>
                                            <p className="font-medium text-gray-900 dark:text-white">{config.boot_file || '-'}</p>
                                        </div>
                                        <div>
                                            <p className="text-gray-500 dark:text-gray-400">Next Server</p>
                                            <p className="font-medium text-gray-900 dark:text-white">{config.next_server || '-'}</p>
                                        </div>
                                        <div>
                                            <p className="text-gray-500 dark:text-gray-400">Boot Server IP</p>
                                            <p className="font-medium text-gray-900 dark:text-white">{config.boot_server_ip || '-'}</p>
                                        </div>
                                        <div className="col-span-2">
                                            <p className="text-gray-500 dark:text-gray-400">Mirror URL</p>
                                            <p className="font-medium text-gray-900 dark:text-white font-mono text-xs">{config.mirror_url || '使用默认镜像源'}</p>
                                        </div>
                                        <div className="col-span-2">
                                            <p className="text-gray-500 dark:text-gray-400">Kernel Params</p>
                                            <p className="font-medium text-gray-900 dark:text-white font-mono text-xs">{config.kernel_params || '-'}</p>
                                        </div>
                                    </div>
                                )}
                            </div>
                        ))
                    )}
                </div>
            )}

            {/* Reservations Tab */}
            {activeTab === 'reservations' && (
                <div className="space-y-4">
                    <div className="flex items-center gap-2">
                        <button
                            onClick={() => { setShowNewReservation(true); setEditingReservation({ id: 0, mac_address: '', ip_address: '', hostname: '', description: '', is_active: true, created_at: '', updated_at: '' }); }}
                            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm"
                        >
                            <Plus className="w-4 h-4" /> Add Reservation
                        </button>
                        <button
                            onClick={handleSyncFromNodes}
                            className="flex items-center gap-2 px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors text-sm"
                        >
                            <RefreshCw className="w-4 h-4" /> Sync from Nodes
                        </button>
                    </div>

                    {showNewReservation && editingReservation && (
                        <ReservationEditor
                            reservation={editingReservation}
                            onSave={handleSaveReservation}
                            onCancel={() => { setShowNewReservation(false); setEditingReservation(null); }}
                        />
                    )}

                    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-2xl shadow-sm overflow-hidden">
                        <table className="w-full text-sm text-left">
                            <thead className="text-xs text-gray-500 dark:text-gray-400 uppercase bg-gray-50 dark:bg-gray-900/50 border-b border-gray-100 dark:border-gray-700">
                                <tr>
                                    <th className="px-6 py-4 font-medium">MAC Address</th>
                                    <th className="px-6 py-4 font-medium">IP Address</th>
                                    <th className="px-6 py-4 font-medium">Hostname</th>
                                    <th className="px-6 py-4 font-medium">Description</th>
                                    <th className="px-6 py-4 font-medium">Status</th>
                                    <th className="px-6 py-4 font-medium text-right">Actions</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                                {reservations.length === 0 ? (
                                    <tr>
                                        <td colSpan={6} className="px-6 py-8 text-center text-gray-500 dark:text-gray-400">
                                            No reservations configured. Add one or sync from registered nodes.
                                        </td>
                                    </tr>
                                ) : (
                                    reservations.map((r) => (
                                        <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors">
                                            <td className="px-6 py-4 font-mono text-gray-900 dark:text-white">{r.mac_address}</td>
                                            <td className="px-6 py-4 font-mono text-gray-900 dark:text-white">{r.ip_address}</td>
                                            <td className="px-6 py-4 text-gray-900 dark:text-white">{r.hostname}</td>
                                            <td className="px-6 py-4 text-gray-500 dark:text-gray-400">{r.description || '-'}</td>
                                            <td className="px-6 py-4">
                                                {r.is_active ? (
                                                    <span className="px-2 py-1 bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400 text-xs font-medium rounded-full">Active</span>
                                                ) : (
                                                    <span className="px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 text-xs font-medium rounded-full">Disabled</span>
                                                )}
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <button
                                                    onClick={() => setEditingReservation(r)}
                                                    className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 p-1"
                                                >
                                                    <Edit2 className="w-4 h-4" />
                                                </button>
                                                <button
                                                    onClick={() => handleDeleteReservation(r.id)}
                                                    className="text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-300 p-1 ml-2"
                                                >
                                                    <Trash2 className="w-4 h-4" />
                                                </button>
                                            </td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>

                    {editingReservation && !showNewReservation && (
                        <ReservationEditor
                            reservation={editingReservation}
                            onSave={handleSaveReservation}
                            onCancel={() => setEditingReservation(null)}
                        />
                    )}
                </div>
            )}

            {/* Boot Assets Tab */}
            {activeTab === 'assets' && (
                <div className="space-y-4">
                    <div className="flex justify-end">
                        <button
                            onClick={() => setShowUploadAsset(true)}
                            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm"
                        >
                            <Upload className="w-4 h-4" /> Upload Asset
                        </button>
                    </div>

                    {showUploadAsset && (
                        <AssetUploader
                            onUpload={async (file, type, name) => {
                                try {
                                    await AssetApi.uploadBootAsset(file, type, name);
                                    setShowUploadAsset(false);
                                    loadData();
                                } catch (e: any) {
                                    setError(e.message || 'Failed to upload asset');
                                }
                            }}
                            onCancel={() => setShowUploadAsset(false)}
                        />
                    )}

                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        {assets.length === 0 ? (
                            <div className="col-span-full bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-2xl shadow-sm p-8 text-center">
                                <HardDrive className="w-12 h-12 text-gray-300 dark:text-gray-600 mx-auto mb-4" />
                                <p className="text-gray-500 dark:text-gray-400">No boot assets found.</p>
                                <p className="text-sm text-gray-400 dark:text-gray-500 mt-1">Upload kernels, initrds, or bootloaders here.</p>
                            </div>
                        ) : (
                            assets.map((asset) => (
                                <div key={asset.id} className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl shadow-sm p-4 relative group">
                                    <div className="flex items-start gap-4">
                                        <div className="p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                                            <FileText className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <h4 className="font-medium text-gray-900 dark:text-white truncate" title={asset.name}>{asset.name}</h4>
                                            <p className="text-xs text-gray-500 dark:text-gray-400 truncate" title={asset.file_name}>{asset.file_name}</p>
                                            <div className="flex items-center gap-2 mt-2">
                                                <span className="text-xs font-mono bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded text-gray-600 dark:text-gray-300">
                                                    {asset.type}
                                                </span>
                                                <span className="text-xs text-gray-400">
                                                    {(asset.size / 1024 / 1024).toFixed(2)} MB
                                                </span>
                                            </div>
                                        </div>
                                        <button
                                            onClick={() => handleDeleteAsset(asset.id)}
                                            className="p-2 text-gray-400 hover:text-red-500 transition-colors opacity-0 group-hover:opacity-100 absolute top-2 right-2"
                                        >
                                            <Trash2 className="w-4 h-4" />
                                        </button>
                                    </div>
                                    <div className="mt-3 pt-3 border-t border-gray-100 dark:border-gray-700">
                                        <p className="text-xs font-mono text-gray-400 truncate select-all" title={asset.path}>
                                            /tftp/{asset.path}
                                        </p>
                                    </div>
                                </div>
                            ))
                        )}
                    </div>
                </div>
            )}
        </div>
    );
};

// Config Editor Component
const ConfigEditor: React.FC<{ config: DHCPConfig | null; assets: BootAsset[]; onSave: (c: DHCPConfig) => void; onCancel: () => void }> = ({ config, assets, onSave, onCancel }) => {
    // Guard against null config
    const safeConfig = config || {
        id: 0,
        name: '',
        interface: '',
        range_start: '',
        range_end: '',
        subnet_mask: '',
        gateway: '',
        dns_server: '',
        lease_time: '',
        domain: '',
        tftp_server: '',
        boot_file: '',
        next_server: '',
        boot_server_ip: '',
        mirror_url: '',
        kernel_params: '',
        is_active: false,
        enable_pxe: false,
        created_at: '',
        updated_at: '',
    };

    const [form, setForm] = useState({
        ...safeConfig,
        interface: safeConfig.interface || '',
        range_start: safeConfig.range_start || '',
        range_end: safeConfig.range_end || '',
        subnet_mask: safeConfig.subnet_mask || '',
        gateway: safeConfig.gateway || '',
        dns_server: safeConfig.dns_server || '',
        lease_time: safeConfig.lease_time || '',
        domain: safeConfig.domain || '',
        tftp_server: safeConfig.tftp_server || '',
        boot_file: safeConfig.boot_file || '',
        next_server: safeConfig.next_server || '',
        boot_server_ip: safeConfig.boot_server_ip || '',
        mirror_url: safeConfig.mirror_url || '',
        kernel_params: safeConfig.kernel_params || '',
        is_active: safeConfig.is_active || false,
        enable_pxe: safeConfig.enable_pxe || false,
    });

    if (!config) {
        return null;
    }

    const [interfaces, setInterfaces] = useState<NetworkInterface[]>([]);

    useEffect(() => {
        const loadInterfaces = async () => {
            try {
                const ifaces = await SystemApi.listInterfaces();
                setInterfaces(ifaces);
            } catch (e) {
                console.error("Failed to load interfaces", e);
            }
        };
        loadInterfaces();
    }, []);

    return (
        <div className="p-6 bg-gray-50 dark:bg-gray-900/50 space-y-4">
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Interface</label>
                    <select
                        value={form.interface || ''}
                        onChange={(e) => setForm({ ...form, interface: e.target.value })}
                        className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm"
                    >
                        <option value="">Any / Host (0.0.0.0)</option>
                        {interfaces.map((iface) => (
                            <option key={iface.name} value={iface.name}>
                                {iface.name} ({iface.ip_addresses[0] || 'No IP'})
                            </option>
                        ))}
                    </select>
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Name</label>
                    <input type="text" value={form.name || ''} onChange={(e) => setForm({ ...form, name: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm" />
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Range Start</label>
                    <input type="text" value={form.range_start || ''} onChange={(e) => setForm({ ...form, range_start: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm" />
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Range End</label>
                    <input type="text" value={form.range_end || ''} onChange={(e) => setForm({ ...form, range_end: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm" />
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Subnet Mask</label>
                    <input type="text" value={form.subnet_mask || ''} onChange={(e) => setForm({ ...form, subnet_mask: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm" />
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Gateway</label>
                    <input type="text" value={form.gateway || ''} onChange={(e) => setForm({ ...form, gateway: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm" />
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">DNS Server</label>
                    <input type="text" value={form.dns_server || ''} onChange={(e) => setForm({ ...form, dns_server: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm" />
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Lease Time</label>
                    <input type="text" value={form.lease_time || ''} onChange={(e) => setForm({ ...form, lease_time: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm" />
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Domain</label>
                    <input type="text" value={form.domain || ''} onChange={(e) => setForm({ ...form, domain: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm" />
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Boot File</label>
                    <input
                        type="text"
                        value={form.boot_file || ''}
                        onChange={(e) => setForm({ ...form, boot_file: e.target.value })}
                        list="boot-assets-list"
                        placeholder="undionly.kpxe or select asset"
                        className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm"
                    />
                    <datalist id="boot-assets-list">
                        <option value="undionly.kpxe" />
                        <option value="ipxe.efi" />
                        {assets.map(a => (
                            <option key={a.id} value={`/tftp/${a.path}`}>{a.name}</option>
                        ))}
                    </datalist>
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Next Server</label>
                    <input type="text" value={form.next_server || ''} onChange={(e) => setForm({ ...form, next_server: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm" />
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">TFTP Server</label>
                    <input type="text" value={form.tftp_server || ''} onChange={(e) => setForm({ ...form, tftp_server: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm" />
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Boot Server IP (HTTP)</label>
                    <input type="text" value={form.boot_server_ip || ''} placeholder="Nginx/API IP" onChange={(e) => setForm({ ...form, boot_server_ip: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm" />
                </div>
                <div className="col-span-2">
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Debian 镜像源（留空使用默认官方源）</label>
                    <input 
                        type="text" 
                        value={form.mirror_url || ''} 
                        placeholder="选择预设镜像或输入自定义URL" 
                        onChange={(e) => setForm({ ...form, mirror_url: e.target.value })} 
                        list="mirror-presets"
                        className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm font-mono"
                    />
                    <datalist id="mirror-presets">
                        <option value="http://deb.debian.org/debian" label="官方源 (默认)" />
                        <option value="https://mirrors.aliyun.com/debian" label="阿里云" />
                        <option value="https://mirrors.tuna.tsinghua.edu.cn/debian" label="清华大学" />
                        <option value="https://mirrors.ustc.edu.cn/debian" label="中科大" />
                        <option value="https://mirrors.163.com/debian" label="网易" />
                        <option value="https://mirrors.huaweicloud.com/debian" label="华为云" />
                        <option value="https://repo.huaweicloud.com/debian" label="华为云（备用）" />
                    </datalist>
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">用于PXE网络安装的Debian软件包镜像源，留空则使用官方源</p>
                </div>
                <div className="col-span-2">
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Kernel Params</label>
                    <input type="text" value={form.kernel_params || ''} onChange={(e) => setForm({ ...form, kernel_params: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm font-mono" />
                </div>
                <div className="flex items-end gap-4">
                    <label className="flex items-center gap-2 cursor-pointer">
                        <input type="checkbox" checked={form.is_active || false} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} className="rounded border-gray-300 dark:border-gray-600" />
                        <span className="text-sm text-gray-700 dark:text-gray-300">Active</span>
                    </label>
                    <label className="flex items-center gap-2 cursor-pointer">
                        <input type="checkbox" checked={form.enable_pxe || false} onChange={(e) => setForm({ ...form, enable_pxe: e.target.checked })} className="rounded border-gray-300 dark:border-gray-600" />
                        <span className="text-sm text-gray-700 dark:text-gray-300">PXE</span>
                    </label>
                </div>
            </div>
            <div className="flex gap-2">
                <button onClick={() => onSave(form)} className="flex items-center gap-1 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors text-sm">
                    <Save className="w-4 h-4" /> Save
                </button>
                <button onClick={onCancel} className="flex items-center gap-1 px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors text-sm">
                    <X className="w-4 h-4" /> Cancel
                </button>
            </div>
        </div>
    );
};

// Reservation Editor Component
const ReservationEditor: React.FC<{ reservation: DHCPReservation; onSave: (r: DHCPReservation) => void; onCancel: () => void }> = ({ reservation, onSave, onCancel }) => {
    const [form, setForm] = useState(reservation);

    return (
        <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-2xl shadow-sm p-6 space-y-4">
            <h4 className="font-bold text-gray-900 dark:text-white">{reservation.id ? 'Edit Reservation' : 'New Reservation'}</h4>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">MAC Address</label>
                    <input type="text" placeholder="aa:bb:cc:dd:ee:ff" value={form.mac_address} onChange={(e) => setForm({ ...form, mac_address: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white text-sm font-mono" />
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">IP Address</label>
                    <input type="text" placeholder="192.168.10.50" value={form.ip_address} onChange={(e) => setForm({ ...form, ip_address: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white text-sm font-mono" />
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Hostname</label>
                    <input type="text" placeholder="node-01" value={form.hostname} onChange={(e) => setForm({ ...form, hostname: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white text-sm" />
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Description</label>
                    <input type="text" placeholder="Optional" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white text-sm" />
                </div>
            </div>
            <div className="flex items-center gap-4">
                <label className="flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" checked={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} className="rounded border-gray-300 dark:border-gray-600" />
                    <span className="text-sm text-gray-700 dark:text-gray-300">Active</span>
                </label>
            </div>
            <div className="flex gap-2">
                <button onClick={() => onSave(form)} className="flex items-center gap-1 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors text-sm">
                    <Save className="w-4 h-4" /> Save
                </button>
                <button onClick={onCancel} className="flex items-center gap-1 px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors text-sm">
                    <X className="w-4 h-4" /> Cancel
                </button>
            </div>
        </div>
    );
};

const AssetUploader: React.FC<{ onUpload: (file: File, type: string, name?: string) => void; onCancel: () => void }> = ({ onUpload, onCancel }) => {
    const [file, setFile] = useState<File | null>(null);
    const [type, setType] = useState('kernel');
    const [name, setName] = useState('');
    const [uploading, setUploading] = useState(false);

    const handleUpload = () => {
        if (!file) return;
        setUploading(true);
        onUpload(file, type, name);
    };

    return (
        <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-2xl shadow-sm p-6 mb-6">
            <h4 className="font-bold text-gray-900 dark:text-white mb-4">Upload Boot Asset</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Asset Type</label>
                    <select
                        value={type}
                        onChange={(e) => setType(e.target.value)}
                        className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white text-sm"
                    >
                        <option value="kernel">Kernel (vmlinuz)</option>
                        <option value="initrd">Initrd / Initramfs</option>
                        <option value="bootloader">Bootloader (iPXE/Grub)</option>
                        <option value="config">Configuration / Script</option>
                        <option value="other">Other</option>
                    </select>
                </div>
                <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Display Name (Optional)</label>
                    <input
                        type="text"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        placeholder="e.g. Linux Kernel 6.1"
                        className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white text-sm"
                    />
                </div>
                <div className="col-span-full">
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">File</label>
                    <div className="flex items-center justify-center w-full">
                        <label className="flex flex-col items-center justify-center w-full h-32 border-2 border-gray-300 border-dashed rounded-lg cursor-pointer bg-gray-50 dark:hover:bg-gray-800 dark:bg-gray-700 hover:bg-gray-100 dark:border-gray-600 dark:hover:border-gray-500">
                            <div className="flex flex-col items-center justify-center pt-5 pb-6">
                                <Upload className="w-8 h-8 mb-4 text-gray-500 dark:text-gray-400" />
                                <p className="mb-2 text-sm text-gray-500 dark:text-gray-400"><span className="font-semibold">Click to upload</span> or drag and drop</p>
                                <p className="text-xs text-gray-500 dark:text-gray-400">{file ? file.name : "Select a file"}</p>
                            </div>
                            <input type="file" className="hidden" onChange={(e) => setFile(e.target.files ? e.target.files[0] : null)} />
                        </label>
                    </div>
                </div>
            </div>
            <div className="flex gap-2">
                <button
                    onClick={handleUpload}
                    disabled={!file || uploading}
                    className="flex items-center gap-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    {uploading ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Upload className="w-4 h-4" />} Upload
                </button>
                <button onClick={onCancel} className="flex items-center gap-1 px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors text-sm">
                    <X className="w-4 h-4" /> Cancel
                </button>
            </div>
        </div>
    );
};
