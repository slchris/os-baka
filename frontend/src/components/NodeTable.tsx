import React from 'react';
import { RefreshCcw, HardDrive, Download, Trash2, Pencil, Laptop, Network } from 'lucide-react';
import { NodeConfig, NodeStatus } from '../types';

interface NodeTableProps {
    nodes: NodeConfig[];
    onEdit: (node: NodeConfig) => void;
    onRebuild: (node: NodeConfig) => void;
    onDelete: (id: string) => void;
    onDownloadPassphrase: (node: NodeConfig) => void;
    onNodeHover: (e: React.MouseEvent, node: NodeConfig) => void;
    onNodeLeave: () => void;
}

export const NodeTable: React.FC<NodeTableProps> = ({
    nodes, onEdit, onRebuild, onDelete, onDownloadPassphrase, onNodeHover, onNodeLeave
}) => (
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
                    {nodes.map((node) => (
                        <tr
                            key={node.id}
                            className="hover:bg-gray-50/50 dark:hover:bg-gray-700/30 transition-colors cursor-default"
                            onMouseLeave={onNodeLeave}
                        >
                            <td className="px-6 py-4" onMouseEnter={(e) => onNodeHover(e, node)}>
                                <div className="font-medium text-gray-900 dark:text-white">{node.hostname}</div>
                                <div className="text-xs text-gray-500 dark:text-gray-400">OS: {node.osType || 'unknown'} {node.osVersion ? `(${node.osVersion})` : ''}</div>
                            </td>
                            <td className="px-6 py-4" onMouseEnter={(e) => onNodeHover(e, node)}>
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
                            <td className="px-6 py-4" onMouseEnter={(e) => onNodeHover(e, node)}>
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
                            <td className="px-6 py-4" onMouseEnter={(e) => onNodeHover(e, node)}>
                                <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium
                  ${node.status === NodeStatus.ACTIVE ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-400' : ''}
                  ${node.status === NodeStatus.REBUILDING ? 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-400 animate-pulse' : ''}
                  ${node.status === NodeStatus.PENDING ? 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300' : ''}
                  ${node.status === NodeStatus.ERROR ? 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-400' : ''}
                `}>
                                    {node.status}
                                </span>
                            </td>
                            <td className="px-6 py-4 text-right flex justify-end gap-1" onMouseEnter={onNodeLeave}>
                                <button onClick={(e) => { e.stopPropagation(); onEdit(node); }} className="text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors p-2" title="Edit Configuration">
                                    <Pencil className="w-4 h-4" />
                                </button>
                                <button onClick={(e) => { e.stopPropagation(); onRebuild(node); }} className="text-gray-400 hover:text-green-600 dark:hover:text-green-400 transition-colors p-2" title="Rebuild System">
                                    <RefreshCcw className="w-4 h-4" />
                                </button>
                                <button onClick={(e) => { e.stopPropagation(); onDownloadPassphrase(node); }} className="text-gray-400 hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors p-2" title="Download Encryption Passphrase">
                                    <Download className="w-4 h-4" />
                                </button>
                                <button onClick={(e) => { e.stopPropagation(); onDelete(node.id); }} className="text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors p-2" title="Delete Node">
                                    <Trash2 className="w-4 h-4" />
                                </button>
                            </td>
                        </tr>
                    ))}
                    {nodes.length === 0 && (
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
);
