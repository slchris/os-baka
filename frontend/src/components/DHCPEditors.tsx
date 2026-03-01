import React, { useEffect, useState } from 'react';
import { Save, X, Upload, RefreshCw } from 'lucide-react';
import { DHCPConfig, DHCPReservation, SystemApi, NetworkInterface, BootAsset } from '../services/apiClient';

// Config Editor Component
export const ConfigEditor: React.FC<{ config: DHCPConfig | null; assets: BootAsset[]; onSave: (c: DHCPConfig) => void; onCancel: () => void }> = ({ config, assets, onSave, onCancel }) => {
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
export const ReservationEditor: React.FC<{ reservation: DHCPReservation; onSave: (r: DHCPReservation) => void; onCancel: () => void }> = ({ reservation, onSave, onCancel }) => {
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

// Asset Uploader Component
export const AssetUploader: React.FC<{ onUpload: (file: File, type: string, name?: string) => void; onCancel: () => void }> = ({ onUpload, onCancel }) => {
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
