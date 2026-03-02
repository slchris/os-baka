
import React from 'react';
import { NavLink } from 'react-router-dom';
import { LayoutDashboard, Server, Key, Terminal, Settings, Network, BookOpen, ClipboardList } from 'lucide-react';

export const Sidebar: React.FC = () => {
  const menuItems = [
    { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard, path: '/dashboard' },
    { id: 'nodes', label: 'Nodes & Config', icon: Server, path: '/nodes' },
    { id: 'dhcp', label: 'DHCP Settings', icon: Network, path: '/dhcp' },
    { id: 'keyvault', label: 'Key Vault', icon: Key, path: '/keyvault' },
    { id: 'webshell', label: 'WebShell', icon: Terminal, path: '/webshell' },
    { id: 'audit', label: 'Audit Logs', icon: ClipboardList, path: '/audit' },
    { id: 'settings', label: 'Settings', icon: Settings, path: '/settings' },
    { id: 'documentation', label: 'Documentation', icon: BookOpen, path: '/documentation' },
  ];

  return (
    <aside className="w-64 bg-white/80 dark:bg-gray-900/80 backdrop-blur-xl h-screen fixed left-0 top-0 border-r border-gray-200 dark:border-gray-800 z-20 flex flex-col transition-colors duration-300">
      <div className="p-6 flex items-center gap-3">
        <div className="w-8 h-8 bg-black dark:bg-white rounded-lg flex items-center justify-center text-white dark:text-black font-bold text-sm transition-colors">
          OB
        </div>
        <h1 className="text-xl font-bold tracking-tight text-gray-900 dark:text-white">OS-Baka</h1>
      </div>

      <nav className="flex-1 px-4 space-y-1">
        {menuItems.map((item) => {
          const Icon = item.icon;
          return (
            <NavLink
              key={item.id}
              to={item.path}
              className={({ isActive }) =>
                `w-full flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-all duration-200 group ${isActive
                  ? 'bg-black dark:bg-white text-white dark:text-black shadow-lg shadow-gray-200 dark:shadow-none'
                  : 'text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white'
                }`
              }
            >
              {({ isActive }) => (
                <>
                  <Icon className={`w-5 h-5 ${isActive ? 'text-white dark:text-black' : 'text-gray-400 group-hover:text-gray-600 dark:group-hover:text-gray-200'}`} />
                  {item.label}
                </>
              )}
            </NavLink>
          );
        })}
      </nav>

      <div className="p-4 border-t border-gray-200 dark:border-gray-800">
        <div className="flex items-center gap-3 px-4 py-3 rounded-xl bg-gray-50 dark:bg-gray-800 border border-gray-100 dark:border-gray-700 transition-colors">
          <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse"></div>
          <span className="text-xs font-medium text-gray-500 dark:text-gray-400">System Operational</span>
        </div>
      </div>
    </aside>
  );
};