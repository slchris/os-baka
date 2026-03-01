import React from 'react';
import { Cpu, Hash, Network, Laptop, Shield, Activity } from 'lucide-react';
import { NodeConfig, NodeStatus } from '../types';

interface NodeTooltipProps {
    node: NodeConfig;
    position: { x: number; y: number };
}

export const NodeTooltip: React.FC<NodeTooltipProps> = ({ node, position }) => (
    <div
        className="fixed z-50 bg-gray-900/95 backdrop-blur-md text-white p-4 rounded-xl shadow-2xl pointer-events-none border border-gray-700 min-w-[280px] animate-in fade-in duration-200"
        style={{
            top: Math.min(position.y + 16, window.innerHeight - 200),
            left: Math.min(position.x + 16, window.innerWidth - 300)
        }}
    >
        <div className="flex items-center justify-between border-b border-gray-700 pb-2 mb-3">
            <span className="font-bold text-base flex items-center gap-2">
                <Cpu className="w-4 h-4 text-blue-400" />
                {node.hostname}
            </span>
            <span className={`px-2 py-0.5 rounded-full text-[10px] uppercase font-bold tracking-wider ${node.status === NodeStatus.ACTIVE ? 'bg-green-500/20 text-green-400' : 'bg-gray-700 text-gray-300'}`}>
                {node.status}
            </span>
        </div>

        <div className="space-y-2 text-sm text-gray-300 font-mono">
            <div className="flex items-center gap-3">
                <Hash className="w-4 h-4 text-gray-500" />
                <span className="text-gray-500">ID:</span>
                <span className="text-white">{node.id}</span>
            </div>
            <div className="flex items-center gap-3">
                <Network className="w-4 h-4 text-gray-500" />
                <span className="text-gray-500">IP:</span>
                <span className="text-white">{node.ipAddress}</span>
            </div>
            <div className="flex items-center gap-3">
                <div className="w-4" />
                <span className="text-gray-500">MAC:</span>
                <span className="text-white">{node.macAddress}</span>
            </div>
            {node.ipmi && (
                <div className="flex items-center gap-3 text-yellow-500">
                    <Laptop className="w-4 h-4" />
                    <span className="text-gray-500">IPMI:</span>
                    <span className="text-white">{node.ipmi.ip}</span>
                </div>
            )}

            <div className="pt-2 mt-2 border-t border-gray-700">
                <div className="flex items-center gap-2 mb-1">
                    <Shield className="w-4 h-4 text-gray-500" />
                    <span className="text-gray-400 font-sans text-xs font-semibold uppercase">Security Profile</span>
                </div>
                <div className="grid grid-cols-2 gap-x-4 gap-y-1 pl-6">
                    <span className="text-gray-500">Encryption:</span>
                    <span className={node.encryption.enabled ? "text-green-400" : "text-red-400"}>
                        {node.encryption.enabled ? 'Enabled' : 'Disabled'}
                    </span>

                    {node.encryption.enabled && (
                        <>
                            <span className="text-gray-500">Method:</span>
                            <span>{node.encryption.luksVersion.toUpperCase()}</span>

                            <span className="text-gray-500">TPM2:</span>
                            <span className={node.encryption.tpmEnabled ? "text-green-400" : "text-gray-500"}>
                                {node.encryption.tpmEnabled ? 'Active' : 'Inactive'}
                            </span>

                            <span className="text-gray-500">USB Key:</span>
                            <span className={node.encryption.usbKeyRequired ? "text-blue-400" : "text-gray-500"}>
                                {node.encryption.usbKeyRequired ? 'Required' : 'Optional'}
                            </span>
                        </>
                    )}
                </div>
            </div>

            <div className="pt-2 mt-2 border-t border-gray-700 flex items-center gap-2 text-xs">
                <Activity className="w-3 h-3 text-gray-500" />
                <span className="text-gray-500">Last Seen:</span>
                <span className="text-gray-300">{node.lastSeen || 'Unknown'}</span>
            </div>
        </div>
    </div>
);
