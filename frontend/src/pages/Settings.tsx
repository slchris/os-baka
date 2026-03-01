
import React, { useEffect, useState } from 'react';
import { ShieldAlert, UserCog, Lock, Check, X, Shield } from 'lucide-react';
import { Role, User } from '../types';
import { BackendService } from '../services/backendService';

const PERMISSIONS_MATRIX = [
  { feature: 'Node Provisioning (Add/Delete)', admin: true, operator: false, auditor: false },
  { feature: 'Node Lifecycle (Start/Rebuild)', admin: true, operator: true, auditor: false },
  { feature: 'Key Vault (Generate/Revoke)', admin: true, operator: false, auditor: false },
  { feature: 'Key Vault (View Status)', admin: true, operator: true, auditor: true },
  { feature: 'WebShell (Root Access)', admin: true, operator: true, auditor: false },
  { feature: 'Audit Logs (View)', admin: true, operator: true, auditor: true },
  { feature: 'User Management (RBAC)', admin: true, operator: false, auditor: false },
];

export const Settings: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);

  useEffect(() => {
    setUsers(BackendService.getUsers());
  }, []);

  const handleRoleChange = (userId: string, newRole: string) => {
    // In a real app, we'd validate this casting
    const updated = BackendService.updateUserRole(userId, newRole as Role);
    setUsers(updated);
  };

  const getRoleColor = (role: Role) => {
    switch (role) {
      case Role.ADMIN: return 'text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 border-red-100 dark:border-red-900/30';
      case Role.OPERATOR: return 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20 border-blue-100 dark:border-blue-900/30';
      case Role.AUDITOR: return 'text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-700 border-gray-200 dark:border-gray-600';
      default: return 'text-gray-500';
    }
  };

  return (
    <div className="space-y-8 max-w-5xl animate-fade-in pb-10">
      <div>
        <h2 className="text-2xl font-bold text-gray-900 dark:text-white">Platform Settings</h2>
        <p className="text-gray-500 dark:text-gray-400">Manage user access controls and system permissions.</p>
      </div>

      {/* RBAC Users Section */}
      <section>
        <h2 className="text-lg font-bold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
          <UserCog className="w-5 h-5" /> User Management
        </h2>
        <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-2xl shadow-sm overflow-hidden">
          <div className="p-6 border-b border-gray-100 dark:border-gray-700">
            <h3 className="font-medium text-gray-900 dark:text-white">Registered Users</h3>
            <p className="text-sm text-gray-500 dark:text-gray-400">Assign roles to control access levels across the platform.</p>
          </div>
          <div className="divide-y divide-gray-100 dark:divide-gray-700">
            {users.map(user => (
              <div key={user.id} className="flex items-center justify-between p-6 hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors">
                <div className="flex items-center gap-4">
                  <img src={user.avatarUrl} alt={user.name} className="w-12 h-12 rounded-full bg-gray-100 dark:bg-gray-700" />
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="font-bold text-gray-900 dark:text-white">{user.name}</p>
                      {user.email === 'admin@os-baka.local' && (
                        <span className="text-[10px] uppercase font-bold bg-gray-100 dark:bg-gray-600 text-gray-500 dark:text-gray-300 px-1.5 py-0.5 rounded">You</span>
                      )}
                    </div>
                    <p className="text-sm text-gray-500 dark:text-gray-400">{user.email}</p>
                  </div>
                </div>

                <div className="flex items-center gap-4">
                  <div className={`px-3 py-1 rounded-lg text-xs font-bold border ${getRoleColor(user.role)}`}>
                    {user.role}
                  </div>
                  <select
                    value={user.role}
                    onChange={(e) => handleRoleChange(user.id, e.target.value)}
                    disabled={user.email === 'admin@os-baka.local'} // Prevent self-demotion for demo
                    className="text-sm text-gray-900 dark:text-white border-gray-200 dark:border-gray-600 rounded-lg shadow-sm border p-2 bg-white dark:bg-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500/20 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed block"
                  >
                    {Object.values(Role).map((r) => (
                      <option key={r} value={r} className="text-gray-900 dark:text-white bg-white dark:bg-gray-900">{r}</option>
                    ))}
                  </select>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Permissions Matrix */}
      <section>
        <h2 className="text-lg font-bold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
          <ShieldAlert className="w-5 h-5" /> Permission Definitions
        </h2>
        <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-2xl shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="text-xs text-gray-500 dark:text-gray-400 uppercase bg-gray-50 dark:bg-gray-900/50 border-b border-gray-100 dark:border-gray-700">
                <tr>
                  <th className="px-6 py-4 font-medium">Feature / Capability</th>
                  <th className="px-6 py-4 font-medium text-center text-red-600 dark:text-red-400">Administrator</th>
                  <th className="px-6 py-4 font-medium text-center text-blue-600 dark:text-blue-400">Operator</th>
                  <th className="px-6 py-4 font-medium text-center text-gray-600 dark:text-gray-400">Auditor</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {PERMISSIONS_MATRIX.map((perm, idx) => (
                  <tr key={idx} className="hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors">
                    <td className="px-6 py-4 font-medium text-gray-900 dark:text-white">{perm.feature}</td>
                    <td className="px-6 py-4 text-center">
                      {perm.admin ? <Check className="w-5 h-5 text-green-500 mx-auto" /> : <X className="w-5 h-5 text-gray-300 dark:text-gray-600 mx-auto" />}
                    </td>
                    <td className="px-6 py-4 text-center">
                      {perm.operator ? <Check className="w-5 h-5 text-green-500 mx-auto" /> : <X className="w-5 h-5 text-gray-300 dark:text-gray-600 mx-auto" />}
                    </td>
                    <td className="px-6 py-4 text-center">
                      {perm.auditor ? <Check className="w-5 h-5 text-green-500 mx-auto" /> : <X className="w-5 h-5 text-gray-300 dark:text-gray-600 mx-auto" />}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="p-4 bg-gray-50 dark:bg-gray-900/50 text-xs text-gray-500 dark:text-gray-400 text-center border-t border-gray-100 dark:border-gray-700">
            * Root access to physical nodes via WebShell is logged for all roles.
          </div>
        </div>
      </section>

      {/* Profile Section */}
      <section>
        <h2 className="text-lg font-bold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
          <Shield className="w-5 h-5" /> Current Session
        </h2>
        <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-2xl shadow-sm p-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Display Name</label>
              <input type="text" value="System Administrator" readOnly className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-200" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Email Address</label>
              <input type="email" value="admin@os-baka.local" readOnly className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-200" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Authentication Method</label>
              <div className="flex items-center gap-2 text-sm text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20 p-2 rounded-lg border border-green-100 dark:border-green-900/30">
                <Lock className="w-4 h-4" /> MFA Token (Hardware Key)
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
};
