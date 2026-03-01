
import React, { useState, useRef, useEffect } from 'react';
import { Upload, RefreshCcw, HardDrive, Shield, Plus, Download, Bot, Trash2, Filter, Hash, Network, Activity, Cpu, Laptop, Save, X, Eye, EyeOff, Pencil, Zap, Loader2, CheckCircle2, AlertCircle, ChevronDown } from 'lucide-react';
import { NodeConfig, NodeStatus, GeminiAnalysisResult, KeySlot } from '../types';
import { analyzeCsvData } from '../services/geminiService';
import { NodesApi, NodeCreateRequest } from '../services/apiClient';

export const Nodes: React.FC = () => {
  const [nodes, setNodes] = useState<NodeConfig[]>([]);
  const [filterStatus, setFilterStatus] = useState('ALL');

  const fileInputRef = useRef<HTMLInputElement>(null);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [analysisResult, setAnalysisResult] = useState<GeminiAnalysisResult | null>(null);

  // Load nodes on mount
  useEffect(() => {
    loadNodes();
  }, []);

  // Auto-refresh nodes every 30 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      loadNodes();
    }, 30000); // 30 seconds

    return () => clearInterval(interval);
  }, []);

  const loadNodes = async () => {
    try {
      const res = await NodesApi.list();
      // Map backend response to frontend NodeConfig type
      // The backend response is simpler than NodeConfig, so we map defaults
      const mapped: NodeConfig[] = res.items.map(n => {
        // Debug: log the status mapping
        const originalStatus = n.status;
        const mappedStatus = (n.status?.toUpperCase() as NodeStatus) || NodeStatus.PENDING;
        console.log(`Node ${n.hostname}: ${originalStatus} -> ${mappedStatus}`);
        
        return {
          id: n.id.toString(),
          hostname: n.hostname,
          ipAddress: n.ip_address,
          macAddress: n.mac_address,
          status: mappedStatus,
          provisioningMethod: 'PXE_MAC', // Backend doesn't store this explicitly yet, assuming PXE
          encryption: {
            enabled: n.encryption_enabled,
            luksVersion: 'luks2',
            tpmEnabled: n.tpm_enabled ?? true,
            usbKeyRequired: n.usb_key_required ?? false,
            pcrBinding: n.pcr_binding ? n.pcr_binding.split(',').filter(Boolean).map(x => parseInt(x, 10)).filter(x => !Number.isNaN(x)) : [7],
            keySlots: []
          },
          lastSeen: n.last_seen || 'Never',
          // ipmi: n.ipmi // Backend missing IPMI fields currently
          osType: n.os_type || 'ubuntu',
          osVersion: n.os_version || '12',
          mirrorUrl: n.mirror_url || '',
          timezone: n.timezone || 'UTC'
        };
      });
      setNodes(mapped);
    } catch (e) {
      console.error("Failed to load nodes", e);
    }
  };

  // Tooltip State
  const [tooltipNode, setTooltipNode] = useState<NodeConfig | null>(null);
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });

  // Add/Edit Node Modal State
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [editingNodeId, setEditingNodeId] = useState<string | null>(null);

  const [addMode, setAddMode] = useState<'PXE_MAC' | 'IPMI_BMC'>('PXE_MAC');
  const [newHostname, setNewHostname] = useState('');
  const [newIp, setNewIp] = useState('');
  const [newMac, setNewMac] = useState('');
  const [rootPassword, setRootPassword] = useState('');
  const [showRootPassword, setShowRootPassword] = useState(false);
  const [sshEnabled, setSshEnabled] = useState(true);
  const [sshRootLogin, setSshRootLogin] = useState(false);
  const [encryptionPassphrase, setEncryptionPassphrase] = useState('');
  const [showPassphrase, setShowPassphrase] = useState(false);
  const [osType, setOsType] = useState('debian');
  const [osVersion, setOsVersion] = useState('12');
  const [mirrorUrl, setMirrorUrl] = useState('');
  const [timezone, setTimezone] = useState('UTC');
  const [encEnabled, setEncEnabled] = useState(true);
  const [tpmEnabled, setTpmEnabled] = useState(true);
  const [usbKeyRequired, setUsbKeyRequired] = useState(false);
  const [pcrBinding, setPcrBinding] = useState<number[]>([7]); // Default to PCR 7
  const [showAdvanced, setShowAdvanced] = useState(false); // Advanced settings toggle

  // IPMI State
  const [ipmiIp, setIpmiIp] = useState('');
  const [ipmiUser, setIpmiUser] = useState('');
  const [ipmiPass, setIpmiPass] = useState('');
  const [showIpmiPass, setShowIpmiPass] = useState(false);
  const [ipmiAllowUntrusted, setIpmiAllowUntrusted] = useState(false);
  const [ipmiTestStatus, setIpmiTestStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle');

  const refreshNodes = () => {
    loadNodes();
  };

  const handleNodeEnter = (e: React.MouseEvent, node: NodeConfig) => {
    setTooltipNode(node);
    setTooltipPos({ x: e.clientX, y: e.clientY });
  };

  const createDefaultKeySlots = (tpmEnabled: boolean): KeySlot[] => {
    const slots: KeySlot[] = [];
    for (let i = 0; i < 8; i++) {
      slots.push({ index: i, active: false });
    }

    // Setup default Slot 0 for TPM if enabled, Slot 1 for Recovery
    if (tpmEnabled) {
      slots[0] = { index: 0, active: true, type: 'tpm', label: 'TPM2 Auto-Unlock' };
    }
    slots[1] = { index: 1, active: true, type: 'recovery', label: 'Initial Recovery Key' };
    return slots;
  };

  const buildRandomPassphrase = () => {
    const alphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789@#$%';
    const bytes = new Uint8Array(24);
    crypto.getRandomValues(bytes);
    let result = '';
    bytes.forEach(b => {
      result += alphabet[b % alphabet.length];
    });
    return result;
  };

  const generatePassphrase = () => {
    const pass = buildRandomPassphrase();
    setEncryptionPassphrase(pass);
    setShowPassphrase(false);
    return pass;
  };

  const handleFileUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setIsAnalyzing(true);
    const reader = new FileReader();
    reader.onload = async (e) => {
      const text = e.target?.result as string;

      // AI Analysis
      const aiResult = await analyzeCsvData(text);
      setAnalysisResult(aiResult);
      setIsAnalyzing(false);

      if (aiResult.isValid || window.confirm("AI detected issues. Import anyway?")) {
        const lines = text.split('\n');
        // Skip header
        const startIdx = lines[0].toLowerCase().includes('mac') ? 1 : 0;

        let successCount = 0;

        for (let i = startIdx; i < lines.length; i++) {
          const line = lines[i].trim();
          if (!line) continue;

          const parts = line.split(',').map(s => s.trim());
          // Expected: MAC, IP, Hostname, (AssetTag optional)
          if (parts.length < 3) continue;

          const [mac, ip, hostname, assetTag] = parts;

          try {
            await NodesApi.create({
              mac_address: mac,
              ip_address: ip,
              hostname: hostname,
              asset_tag: assetTag || '',
              status: 'installing',
              os_type: 'debian',
              os_version: '12',
              mirror_url: '',
              root_password: 'changeme',
              ssh_enabled: true,
              ssh_root_login: false,
              encryption_enabled: true,
              encryption_passphrase: buildRandomPassphrase(),
              tpm_enabled: true,
              usb_key_required: false
            });
            successCount++;
          } catch (err) {
            console.error(`Failed to import line ${i + 1}: ${line}`, err);
          }
        }

        if (successCount > 0) {
          alert(`Imported ${successCount} nodes successfully.`);
          refreshNodes();
        } else {
          alert("No nodes imported. Check CSV format.");
        }
      }
    };
    reader.readAsText(file);
  };

  const handleEdit = (node: NodeConfig) => {
    setEditingNodeId(node.id);
    setNewHostname(node.hostname);
    setNewIp(node.ipAddress);
    setAddMode(node.provisioningMethod);
    setEncryptionPassphrase('');
    setShowPassphrase(false);
    setOsType(node.osType || 'debian');
    setOsVersion(node.osVersion || '12');
    setMirrorUrl(node.mirrorUrl || '');
    setTimezone(node.timezone || 'UTC');
    setEncEnabled(node.encryption.enabled);
    setTpmEnabled(node.encryption.tpmEnabled);
    setUsbKeyRequired(node.encryption.usbKeyRequired);
    setPcrBinding(node.encryption.pcrBinding || [7]);

    if (node.provisioningMethod === 'PXE_MAC') {
      setNewMac(node.macAddress);
      setIpmiIp('');
      setIpmiUser('');
      setIpmiPass('');
      setIpmiAllowUntrusted(false);
    } else {
      setNewMac(node.macAddress);
      setIpmiIp(node.ipmi?.ip || '');
      setIpmiUser(node.ipmi?.username || '');
      setIpmiPass(''); // Don't populate password for security, user can overwrite
      setIpmiAllowUntrusted(node.ipmi?.allowUntrustedCerts || false);
    }
    setIpmiTestStatus('idle');
    setIsAddModalOpen(true);
  };

  const handleManualSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const trimmedPass = encryptionPassphrase.trim();
    const passForCreate = trimmedPass || '';

    if (editingNodeId) {
      // Update Existing Node
      const existingNode = nodes.find(n => n.id === editingNodeId);
      if (!existingNode) return;

      try {
        await NodesApi.update(parseInt(editingNodeId), {
          hostname: newHostname,
          ip_address: newIp,
          mac_address: newMac,
          // provisioningMethod: addMode, // Pending backend support
          status: existingNode.status,
          os_type: osType,
          os_version: osVersion,
          mirror_url: mirrorUrl,
          timezone: timezone,
          encryption_enabled: encEnabled,
          encryption_passphrase: encEnabled ? (trimmedPass || undefined) : undefined,
          tpm_enabled: encEnabled ? tpmEnabled : false,
          usb_key_required: encEnabled ? usbKeyRequired : false,
          pcr_binding: encEnabled && tpmEnabled ? pcrBinding.join(',') : undefined
        });
        refreshNodes();
        resetModal();
        setIsAddModalOpen(false);
      } catch (err) {
        console.error("Failed to update node", err);
        alert("Failed to update node");
      }
    } else {
      // Create New Node
      if (!passForCreate) {
        alert("请先生成或填写磁盘加密密码");
        return;
      }
      try {
        await NodesApi.create({
          hostname: newHostname,
          ip_address: newIp,
          mac_address: newMac || '00:00:00:00:00:00',
          asset_tag: '',
          status: 'installing',
          os_type: osType,
          os_version: osVersion,
          mirror_url: mirrorUrl,
          timezone: timezone,
          root_password: rootPassword || 'changeme',
          ssh_enabled: sshEnabled,
          ssh_root_login: sshRootLogin,
          encryption_enabled: encEnabled,
          encryption_passphrase: encEnabled ? passForCreate : undefined,
          tpm_enabled: encEnabled ? tpmEnabled : false,
          usb_key_required: encEnabled ? usbKeyRequired : false,
          pcr_binding: encEnabled && tpmEnabled ? pcrBinding.join(',') : undefined
        });
        refreshNodes();
        resetModal();
        setIsAddModalOpen(false);
      } catch (err) {
        console.error("Failed to create node", err);
        alert("Failed to create node");
      }
    }
  };

  const resetModal = () => {
    setEditingNodeId(null);
    setNewHostname('');
    setNewIp('');
    setNewMac('');
    setRootPassword('');
    setShowRootPassword(false);
    setSshEnabled(true);
    setSshRootLogin(false);
    setEncryptionPassphrase('');
    setShowPassphrase(false);
    setOsType('debian');
    setOsVersion('12');
    setMirrorUrl('');
    setTimezone('UTC');
    setEncEnabled(true);
    setTpmEnabled(true);
    setUsbKeyRequired(false);
    setPcrBinding([7]);
    setShowAdvanced(false);
    setIpmiIp('');
    setIpmiUser('');
    setIpmiPass('');
    setIpmiAllowUntrusted(false);
    setIpmiTestStatus('idle');
    setAddMode('PXE_MAC');
  }

  const handleDelete = async (id: string) => {
    if (confirm("Are you sure you want to delete this node?")) {
      try {
        await NodesApi.delete(parseInt(id));
        refreshNodes();
      } catch (err) {
        console.error("Failed to delete node", err);
        alert("Failed to delete node");
      }
    }
  }

  const handleDownloadPassphrase = async (node: NodeConfig) => {
    try {
      const res = await NodesApi.getPassphrase(parseInt(node.id));
      const blob = new Blob([
        `hostname: ${node.hostname}\n`,
        `mac: ${node.macAddress}\n`,
        `passphrase: ${res.passphrase}\n`
      ], { type: 'text/plain' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${node.hostname}-luks-passphrase.txt`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error("Failed to download passphrase", err);
      alert("无法获取该节点的加密密码，请确认已在后台存储并开启加密。");
    }
  }

  const testIpmiConnection = () => {
    if (!ipmiIp || !ipmiUser) return;
    setIpmiTestStatus('loading');

    // Simulate network call
    setTimeout(() => {
      // Mock logic: Success if IP is not empty and doesn't simulate error condition
      // In real app, this would call backend with credentials
      const isIp = /^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/.test(ipmiIp);
      if (isIp) {
        setIpmiTestStatus('success');
      } else {
        setIpmiTestStatus('error');
      }
    }, 1500);
  };

  const downloadTemplate = () => {
    const headers = "MAC Address,IP Address,Hostname,Asset Tag";
    const example = "00:11:22:33:44:55,192.168.10.50,worker-01,ASSET-001";
    const blob = new Blob([headers + "\n" + example], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'nodes_template.csv';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  const rebuildNode = async (node: NodeConfig) => {
    if (!confirm(`Are you sure you want to rebuild ${node.hostname}? This will reinstall the OS on next boot.`)) {
      return;
    }

    try {
      const result = await NodesApi.rebuild(parseInt(node.id));
      alert(result.message || 'Node rebuild initiated successfully');
      refreshNodes();
    } catch (err) {
      console.error("Failed to rebuild node", err);
      alert("Failed to initiate node rebuild");
    }
  };

  const filteredNodes = filterStatus === 'ALL'
    ? nodes
    : nodes.filter(node => node.status === filterStatus);

  return (
    <>
      {/* Floating Tooltip - Outside container to prevent layout shift */}
      {tooltipNode && (
        <div
          className="fixed z-50 bg-gray-900/95 backdrop-blur-md text-white p-4 rounded-xl shadow-2xl pointer-events-none border border-gray-700 min-w-[280px] animate-in fade-in duration-200"
          style={{
            top: Math.min(tooltipPos.y + 16, window.innerHeight - 200),
            left: Math.min(tooltipPos.x + 16, window.innerWidth - 300)
          }}
        >
          <div className="flex items-center justify-between border-b border-gray-700 pb-2 mb-3">
            <span className="font-bold text-base flex items-center gap-2">
              <Cpu className="w-4 h-4 text-blue-400" />
              {tooltipNode.hostname}
            </span>
            <span className={`px-2 py-0.5 rounded-full text-[10px] uppercase font-bold tracking-wider ${tooltipNode.status === NodeStatus.ACTIVE ? 'bg-green-500/20 text-green-400' : 'bg-gray-700 text-gray-300'
              }`}>
              {tooltipNode.status}
            </span>
          </div>

          <div className="space-y-2 text-sm text-gray-300 font-mono">
            <div className="flex items-center gap-3">
              <Hash className="w-4 h-4 text-gray-500" />
              <span className="text-gray-500">ID:</span>
              <span className="text-white">{tooltipNode.id}</span>
            </div>
            <div className="flex items-center gap-3">
              <Network className="w-4 h-4 text-gray-500" />
              <span className="text-gray-500">IP:</span>
              <span className="text-white">{tooltipNode.ipAddress}</span>
            </div>
            <div className="flex items-center gap-3">
              <div className="w-4" /> {/* Spacer alignment */}
              <span className="text-gray-500">MAC:</span>
              <span className="text-white">{tooltipNode.macAddress}</span>
            </div>
            {tooltipNode.ipmi && (
              <div className="flex items-center gap-3 text-yellow-500">
                <Laptop className="w-4 h-4" />
                <span className="text-gray-500">IPMI:</span>
                <span className="text-white">{tooltipNode.ipmi.ip}</span>
              </div>
            )}

            <div className="pt-2 mt-2 border-t border-gray-700">
              <div className="flex items-center gap-2 mb-1">
                <Shield className="w-4 h-4 text-gray-500" />
                <span className="text-gray-400 font-sans text-xs font-semibold uppercase">Security Profile</span>
              </div>
              <div className="grid grid-cols-2 gap-x-4 gap-y-1 pl-6">
                <span className="text-gray-500">Encryption:</span>
                <span className={tooltipNode.encryption.enabled ? "text-green-400" : "text-red-400"}>
                  {tooltipNode.encryption.enabled ? 'Enabled' : 'Disabled'}
                </span>

                {tooltipNode.encryption.enabled && (
                  <>
                    <span className="text-gray-500">Method:</span>
                    <span>{tooltipNode.encryption.luksVersion.toUpperCase()}</span>

                    <span className="text-gray-500">TPM2:</span>
                    <span className={tooltipNode.encryption.tpmEnabled ? "text-green-400" : "text-gray-500"}>
                      {tooltipNode.encryption.tpmEnabled ? 'Active' : 'Inactive'}
                    </span>

                    <span className="text-gray-500">USB Key:</span>
                    <span className={tooltipNode.encryption.usbKeyRequired ? "text-blue-400" : "text-gray-500"}>
                      {tooltipNode.encryption.usbKeyRequired ? 'Required' : 'Optional'}
                    </span>
                  </>
                )}
              </div>
            </div>

            <div className="pt-2 mt-2 border-t border-gray-700 flex items-center gap-2 text-xs">
              <Activity className="w-3 h-3 text-gray-500" />
              <span className="text-gray-500">Last Seen:</span>
              <span className="text-gray-300">{tooltipNode.lastSeen || 'Unknown'}</span>
            </div>
          </div>
        </div>
      )}

      <div className="space-y-6 relative">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
          <div>
            <h2 className="text-2xl font-bold text-gray-900 dark:text-white">Node Management</h2>
            <p className="text-gray-500 dark:text-gray-400">Import targets, configure LUKS encryption, and bind networking.</p>
          </div>
          <div className="flex gap-3 flex-wrap">
            <div className="relative">
              <Filter className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 pointer-events-none" />
              <select
                value={filterStatus}
                onChange={(e) => setFilterStatus(e.target.value)}
                className="pl-10 pr-8 py-2 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-200 rounded-xl font-medium hover:bg-gray-50 dark:hover:bg-gray-700 shadow-sm transition-all appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/20 h-full"
              >
                <option value="ALL">All Statuses</option>
                {Object.values(NodeStatus).map((status) => (
                  <option key={status} value={status}>{status}</option>
                ))}
              </select>
            </div>

            <input
              type="file"
              accept=".csv"
              className="hidden"
              ref={fileInputRef}
              onChange={handleFileUpload}
            />
            <button
              onClick={() => fileInputRef.current?.click()}
              className="flex items-center gap-2 px-4 py-2 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-200 rounded-xl font-medium hover:bg-gray-50 dark:hover:bg-gray-700 shadow-sm transition-all"
            >
              <Upload className="w-4 h-4" />
              Import CSV
            </button>

            <button
              onClick={() => { resetModal(); generatePassphrase(); setIsAddModalOpen(true); }}
              className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-xl font-medium hover:bg-blue-700 shadow-lg shadow-blue-200 dark:shadow-blue-900/20 transition-all"
            >
              <Plus className="w-4 h-4" />
              Add Node
            </button>
          </div>
        </div>

        {/* Add/Edit Node Modal */}
        {isAddModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm animate-fade-in">
            <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl max-w-3xl w-full max-h-[90vh] overflow-hidden animate-in zoom-in-95 duration-200 flex flex-col">
              <div className="p-5 border-b border-gray-100 dark:border-gray-700 flex justify-between items-center bg-gray-50 dark:bg-gray-900 flex-shrink-0">
                <h3 className="font-bold text-gray-900 dark:text-white text-lg">
                  {editingNodeId ? 'Edit Configuration' : 'Add New Node'}
                </h3>
                <button onClick={() => setIsAddModalOpen(false)} className="p-2 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-full text-gray-500">
                  <X className="w-5 h-5" />
                </button>
              </div>

              <form onSubmit={handleManualSubmit} className="flex-1 overflow-y-auto">
                <div className="p-6 space-y-4">
                  {/* Provisioning Method */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Provisioning Method</label>
                    <div className="grid grid-cols-2 gap-3">
                      <button
                        type="button"
                        onClick={() => setAddMode('PXE_MAC')}
                        className={`p-3 rounded-lg border flex flex-col items-center gap-2 transition-all ${addMode === 'PXE_MAC' ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400' : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:bg-gray-50 dark:hover:bg-gray-700'}`}
                      >
                        <Network className="w-5 h-5" />
                        <span className="text-xs font-bold">PXE / MAC</span>
                      </button>
                      <button
                        type="button"
                        onClick={() => setAddMode('IPMI_BMC')}
                        className={`p-3 rounded-lg border flex flex-col items-center gap-2 transition-all ${addMode === 'IPMI_BMC' ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20 text-purple-600 dark:text-purple-400' : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:bg-gray-50 dark:hover:bg-gray-700'}`}
                      >
                        <Laptop className="w-5 h-5" />
                        <span className="text-xs font-bold">IPMI / BMC</span>
                      </button>
                    </div>
                  </div>

                  {/* Basic Info - Two columns */}
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Hostname</label>
                      <input required type="text" value={newHostname} onChange={e => setNewHostname(e.target.value)} className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/20" placeholder="worker-01" />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">IP Address</label>
                      <input required type="text" value={newIp} onChange={e => setNewIp(e.target.value)} className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/20" placeholder="192.168.x.x" />
                    </div>
                  </div>

                  {addMode === 'PXE_MAC' && (
                    <div>
                      <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">MAC Address</label>
                      <input required type="text" value={newMac} onChange={e => setNewMac(e.target.value)} className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/20" placeholder="00:11:22:33:44:55" />
                    </div>
                  )}

                  {/* OS Selection - Two columns */}
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">OS Type</label>
                      <select value={osType} onChange={e => setOsType(e.target.value)} className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/20">
                        <option value="debian">Debian</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Version</label>
                      <select value={osVersion} onChange={e => setOsVersion(e.target.value)} className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/20">
                        <option value="12">12 (Bookworm)</option>
                        <option value="11">11 (Bullseye)</option>
                      </select>
                    </div>
                  </div>

                  {/* Root Password */}
                  <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Root Password</label>
                    <div className="relative">
                      <input 
                        type={showRootPassword ? "text" : "password"}
                        value={rootPassword} 
                        onChange={e => setRootPassword(e.target.value)} 
                        className="w-full px-3 py-2 pr-10 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 font-mono" 
                        placeholder="留空使用默认: changeme" 
                      />
                      <button type="button" onClick={() => setShowRootPassword(!showRootPassword)} className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
                        {showRootPassword ? <EyeOff size={16} /> : <Eye size={16} />}
                      </button>
                    </div>
                  </div>

                  {/* SSH Options */}
                  <div className="flex items-center gap-4 p-3 bg-gray-50 dark:bg-gray-900/50 rounded-lg">
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input type="checkbox" checked={sshEnabled} onChange={e => setSshEnabled(e.target.checked)} className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500" />
                      <span className="text-xs font-medium text-gray-700 dark:text-gray-300">SSH</span>
                    </label>
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input type="checkbox" checked={sshRootLogin} onChange={e => setSshRootLogin(e.target.checked)} disabled={!sshEnabled} className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500" />
                      <span className="text-xs font-medium text-gray-700 dark:text-gray-300">Root Login</span>
                    </label>
                  </div>

                  {/* Encryption Options */}
                  <div className="space-y-3 p-3 border border-gray-200 dark:border-gray-700 rounded-lg">
                    <div className="flex items-center gap-4">
                      <label className="flex items-center gap-2 cursor-pointer">
                        <input type="checkbox" checked={encEnabled} onChange={e => setEncEnabled(e.target.checked)} className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500" />
                        <span className="text-xs font-medium text-gray-700 dark:text-gray-300">Encryption</span>
                      </label>
                      <label className="flex items-center gap-2 cursor-pointer">
                        <input type="checkbox" checked={tpmEnabled} onChange={e => setTpmEnabled(e.target.checked)} disabled={!encEnabled} className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500" />
                        <span className="text-xs font-medium text-gray-700 dark:text-gray-300">TPM2</span>
                      </label>
                      <label className="flex items-center gap-2 cursor-pointer">
                        <input type="checkbox" checked={usbKeyRequired} onChange={e => setUsbKeyRequired(e.target.checked)} disabled={!encEnabled} className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500" />
                        <span className="text-xs font-medium text-gray-700 dark:text-gray-300">USB Key</span>
                      </label>
                    </div>

                    {encEnabled && tpmEnabled && (
                      <div className="pt-2 border-t border-gray-200 dark:border-gray-700">
                        <p className="text-[10px] text-gray-600 dark:text-gray-400 mb-2">TPM2 PCR Bindings:</p>
                        <div className="grid grid-cols-2 gap-1.5">
                          {[
                            { id: 0, label: 'PCR 0: BIOS' },
                            { id: 2, label: 'PCR 2: ROM' },
                            { id: 4, label: 'PCR 4: Bootloader' },
                            { id: 7, label: 'PCR 7: SecureBoot' }
                          ].map(pcr => (
                            <label key={pcr.id} className="flex items-center gap-1.5 text-xs cursor-pointer">
                              <input
                                type="checkbox"
                                checked={pcrBinding.includes(pcr.id)}
                                onChange={(e) => {
                                  if (e.target.checked) {
                                    setPcrBinding([...pcrBinding, pcr.id].sort());
                                  } else {
                                    setPcrBinding(pcrBinding.filter(p => p !== pcr.id));
                                  }
                                }}
                                className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500"
                              />
                              <span className="text-gray-700 dark:text-gray-300">{pcr.label}</span>
                            </label>
                          ))}
                        </div>
                      </div>
                    )}

                    {encEnabled && (
                      <div className="pt-2 border-t border-gray-200 dark:border-gray-700">
                        <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">LUKS Passphrase</label>
                        <div className="flex items-center gap-2">
                          <input
                            required={encEnabled && !editingNodeId}
                            disabled={!encEnabled}
                            type={showPassphrase ? 'text' : 'password'}
                            value={encryptionPassphrase}
                            onChange={e => setEncryptionPassphrase(e.target.value)}
                            className="flex-1 px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 disabled:opacity-60 font-mono"
                            placeholder={encEnabled ? "自动生成或自定义" : "加密已关闭"}
                          />
                          <button type="button" onClick={generatePassphrase} disabled={!encEnabled} className="px-3 py-2 text-xs bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg border border-gray-200 dark:border-gray-600 text-gray-700 dark:text-gray-200 disabled:opacity-60">生成</button>
                          <button type="button" onClick={() => setShowPassphrase(!showPassphrase)} disabled={!encEnabled} className="p-2 text-gray-500 hover:text-gray-800 dark:hover:text-gray-200 rounded-lg border border-gray-200 dark:border-gray-700 disabled:opacity-60">
                            {showPassphrase ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                          </button>
                        </div>
                      </div>
                    )}
                  </div>

                  {/* Advanced Settings (Collapsible) */}
                  <div className="border-t border-gray-200 dark:border-gray-700 pt-3">
                    <button
                      type="button"
                      onClick={() => setShowAdvanced(!showAdvanced)}
                      className="flex items-center justify-between w-full text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100"
                    >
                      <span>高级设置</span>
                      <ChevronDown className={`w-4 h-4 transition-transform ${showAdvanced ? 'rotate-180' : ''}`} />
                    </button>
                    {showAdvanced && (
                      <div className="mt-3 space-y-3">
                        <div>
                          <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">镜像源</label>
                          <input
                            type="text"
                            value={mirrorUrl}
                            onChange={e => setMirrorUrl(e.target.value)}
                            list="node-mirror-presets"
                            placeholder="留空使用默认"
                            className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 font-mono"
                          />
                          <datalist id="node-mirror-presets">
                            <option value="">官方源</option>
                            <option value="http://mirrors.aliyun.com/debian">阿里云</option>
                            <option value="http://mirrors.tuna.tsinghua.edu.cn/debian">清华</option>
                            <option value="http://mirrors.ustc.edu.cn/debian">中科大</option>
                          </datalist>
                        </div>
                        <div>
                          <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">时区</label>
                          <input
                            type="text"
                            value={timezone}
                            onChange={e => setTimezone(e.target.value)}
                            list="timezone-presets"
                            placeholder="UTC"
                            className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 font-mono"
                          />
                          <datalist id="timezone-presets">
                            <option value="UTC">UTC</option>
                            <option value="Asia/Shanghai">Asia/Shanghai</option>
                            <option value="Asia/Hong_Kong">Asia/Hong_Kong</option>
                          </datalist>
                        </div>
                      </div>
                    )}
                  </div>

                  {/* IPMI Configuration */}
                  {addMode === 'IPMI_BMC' && (
                    <div className="pt-2 border-t border-gray-100 dark:border-gray-700 space-y-3">
                      <p className="text-xs font-bold text-purple-600 dark:text-purple-400">IPMI / BMC</p>
                      <div>
                        <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">BMC IP</label>
                        <input required type="text" value={ipmiIp} onChange={e => setIpmiIp(e.target.value)} className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500/20" placeholder="10.10.x.x" />
                      </div>
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Username</label>
                          <input required autoComplete="off" type="text" value={ipmiUser} onChange={e => setIpmiUser(e.target.value)} className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500/20" placeholder="ADMIN" />
                        </div>
                        <div>
                          <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Password</label>
                          <div className="relative">
                            <input
                              required={!editingNodeId}
                              autoComplete="new-password"
                              type={showIpmiPass ? "text" : "password"}
                              value={ipmiPass}
                              onChange={e => setIpmiPass(e.target.value)}
                              className="w-full px-3 py-2 pr-8 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500/20"
                              placeholder={editingNodeId ? "不变" : ""}
                            />
                            <button type="button" onClick={() => setShowIpmiPass(!showIpmiPass)} className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400">
                              {showIpmiPass ? <EyeOff className="w-3 h-3" /> : <Eye className="w-3 h-3" />}
                            </button>
                          </div>
                        </div>
                      </div>
                      <label className="flex items-start gap-2 cursor-pointer">
                        <input type="checkbox" checked={ipmiAllowUntrusted} onChange={(e) => setIpmiAllowUntrusted(e.target.checked)} className="mt-1 rounded border-gray-300 dark:border-gray-600 text-purple-600 focus:ring-purple-500" />
                        <span className="text-xs text-gray-600 dark:text-gray-400">Allow Untrusted Certs</span>
                      </label>
                      <div>
                        <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Data MAC (Optional)</label>
                        <input type="text" value={newMac} onChange={e => setNewMac(e.target.value)} className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/20" placeholder="00:11:22:33:44:55" />
                      </div>
                      <div className="flex items-center justify-between pt-2 border-t border-gray-100 dark:border-gray-700">
                        <div className="text-xs flex items-center gap-1">
                          {ipmiTestStatus === 'loading' && <><Loader2 className="w-3 h-3 animate-spin" /> Testing...</>}
                          {ipmiTestStatus === 'success' && <><CheckCircle2 className="w-3 h-3 text-green-600" /> OK</>}
                          {ipmiTestStatus === 'error' && <><AlertCircle className="w-3 h-3 text-red-600" /> Failed</>}
                        </div>
                        <button type="button" onClick={testIpmiConnection} disabled={ipmiTestStatus === 'loading' || !ipmiIp || !ipmiUser} className="text-xs flex items-center gap-1 px-3 py-1.5 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 rounded-lg transition-colors disabled:opacity-50">
                          <Zap className="w-3 h-3" /> Test
                        </button>
                      </div>
                    </div>
                  )}
                </div>

                {/* Fixed Footer with Buttons */}
                <div className="p-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900 flex justify-end gap-2 flex-shrink-0">
                  <button type="button" onClick={() => setIsAddModalOpen(false)} className="px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors">Cancel</button>
                  <button type="submit" className="px-4 py-2 text-sm bg-black dark:bg-blue-600 text-white rounded-lg hover:bg-gray-800 dark:hover:bg-blue-700 transition-colors flex items-center gap-2">
                    <Save className="w-4 h-4" /> {editingNodeId ? 'Update' : 'Save'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {isAnalyzing && (
          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-100 dark:border-blue-800 p-4 rounded-xl flex items-center gap-3 animate-pulse">
            <Bot className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <span className="text-blue-700 dark:text-blue-300 font-medium">AI is analyzing your network configuration for conflicts...</span>
          </div>
        )}

        {analysisResult && !isAnalyzing && (
          <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 p-4 rounded-xl shadow-sm space-y-2">
            <h4 className="font-semibold flex items-center gap-2 text-gray-900 dark:text-white">
              <Bot className="w-4 h-4 text-purple-600 dark:text-purple-400" /> AI Analysis Report
            </h4>
            {analysisResult.suggestions.length > 0 && (
              <div className="text-sm text-gray-600 dark:text-gray-300">
                <strong>Suggestions:</strong>
                <ul className="list-disc list-inside mt-1">
                  {analysisResult.suggestions.map((s, i) => <li key={i}>{s}</li>)}
                </ul>
              </div>
            )}
            {analysisResult.risks.length > 0 && (
              <div className="text-sm text-red-600 dark:text-red-400">
                <strong>Risks Detected:</strong>
                <ul className="list-disc list-inside mt-1">
                  {analysisResult.risks.map((s, i) => <li key={i}>{s}</li>)}
                </ul>
              </div>
            )}
          </div>
        )}

        <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-2xl shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-gray-50 dark:bg-gray-900/50 border-b border-gray-100 dark:border-gray-700 text-xs uppercase text-gray-500 dark:text-gray-400 font-semibold tracking-wider">
                  <th className="px-6 py-4">Hostname</th>
                  <th className="px-6 py-4">Provisioning</th>
                  <th className="px-6 py-4">Encryption</th>
                  <th className="px-6 py-4">Status</th>
                  <th className="px-6 py-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {filteredNodes.map((node) => (
                  <tr
                    key={node.id}
                    className="hover:bg-gray-50/50 dark:hover:bg-gray-700/30 transition-colors cursor-default"
                    onMouseLeave={() => setTooltipNode(null)}
                  >
                    <td className="px-6 py-4" onMouseEnter={(e) => handleNodeEnter(e, node)}>
                      <div className="font-medium text-gray-900 dark:text-white">{node.hostname}</div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">OS: {node.osType || 'unknown'} {node.osVersion ? `(${node.osVersion})` : ''}</div>
                    </td>
                    <td className="px-6 py-4" onMouseEnter={(e) => handleNodeEnter(e, node)}>
                      {node.provisioningMethod === 'IPMI_BMC' ? (
                        <div className="flex items-center gap-2 text-purple-700 dark:text-purple-400">
                          <Laptop className="w-4 h-4" />
                          <div>
                            <div className="text-sm font-medium">IPMI</div>
                            <div className="text-xs opacity-75 font-mono">
                              {node.ipmi?.username ? `${node.ipmi.username}@` : ''}{node.ipmi?.ip || 'N/A'}
                            </div>
                          </div>
                        </div>
                      ) : (
                        <div className="text-sm text-gray-700 dark:text-gray-300 space-y-1">
                          <div className="font-mono text-xs">MAC: {node.macAddress || 'N/A'}</div>
                          <div className="font-mono text-xs">IP: {node.ipAddress || 'N/A'}</div>
                        </div>
                      )}
                    </td>
                    <td className="px-6 py-4" onMouseEnter={(e) => handleNodeEnter(e, node)}>
                      <div className="flex items-center gap-2">
                        {node.encryption.enabled ? (
                          <span className="inline-flex items-center gap-1 px-2 py-1 rounded-md bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400 text-xs font-medium border border-green-100 dark:border-green-900/30">
                            Encrypted
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1 px-2 py-1 rounded-md bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 text-xs font-medium border border-red-100 dark:border-red-900/30">
                            Unencrypted
                          </span>
                        )}
                        {node.encryption.tpmEnabled && (
                          <span className="px-2 py-1 rounded-md bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 text-xs font-medium border border-gray-200 dark:border-gray-600">TPM2</span>
                        )}
                        {node.encryption.usbKeyRequired && (
                          <span className="px-2 py-1 rounded-md bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-400 text-xs font-medium border border-blue-100 dark:border-blue-900/30">USB</span>
                        )}
                        {node.encryption.pcrBinding?.length ? (
                          <span className="px-2 py-1 rounded-md bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-300 text-xs font-medium border border-amber-100 dark:border-amber-900/30">PCRs {node.encryption.pcrBinding.join(',')}</span>
                        ) : null}
                      </div>
                    </td>
                    <td className="px-6 py-4" onMouseEnter={(e) => handleNodeEnter(e, node)}>
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium
                        ${node.status === NodeStatus.ACTIVE ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-400' : ''}
                        ${node.status === NodeStatus.REBUILDING ? 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-400 animate-pulse' : ''}
                        ${node.status === NodeStatus.PENDING ? 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300' : ''}
                        ${node.status === NodeStatus.ERROR ? 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-400' : ''}
                      `}>
                        {node.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-right flex justify-end gap-1" onMouseEnter={() => setTooltipNode(null)}>
                      <button
                        onClick={(e) => { e.stopPropagation(); handleEdit(node); }}
                        className="text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors p-2"
                        title="Edit Configuration"
                      >
                        <Pencil className="w-4 h-4" />
                      </button>
                      <button
                        onClick={(e) => { e.stopPropagation(); rebuildNode(node); }}
                        className="text-gray-400 hover:text-green-600 dark:hover:text-green-400 transition-colors p-2"
                        title="Rebuild System"
                      >
                        <RefreshCcw className="w-4 h-4" />
                      </button>
                      <button
                        onClick={(e) => { e.stopPropagation(); handleDownloadPassphrase(node); }}
                        className="text-gray-400 hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors p-2"
                        title="Download Encryption Passphrase"
                      >
                        <Download className="w-4 h-4" />
                      </button>
                      <button
                        onClick={(e) => { e.stopPropagation(); handleDelete(node.id); }}
                        className="text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors p-2"
                        title="Delete Node"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </td>
                  </tr>
                ))}
                {filteredNodes.length === 0 && (
                  <tr>
                    <td colSpan={5} className="px-6 py-12 text-center text-gray-500 dark:text-gray-400">
                      <div className="flex flex-col items-center gap-3">
                        <HardDrive className="w-10 h-10 text-gray-300 dark:text-gray-600" />
                        <p>No nodes found matching current filters.</p>
                      </div>
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </>
  );
};
