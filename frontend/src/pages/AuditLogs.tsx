import React, { useState, useEffect, useCallback } from 'react';
import { FileText, Search, ChevronLeft, ChevronRight, Filter, User, Server, Network, Shield, Key, Bell } from 'lucide-react';
import { AuditApi, type AuditLogItem } from '../services/apiClient';

const RESOURCE_ICONS: Record<string, React.ElementType> = {
    node: Server,
    user: User,
    auth: Shield,
    dhcp: Network,
    asset: Key,
    notification: Bell,
    system: FileText,
};

const ACTION_COLORS: Record<string, string> = {
    create: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
    update: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
    delete: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
    login: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
    rebuild: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
    password: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
    restart: 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400',
    sync: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400',
    upload: 'bg-teal-100 text-teal-700 dark:bg-teal-900/30 dark:text-teal-400',
};

function getActionColor(action: string): string {
    const parts = action.split('.');
    const verb = parts.length > 1 ? parts[1] : parts[0];
    for (const [key, color] of Object.entries(ACTION_COLORS)) {
        if (verb.includes(key)) return color;
    }
    return 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300';
}

function formatTimestamp(ts: string): string {
    const d = new Date(ts);
    return d.toLocaleString('zh-CN', {
        year: 'numeric', month: '2-digit', day: '2-digit',
        hour: '2-digit', minute: '2-digit', second: '2-digit',
    });
}

const FILTER_OPTIONS = [
    { label: 'All', value: '' },
    { label: 'Node', value: 'node' },
    { label: 'User', value: 'user' },
    { label: 'DHCP', value: 'dhcp' },
    { label: 'Asset', value: 'asset' },
];

export const AuditLogs: React.FC = () => {
    const [logs, setLogs] = useState<AuditLogItem[]>([]);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);
    const [actionFilter, setActionFilter] = useState('');
    const [loading, setLoading] = useState(false);
    const limit = 25;

    const loadLogs = useCallback(async () => {
        setLoading(true);
        try {
            const resp = await AuditApi.list({ page, limit, action: actionFilter || undefined });
            setLogs(resp.items || []);
            setTotal(resp.total);
        } catch (e) {
            console.error('Failed to load audit logs', e);
        } finally {
            setLoading(false);
        }
    }, [page, actionFilter]);

    useEffect(() => {
        loadLogs();
    }, [loadLogs]);

    const totalPages = Math.max(1, Math.ceil(total / limit));

    return (
        <div className="space-y-6 animate-fade-in">
            <div>
                <h2 className="text-2xl font-bold text-gray-900 dark:text-white">Audit Logs</h2>
                <p className="text-gray-500 dark:text-gray-400">Track all system changes and user actions.</p>
            </div>

            {/* Filters */}
            <div className="flex items-center gap-4 flex-wrap">
                <div className="flex items-center gap-2 bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-1">
                    <Filter className="w-4 h-4 text-gray-400 ml-2" />
                    {FILTER_OPTIONS.map((opt) => (
                        <button
                            key={opt.value}
                            onClick={() => { setActionFilter(opt.value); setPage(1); }}
                            className={`px-3 py-1.5 text-xs font-medium rounded-lg transition-all ${actionFilter === opt.value
                                    ? 'bg-black dark:bg-white text-white dark:text-black shadow-sm'
                                    : 'text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-700'
                                }`}
                        >
                            {opt.label}
                        </button>
                    ))}
                </div>

                <div className="text-xs text-gray-400 dark:text-gray-500 ml-auto">
                    {total} total entries
                </div>
            </div>

            {/* Table */}
            <div className="bg-white dark:bg-gray-800 rounded-2xl border border-gray-200 dark:border-gray-700 shadow-sm overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="w-full text-sm text-left">
                        <thead className="text-xs text-gray-500 dark:text-gray-400 uppercase bg-gray-50 dark:bg-gray-900/50 border-b border-gray-100 dark:border-gray-700">
                            <tr>
                                <th className="px-5 py-3 font-medium">Timestamp</th>
                                <th className="px-5 py-3 font-medium">Action</th>
                                <th className="px-5 py-3 font-medium">User</th>
                                <th className="px-5 py-3 font-medium">Resource</th>
                                <th className="px-5 py-3 font-medium">Details</th>
                                <th className="px-5 py-3 font-medium">IP</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                            {loading && (
                                <tr>
                                    <td colSpan={6} className="px-5 py-12 text-center text-gray-400">
                                        <div className="flex items-center justify-center gap-2">
                                            <div className="w-4 h-4 border-2 border-gray-300 border-t-gray-600 rounded-full animate-spin" />
                                            Loading...
                                        </div>
                                    </td>
                                </tr>
                            )}

                            {!loading && logs.length === 0 && (
                                <tr>
                                    <td colSpan={6} className="px-5 py-12 text-center text-gray-400">
                                        <Search className="w-8 h-8 mx-auto mb-2 opacity-50" />
                                        <p>No audit log entries found.</p>
                                    </td>
                                </tr>
                            )}

                            {!loading && logs.map((log) => {
                                const ResourceIcon = RESOURCE_ICONS[log.resource] || FileText;
                                return (
                                    <tr key={log.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors">
                                        <td className="px-5 py-3 text-xs text-gray-500 dark:text-gray-400 font-mono whitespace-nowrap">
                                            {formatTimestamp(log.timestamp)}
                                        </td>
                                        <td className="px-5 py-3">
                                            <span className={`inline-block px-2 py-0.5 rounded-md text-xs font-bold ${getActionColor(log.action)}`}>
                                                {log.action}
                                            </span>
                                        </td>
                                        <td className="px-5 py-3">
                                            <div className="flex items-center gap-2">
                                                <User className="w-3.5 h-3.5 text-gray-400" />
                                                <span className="text-sm text-gray-900 dark:text-gray-200 font-medium">{log.user}</span>
                                            </div>
                                        </td>
                                        <td className="px-5 py-3">
                                            <div className="flex items-center gap-2">
                                                <ResourceIcon className="w-3.5 h-3.5 text-gray-400" />
                                                <span className="text-sm text-gray-600 dark:text-gray-300">
                                                    {log.resource}
                                                    {log.resource_id && <span className="text-gray-400 ml-1">#{log.resource_id}</span>}
                                                </span>
                                            </div>
                                        </td>
                                        <td className="px-5 py-3 text-xs text-gray-500 dark:text-gray-400 max-w-xs truncate" title={log.details}>
                                            {log.details}
                                        </td>
                                        <td className="px-5 py-3 text-xs text-gray-400 font-mono">
                                            {log.ip_address}
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                </div>

                {/* Pagination */}
                {totalPages > 1 && (
                    <div className="flex items-center justify-between p-4 border-t border-gray-100 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50">
                        <span className="text-xs text-gray-500 dark:text-gray-400">
                            Page {page} of {totalPages}
                        </span>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={() => setPage(Math.max(1, page - 1))}
                                disabled={page <= 1}
                                className="p-1.5 rounded-lg border border-gray-200 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                            >
                                <ChevronLeft className="w-4 h-4" />
                            </button>
                            <button
                                onClick={() => setPage(Math.min(totalPages, page + 1))}
                                disabled={page >= totalPages}
                                className="p-1.5 rounded-lg border border-gray-200 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                            >
                                <ChevronRight className="w-4 h-4" />
                            </button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
};
