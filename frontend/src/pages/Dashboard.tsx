
import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Activity, Server, Lock, AlertTriangle, CheckCircle2, Clock } from 'lucide-react';
import { NodeConfig, NodeStatus } from '../types';
import { NodesApi } from '../services/apiClient';

export const Dashboard: React.FC = () => {
  const navigate = useNavigate();
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

  const activeNodes = nodes.filter(n => n.status === NodeStatus.ACTIVE).length;
  const encryptedNodes = nodes.filter(n => n.encryption.enabled).length;
  const warningNodes = nodes.filter(n => n.status === NodeStatus.ERROR || n.status === NodeStatus.OFFLINE).length;

  const StatCard = ({ title, value, subtitle, icon: Icon, color, onClick }: any) => (
    <div 
      onClick={onClick}
      className="bg-white dark:bg-gray-800 rounded-2xl p-6 border border-gray-100 dark:border-gray-700 shadow-sm hover:shadow-md transition-all cursor-pointer hover:scale-[1.02] group"
    >
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1 group-hover:text-gray-700 dark:group-hover:text-gray-200 transition-colors">{title}</p>
          <h3 className="text-3xl font-bold text-gray-900 dark:text-white">{value}</h3>
          <p className="text-xs text-gray-400 dark:text-gray-500 mt-2">{subtitle}</p>
        </div>
        <div className={`p-3 rounded-xl ${color}`}>
          <Icon className="w-6 h-6 text-white" />
        </div>
      </div>
    </div>
  );

  return (
    <div className="space-y-6 animate-fade-in">
      <div>
        <h2 className="text-2xl font-bold text-gray-900 dark:text-white">Overview</h2>
        <p className="text-gray-500 dark:text-gray-400">Real-time infrastructure status.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard
          title="Total Nodes"
          value={nodes.length}
          subtitle="Target Hosts"
          icon={Server}
          color="bg-blue-500"
          onClick={() => navigate('/nodes')}
        />
        <StatCard
          title="Active Instances"
          value={activeNodes}
          subtitle="Online & Responsive"
          icon={Activity}
          color="bg-green-500"
          onClick={() => navigate('/nodes')}
        />
        <StatCard
          title="Secured (LUKS)"
          value={encryptedNodes}
          subtitle={`${nodes.length > 0 ? Math.round((encryptedNodes / nodes.length) * 100) : 0}% coverage`}
          icon={Lock}
          color="bg-purple-500"
          onClick={() => navigate('/keyvault')}
        />
        <StatCard
          title="Attention Needed"
          value={warningNodes}
          subtitle="Errors / Offline"
          icon={AlertTriangle}
          color="bg-orange-500"
          onClick={() => navigate('/nodes')}
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white dark:bg-gray-800 rounded-2xl border border-gray-100 dark:border-gray-700 shadow-sm p-6">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Recent Activity</h3>
          <div className="space-y-4">
            {nodes.slice(0, 5).map((node) => (
              <div key={node.id} className="flex items-center justify-between py-2 border-b border-gray-50 dark:border-gray-700 last:border-0">
                <div className="flex items-center gap-3">
                  <div className={`w-2 h-2 rounded-full ${node.status === NodeStatus.ACTIVE ? 'bg-green-500' : 'bg-gray-300 dark:bg-gray-600'}`} />
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-200">{node.hostname}</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">{node.ipAddress}</p>
                  </div>
                </div>
                <div className="text-right">
                  <span className="text-xs font-medium px-2 py-1 rounded-md bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 block mb-1">
                    {node.status}
                  </span>
                  <span className="text-[10px] text-gray-400">{node.lastSeen || 'Just now'}</span>
                </div>
              </div>
            ))}
             {nodes.length === 0 && <p className="text-gray-400 text-sm">No recent activity detected.</p>}
          </div>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-2xl border border-gray-100 dark:border-gray-700 shadow-sm p-6 flex flex-col justify-between">
           <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">System Health</h3>
            <div className="space-y-4">
                <div className="flex items-center gap-3">
                    <CheckCircle2 className="w-5 h-5 text-green-500" />
                    <span className="text-sm text-gray-600 dark:text-gray-300">Backend Service: <strong className="dark:text-white">Online</strong></span>
                </div>
                <div className="flex items-center gap-3">
                    <CheckCircle2 className="w-5 h-5 text-green-500" />
                    <span className="text-sm text-gray-600 dark:text-gray-300">Encryption Module: <strong className="dark:text-white">Active (TPM2 Ready)</strong></span>
                </div>
                 <div className="flex items-center gap-3">
                    <CheckCircle2 className="w-5 h-5 text-green-500" />
                    <span className="text-sm text-gray-600 dark:text-gray-300">DNSMasq Generator: <strong className="dark:text-white">Ready</strong></span>
                </div>
            </div>
           </div>
           
           <div className="mt-6 bg-gray-50 dark:bg-gray-700/50 rounded-xl p-4">
                <div className="flex items-center gap-2 mb-2">
                    <Clock className="w-4 h-4 text-gray-400" />
                    <span className="text-xs font-bold text-gray-500 dark:text-gray-400 uppercase">Next Scheduled Maintenance</span>
                </div>
                <p className="text-sm text-gray-900 dark:text-white">Sunday, 02:00 AM UTC</p>
           </div>
        </div>
      </div>
    </div>
  );
};