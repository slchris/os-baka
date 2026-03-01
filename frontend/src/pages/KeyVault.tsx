
import React, { useState, useEffect } from 'react';
import { Key, Usb, RefreshCw, DownloadCloud, AlertOctagon, KeyRound, Settings, X, ShieldCheck, Check, Layers, Trash2, Plus, Save, Copy } from 'lucide-react';
import { NodeConfig, KeySlot, KeySlotType, NodeStatus } from '../types';
import { BackendService } from '../services/backendService';
import { NodesApi } from '../services/apiClient';

export const KeyVault: React.FC = () => {
  const [nodes, setNodes] = useState<NodeConfig[]>([]);

  useEffect(() => {
    loadNodes();
  }, []);

  const loadNodes = async () => {
    try {
      const res = await NodesApi.list();
      const mapped: NodeConfig[] = res.items.map(n => ({
        id: n.id.toString(),
        hostname: n.hostname,
        ipAddress: n.ip_address,
        macAddress: n.mac_address,
        status: (n.status?.toUpperCase() as NodeStatus) || NodeStatus.PENDING,
        provisioningMethod: 'PXE_MAC',
        encryption: {
          enabled: n.encryption_enabled,
          luksVersion: 'luks2',
          tpmEnabled: n.tpm_enabled ?? true,
          usbKeyRequired: n.usb_key_required ?? false,
          pcrBinding: n.pcr_binding ? n.pcr_binding.split(',').filter(Boolean).map(x => parseInt(x, 10)).filter(x => !Number.isNaN(x)) : [7],
          keySlots: []
        },
        lastSeen: n.last_seen || 'Never',
        osType: n.os_type || 'debian',
        osVersion: n.os_version || '12',
        mirrorUrl: n.mirror_url || '',
        timezone: n.timezone || 'UTC'
      }));
      setNodes(mapped);
    } catch (e) {
      console.error('Failed to load nodes', e);
    }
  };

  const encryptedNodes = nodes.filter(n => n.encryption.enabled);

  // TPM Modal State
  const [editingNode, setEditingNode] = useState<NodeConfig | null>(null);
  const [isTpmModalOpen, setIsTpmModalOpen] = useState(false);
  const [tempTpmEnabled, setTempTpmEnabled] = useState(false);
  const [tempPcrBinding, setTempPcrBinding] = useState<number[]>([]);

  // Key Slots Modal State
  const [keysModalNode, setKeysModalNode] = useState<NodeConfig | null>(null);
  const [isKeysModalOpen, setIsKeysModalOpen] = useState(false);
  const [addingToSlot, setAddingToSlot] = useState<number | null>(null);
  const [newKeyType, setNewKeyType] = useState<KeySlotType>('passphrase');

  // Passphrase Modal State
  const [newPassphrase, setNewPassphrase] = useState<string | null>(null);
  const [passphraseNode, setPassphraseNode] = useState<NodeConfig | null>(null);
  const [copied, setCopied] = useState(false);

  const PCR_OPTIONS = [
    { id: 0, label: 'PCR 0: BIOS Core Root of Trust', description: 'BIOS executable code' },
    { id: 2, label: 'PCR 2: Option ROM Code', description: 'Network/Storage card firmware' },
    { id: 4, label: 'PCR 4: Boot Loader', description: 'MBR and Bootloader code' },
    { id: 7, label: 'PCR 7: Secure Boot State', description: 'Secure Boot variables & keys' },
  ];

  // TPM Handlers
  const openTpmModal = (node: NodeConfig) => {
    setEditingNode(node);
    setTempTpmEnabled(node.encryption.tpmEnabled);
    setTempPcrBinding(node.encryption.pcrBinding || []);
    setIsTpmModalOpen(true);
  };

  const closeTpmModal = () => {
    setIsTpmModalOpen(false);
    setEditingNode(null);
  };

  const saveTpmConfig = () => {
    if (editingNode) {
      const updatedNode: NodeConfig = {
        ...editingNode,
        encryption: {
          ...editingNode.encryption,
          tpmEnabled: tempTpmEnabled,
          pcrBinding: tempTpmEnabled ? tempPcrBinding : []
        }
      };
      BackendService.updateNode(updatedNode);
      loadNodes();
      closeTpmModal();
    }
  };

  const togglePcr = (pcrId: number) => {
    if (tempPcrBinding.includes(pcrId)) {
      setTempPcrBinding(tempPcrBinding.filter(id => id !== pcrId));
    } else {
      setTempPcrBinding([...tempPcrBinding, pcrId]);
    }
  };

  // Key Slot Handlers
  const openKeysModal = (node: NodeConfig) => {
    setKeysModalNode(node);
    setAddingToSlot(null);
    setIsKeysModalOpen(true);
  };

  const closeKeysModal = () => {
    setIsKeysModalOpen(false);
    setKeysModalNode(null);
    setAddingToSlot(null);
  };

  const handleAddKey = (slotIndex: number) => {
    if (!keysModalNode || !keysModalNode.encryption.keySlots) return;

    const updatedSlots = [...keysModalNode.encryption.keySlots];
    updatedSlots[slotIndex] = {
      index: slotIndex,
      active: true,
      type: newKeyType,
      label: newKeyType === 'passphrase' ? 'User Passphrase' : 'Keyfile/Token'
    };

    const updatedNode = {
      ...keysModalNode,
      encryption: {
        ...keysModalNode.encryption,
        keySlots: updatedSlots
      }
    };

    BackendService.updateNode(updatedNode);
    setKeysModalNode(updatedNode); // Update local modal state
    setAddingToSlot(null);
    loadNodes(); // Refresh main app state
  };

  const handleRevokeKey = (slotIndex: number) => {
    if (!keysModalNode || !keysModalNode.encryption.keySlots) return;
    if (!confirm("Are you sure you want to revoke this key slot? This action is irreversible.")) return;

    const updatedSlots = [...keysModalNode.encryption.keySlots];
    updatedSlots[slotIndex] = {
      index: slotIndex,
      active: false
    };

    const updatedNode = {
      ...keysModalNode,
      encryption: {
        ...keysModalNode.encryption,
        keySlots: updatedSlots
      }
    };

    BackendService.updateNode(updatedNode);
    setKeysModalNode(updatedNode);
    loadNodes();
  };

  // File Download Utilities
  const downloadFile = (blob: Blob, filename: string) => {
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  const handleDownloadHeader = (node: NodeConfig) => {
    const blob = BackendService.generateLuksHeader(node.hostname);
    downloadFile(blob, `${node.hostname}.luks.img`);
  };

  const handleDownloadUsbKey = (node: NodeConfig) => {
    const blob = BackendService.generateUsbKey(node.hostname);
    downloadFile(blob, `${node.hostname}.usb.key`);
  };

  const handleGeneratePassphrase = (node: NodeConfig) => {
    if (window.confirm(`Generate new LUKS recovery passphrase for ${node.hostname}? This will update Key Slot #1 and invalidate the previous recovery key.`)) {
      try {
        const pass = BackendService.regenerateRecoveryKey(node.id);
        setPassphraseNode(node);
        setNewPassphrase(pass);
        // Note: We do NOT loadNodes() here to prevent re-renders from closing/resetting the modal state unexpectedly.
        // We will refresh when the user closes the modal.
      } catch (e) {
        console.error("Failed to generate passphrase", e);
        alert("Failed to generate passphrase. Please try again.");
      }
    }
  };

  const closePassphraseModal = () => {
    setNewPassphrase(null);
    setPassphraseNode(null);
    setCopied(false);
    loadNodes(); // Refresh parent state now that we are done
  };

  const handleCopyPassphrase = () => {
    if (newPassphrase) {
      navigator.clipboard.writeText(newPassphrase);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleDownloadPassphrase = () => {
    if (passphraseNode && newPassphrase) {
      const blob = BackendService.createPassphraseBlob(passphraseNode.hostname, newPassphrase);
      downloadFile(blob, `${passphraseNode.hostname}.recovery-key.txt`);
    }
  };

  return (
    <div className="space-y-6 relative">
      <div>
        <h2 className="text-2xl font-bold text-gray-900 dark:text-white">Key Vault</h2>
        <p className="text-gray-500 dark:text-gray-400">Manage LUKS passphrases, TPM2 bindings, and USB unlock keys securely.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {encryptedNodes.map(node => (
          <div key={node.id} className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-2xl p-6 shadow-sm hover:shadow-md transition-all relative overflow-hidden group">
            <div className="flex justify-between items-start mb-4">
              <div className="p-3 bg-gray-50 dark:bg-gray-700 rounded-xl">
                <Key className="w-6 h-6 text-gray-700 dark:text-gray-300" />
              </div>
              <div className="flex gap-2">
                {node.encryption.tpmEnabled && (
                  <div className="p-2 bg-gray-100 dark:bg-gray-700 rounded-lg" title="TPM2 Enabled">
                    <ShieldCheck className="w-4 h-4 text-green-600 dark:text-green-400" />
                  </div>
                )}
                {node.encryption.usbKeyRequired && (
                  <div className="p-2 bg-blue-50 dark:bg-blue-900/20 rounded-lg" title="USB Key Required">
                    <Usb className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                  </div>
                )}
              </div>
            </div>

            <h3 className="font-bold text-gray-900 dark:text-white">{node.hostname}</h3>
            <p className="text-xs text-gray-500 dark:text-gray-400 font-mono mb-1 truncate">{node.id}</p>

            {/* Encryption Method Info */}
            <div className="mb-4 p-2 bg-purple-50 dark:bg-purple-900/20 rounded-lg border border-purple-100 dark:border-purple-900/20">
              <p className="text-xs font-semibold text-purple-700 dark:text-purple-400 mb-1">Encryption Method</p>
              <p className="text-xs text-gray-600 dark:text-gray-400">
                <span className="font-mono">{node.encryption.luksVersion?.toUpperCase() || 'LUKS2'}</span>
                {node.encryption.tpmEnabled && ' + TPM2'}
                {node.encryption.usbKeyRequired && ' + USB Key'}
              </p>
              {node.encryption.tpmEnabled && node.encryption.pcrBinding && node.encryption.pcrBinding.length > 0 && (
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  PCR Registers: {node.encryption.pcrBinding.join(', ')}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <button
                onClick={() => handleDownloadHeader(node)}
                className="w-full flex items-center justify-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-50 dark:bg-gray-700 hover:bg-gray-100 dark:hover:bg-gray-600 py-2 rounded-lg border border-gray-200 dark:border-gray-600 transition-colors"
              >
                <DownloadCloud className="w-4 h-4" />
                Download LUKS Header
              </button>

              <button
                onClick={() => openTpmModal(node)}
                className="w-full flex items-center justify-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-50 dark:bg-gray-700 hover:bg-gray-100 dark:hover:bg-gray-600 py-2 rounded-lg border border-gray-200 dark:border-gray-600 transition-colors"
              >
                <Settings className="w-4 h-4" />
                TPM Configuration
              </button>

              <button
                onClick={() => openKeysModal(node)}
                className="w-full flex items-center justify-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-50 dark:bg-gray-700 hover:bg-gray-100 dark:hover:bg-gray-600 py-2 rounded-lg border border-gray-200 dark:border-gray-600 transition-colors"
              >
                <Layers className="w-4 h-4" />
                Manage Key Slots
              </button>

              {node.encryption.usbKeyRequired && (
                <button
                  onClick={() => handleDownloadUsbKey(node)}
                  className="w-full flex items-center justify-center gap-2 text-sm font-medium text-blue-700 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20 hover:bg-blue-100 dark:hover:bg-blue-900/30 py-2 rounded-lg border border-blue-100 dark:border-blue-900/20 transition-colors"
                >
                  <RefreshCw className="w-4 h-4" />
                  Regenerate USB Key
                </button>
              )}
              <button
                onClick={() => handleGeneratePassphrase(node)}
                className="w-full flex items-center justify-center gap-2 text-sm font-medium text-purple-700 dark:text-purple-400 bg-purple-50 dark:bg-purple-900/20 hover:bg-purple-100 dark:hover:bg-purple-900/30 py-2 rounded-lg border border-purple-100 dark:border-purple-900/20 transition-colors"
              >
                <KeyRound className="w-4 h-4" />
                Generate Passphrase
              </button>
            </div>

            {/* Decorative background element */}
            <div className="absolute -right-6 -top-6 w-24 h-24 bg-gray-50 dark:bg-gray-700 rounded-full blur-2xl opacity-50 group-hover:opacity-100 transition-opacity" />
          </div>
        ))}
      </div>
      {encryptedNodes.length === 0 && (
        <div className="text-center py-20 bg-white dark:bg-gray-800 rounded-2xl border border-dashed border-gray-300 dark:border-gray-700">
          <p className="text-gray-400 dark:text-gray-500">No encrypted nodes configured.</p>
        </div>
      )}

      {/* TPM Configuration Modal */}
      {isTpmModalOpen && editingNode && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm animate-fade-in">
          <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl max-w-md w-full overflow-hidden animate-in zoom-in-95 duration-200">
            <div className="p-5 border-b border-gray-100 dark:border-gray-700 flex justify-between items-center bg-gray-50 dark:bg-gray-900">
              <div>
                <h3 className="font-bold text-gray-900 dark:text-white text-lg">TPM2 Configuration</h3>
                <p className="text-xs text-gray-500 dark:text-gray-400">{editingNode.hostname}</p>
              </div>
              <button
                onClick={closeTpmModal}
                className="p-2 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-full transition-colors text-gray-500"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-6 space-y-6">
              {/* Toggle Enrollment */}
              <div className="flex items-center justify-between p-4 rounded-xl border border-gray-200 dark:border-gray-700 hover:border-blue-200 dark:hover:border-blue-800 transition-colors cursor-pointer" onClick={() => setTempTpmEnabled(!tempTpmEnabled)}>
                <div className="flex items-center gap-3">
                  <div className={`p-2 rounded-lg ${tempTpmEnabled ? 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400' : 'bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400'}`}>
                    <ShieldCheck className="w-6 h-6" />
                  </div>
                  <div>
                    <p className="font-medium text-gray-900 dark:text-white">TPM Enrollment</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">{tempTpmEnabled ? 'Keys sealed to TPM' : 'TPM Unbound'}</p>
                  </div>
                </div>
                <div className={`w-12 h-6 rounded-full p-1 transition-colors duration-200 ${tempTpmEnabled ? 'bg-green-500' : 'bg-gray-200 dark:bg-gray-700'}`}>
                  <div className={`w-4 h-4 bg-white rounded-full shadow-sm transition-transform duration-200 ${tempTpmEnabled ? 'translate-x-6' : 'translate-x-0'}`} />
                </div>
              </div>

              {/* PCR Selection */}
              {tempTpmEnabled && (
                <div className="space-y-3 animate-in slide-in-from-top-4 fade-in duration-300">
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block">
                    Platform Configuration Registers (PCRs)
                  </label>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                    Select system states that must match to unseal the key. Warning: Changing hardware/firmware may lock the disk if selected.
                  </p>

                  <div className="grid gap-2">
                    {PCR_OPTIONS.map(pcr => (
                      <div
                        key={pcr.id}
                        onClick={() => togglePcr(pcr.id)}
                        className={`flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-all
                          ${tempPcrBinding.includes(pcr.id)
                            ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 text-blue-900 dark:text-blue-100'
                            : 'border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300'}
                        `}
                      >
                        <div className={`w-5 h-5 rounded border flex items-center justify-center transition-colors
                          ${tempPcrBinding.includes(pcr.id) ? 'bg-blue-500 border-blue-500' : 'bg-white dark:bg-gray-600 border-gray-300 dark:border-gray-500'}
                        `}>
                          {tempPcrBinding.includes(pcr.id) && <Check className="w-3 h-3 text-white" />}
                        </div>
                        <div>
                          <p className="text-sm font-semibold">{pcr.label}</p>
                          <p className="text-[10px] opacity-80">{pcr.description}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            <div className="p-5 border-t border-gray-100 dark:border-gray-700 flex justify-end gap-3 bg-gray-50 dark:bg-gray-900">
              <button
                onClick={closeTpmModal}
                className="px-4 py-2 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={saveTpmConfig}
                className="px-4 py-2 bg-black dark:bg-white text-white dark:text-black text-sm font-medium rounded-lg hover:bg-gray-800 dark:hover:bg-gray-200 transition-all shadow-lg shadow-gray-200 dark:shadow-none"
              >
                Apply Changes
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Key Slots Management Modal */}
      {isKeysModalOpen && keysModalNode && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm animate-fade-in">
          <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl max-w-2xl w-full overflow-hidden animate-in zoom-in-95 duration-200 flex flex-col max-h-[90vh]">
            <div className="p-5 border-b border-gray-100 dark:border-gray-700 flex justify-between items-center bg-gray-50 dark:bg-gray-900 shrink-0">
              <div>
                <h3 className="font-bold text-gray-900 dark:text-white text-lg">LUKS Key Slot Management</h3>
                <p className="text-xs text-gray-500 dark:text-gray-400">Active Keys for {keysModalNode.hostname}</p>
              </div>
              <button
                onClick={closeKeysModal}
                className="p-2 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-full transition-colors text-gray-500"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-6 overflow-y-auto">
              <div className="space-y-4">
                <div className="flex items-center gap-2 p-3 bg-yellow-50 dark:bg-yellow-900/20 text-yellow-800 dark:text-yellow-200 rounded-lg border border-yellow-100 dark:border-yellow-900/30 text-sm">
                  <AlertOctagon className="w-4 h-4 shrink-0" />
                  <p>Warning: Ensure you always have at least one known passphrase or recovery key active. Removing all keys will render data permanently inaccessible.</p>
                </div>

                <table className="w-full text-left border-collapse">
                  <thead>
                    <tr className="border-b border-gray-200 dark:border-gray-700 text-xs uppercase text-gray-500 dark:text-gray-400">
                      <th className="py-2 px-4 w-16">Slot</th>
                      <th className="py-2 px-4">Status</th>
                      <th className="py-2 px-4">Type</th>
                      <th className="py-2 px-4 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                    {keysModalNode.encryption.keySlots?.map((slot) => (
                      <tr key={slot.index} className="group hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors">
                        <td className="py-3 px-4 font-mono text-gray-500 dark:text-gray-400">#{slot.index}</td>
                        <td className="py-3 px-4">
                          {slot.active ? (
                            <span className="inline-flex items-center px-2 py-1 rounded-md bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400 text-xs font-medium border border-green-100 dark:border-green-900/30">
                              Active
                            </span>
                          ) : (
                            <span className="inline-flex items-center px-2 py-1 rounded-md bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 text-xs font-medium border border-gray-200 dark:border-gray-600">
                              Empty
                            </span>
                          )}
                        </td>
                        <td className="py-3 px-4">
                          {slot.active ? (
                            <div>
                              <span className="text-sm font-medium text-gray-900 dark:text-white block capitalize">{slot.type}</span>
                              {slot.label && <span className="text-xs text-gray-500 dark:text-gray-400 block">{slot.label}</span>}
                            </div>
                          ) : (
                            <span className="text-gray-400 text-sm">-</span>
                          )}
                        </td>
                        <td className="py-3 px-4 text-right">
                          {slot.active ? (
                            <button
                              onClick={() => handleRevokeKey(slot.index)}
                              className="text-gray-400 hover:text-red-600 dark:hover:text-red-400 p-2 transition-colors"
                              title="Revoke/Wipe Slot"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          ) : (
                            addingToSlot === slot.index ? (
                              <div className="flex items-center justify-end gap-2">
                                <select
                                  className="text-xs border rounded p-1 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                                  value={newKeyType}
                                  onChange={(e) => setNewKeyType(e.target.value as KeySlotType)}
                                >
                                  <option value="passphrase">Passphrase</option>
                                  <option value="keyfile">Keyfile</option>
                                  <option value="recovery">Recovery</option>
                                </select>
                                <button
                                  onClick={() => handleAddKey(slot.index)}
                                  className="text-green-600 dark:text-green-400 hover:text-green-700 p-1"
                                >
                                  <Save className="w-4 h-4" />
                                </button>
                                <button
                                  onClick={() => setAddingToSlot(null)}
                                  className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 p-1"
                                >
                                  <X className="w-4 h-4" />
                                </button>
                              </div>
                            ) : (
                              <button
                                onClick={() => setAddingToSlot(slot.index)}
                                className="text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 p-2 transition-colors"
                                title="Add Key"
                              >
                                <Plus className="w-4 h-4" />
                              </button>
                            )
                          )}
                        </td>
                      </tr>
                    ))}
                    {!keysModalNode.encryption.keySlots && (
                      <tr><td colSpan={4} className="text-center py-4 text-gray-500 dark:text-gray-400">No key slots information available.</td></tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Generated Passphrase Modal */}
      {newPassphrase && passphraseNode && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm animate-fade-in">
          <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl max-w-md w-full overflow-hidden animate-in zoom-in-95 duration-200">
            <div className="p-5 border-b border-gray-100 dark:border-gray-700 flex justify-between items-center bg-gray-50 dark:bg-gray-900">
              <div className="flex items-center gap-2">
                <div className="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg">
                  <KeyRound className="w-5 h-5 text-green-600 dark:text-green-400" />
                </div>
                <div>
                  <h3 className="font-bold text-gray-900 dark:text-white text-lg">Recovery Key Generated</h3>
                  <p className="text-xs text-gray-500 dark:text-gray-400">{passphraseNode.hostname}</p>
                </div>
              </div>
              <button onClick={closePassphraseModal} className="p-2 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-full text-gray-500">
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-6 space-y-4">
              <div className="flex items-center gap-2 p-3 bg-yellow-50 dark:bg-yellow-900/20 text-yellow-800 dark:text-yellow-200 rounded-lg border border-yellow-100 dark:border-yellow-900/30 text-sm">
                <AlertOctagon className="w-4 h-4 shrink-0" />
                <p>Store this passphrase securely. It will not be displayed again once this window is closed.</p>
              </div>

              <div className="relative">
                <pre className="w-full p-4 bg-gray-100 dark:bg-gray-950 border border-gray-200 dark:border-gray-700 rounded-xl font-mono text-sm text-gray-800 dark:text-gray-200 break-all whitespace-pre-wrap">
                  {newPassphrase}
                </pre>
                <button
                  onClick={handleCopyPassphrase}
                  className="absolute top-2 right-2 p-2 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                  title="Copy to Clipboard"
                >
                  {copied ? <Check className="w-4 h-4 text-green-500" /> : <Copy className="w-4 h-4 text-gray-500" />}
                </button>
              </div>

              <div className="pt-2">
                <button
                  onClick={handleDownloadPassphrase}
                  className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-black dark:bg-white text-white dark:text-black font-medium rounded-xl hover:opacity-90 transition-all shadow-lg"
                >
                  <DownloadCloud className="w-4 h-4" /> Download as File
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
