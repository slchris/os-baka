
import React, { useState, useEffect, useRef } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Sidebar } from './components/Sidebar';
import { Bell, Search, User, LogOut, Moon, Sun } from 'lucide-react';
import { BackendService } from './services/backendService';
import { AuthApi, NotificationsApi } from './services/apiClient';

const App: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();

  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);
  const userMenuRef = useRef<HTMLDivElement>(null);
  const [userName, setUserName] = useState('User');
  const [userRole, setUserRole] = useState('');
  const [unreadCount, setUnreadCount] = useState(0);

  // Theme State
  const [isDarkMode, setIsDarkMode] = useState(() => {
    const saved = localStorage.getItem('osbaka_theme');
    return saved === 'dark' || (!saved && window.matchMedia('(prefers-color-scheme: dark)').matches);
  });

  useEffect(() => {
    const root = window.document.documentElement;
    if (isDarkMode) {
      root.classList.add('dark');
      localStorage.setItem('osbaka_theme', 'dark');
    } else {
      root.classList.remove('dark');
      localStorage.setItem('osbaka_theme', 'light');
    }
  }, [isDarkMode]);

  const toggleTheme = () => {
    setIsDarkMode(!isDarkMode);
  };

  // Close user menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (userMenuRef.current && !userMenuRef.current.contains(event.target as Node)) {
        setIsUserMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Fetch user info and notification count
  useEffect(() => {
    AuthApi.me()
      .then((user) => {
        setUserName(user.full_name || user.username || 'User');
        setUserRole(user.is_superuser ? 'Administrator' : 'Operator');
      })
      .catch(() => { });
    NotificationsApi.list()
      .then((notifications) => {
        const unread = notifications.filter((n) => !n.read).length;
        setUnreadCount(unread);
      })
      .catch(() => { });
  }, []);

  const handleLogout = () => {
    BackendService.logout();
    setIsUserMenuOpen(false);
    navigate('/login');
  };

  const getPageTitle = () => {
    const path = location.pathname;
    if (path.includes('/nodes')) return 'Nodes';
    if (path.includes('/keyvault')) return 'Key Vault';
    if (path.includes('/webshell')) return 'WebShell';
    if (path.includes('/settings')) return 'Settings';
    if (path.includes('/documentation')) return 'Documentation';
    if (path.includes('/notifications')) return 'Notifications';
    return 'Dashboard';
  };

  return (
    <div className="min-h-screen bg-[#f5f5f7] dark:bg-[#121212] transition-colors duration-300">
      <Sidebar />

      <main className="pl-64 transition-all duration-300">
        {/* Header */}
        <header className="sticky top-0 z-10 bg-[#f5f5f7]/90 dark:bg-[#121212]/90 backdrop-blur-md border-b border-gray-200 dark:border-gray-800 px-8 py-4 flex justify-between items-center transition-colors duration-300">
          <div className="flex flex-col">
            <span className="text-xs text-gray-500 dark:text-gray-400 font-medium">Infrastructure / {getPageTitle()}</span>
          </div>

          <div className="flex items-center gap-4">
            <div className="relative hidden md:block">
              <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input
                type="text"
                placeholder="Search nodes..."
                className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-full pl-10 pr-4 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 w-64 transition-all text-gray-900 dark:text-gray-200 placeholder-gray-400"
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    const value = (e.target as HTMLInputElement).value.trim();
                    if (value) {
                      navigate(`/nodes?search=${encodeURIComponent(value)}`);
                    } else {
                      navigate('/nodes');
                    }
                  }
                }}
              />
            </div>

            <button
              onClick={toggleTheme}
              className="p-2 rounded-full text-gray-500 dark:text-gray-400 hover:bg-white dark:hover:bg-gray-800 transition-colors"
            >
              {isDarkMode ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
            </button>

            <button
              onClick={() => navigate('/notifications')}
              className={`relative p-2 rounded-full transition-colors ${location.pathname === '/notifications' ? 'bg-white dark:bg-gray-800 shadow-sm text-blue-600 dark:text-blue-400' : 'text-gray-500 dark:text-gray-400 hover:bg-white dark:hover:bg-gray-800'}`}
            >
              <Bell className="w-5 h-5" />
              {unreadCount > 0 && (
                <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-red-500 rounded-full border-2 border-[#f5f5f7] dark:border-[#121212]"></span>
              )}
            </button>

            <div className="flex items-center gap-3 pl-4 border-l border-gray-200 dark:border-gray-800 relative" ref={userMenuRef}>
              <div className="text-right hidden sm:block">
                <p className="text-sm font-medium text-gray-900 dark:text-gray-200">{userName}</p>
                <p className="text-xs text-gray-500 dark:text-gray-400">{userRole}</p>
              </div>
              <div
                className="w-8 h-8 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center overflow-hidden border border-white dark:border-gray-600 shadow-sm cursor-pointer hover:ring-2 ring-gray-200 dark:ring-gray-700 transition-all"
                onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
              >
                <User className="w-5 h-5 text-gray-500 dark:text-gray-300" />
              </div>

              {/* User Dropdown */}
              {isUserMenuOpen && (
                <div className="absolute top-full right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-xl shadow-xl border border-gray-100 dark:border-gray-700 overflow-hidden py-1 animate-in fade-in slide-in-from-top-2 duration-200 z-50">
                  <button
                    onClick={() => { navigate('/settings'); setIsUserMenuOpen(false); }}
                    className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center gap-2"
                  >
                    <User className="w-4 h-4" /> Profile
                  </button>
                  <div className="border-t border-gray-100 dark:border-gray-700 my-1"></div>
                  <button
                    onClick={handleLogout}
                    className="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 flex items-center gap-2"
                  >
                    <LogOut className="w-4 h-4" /> Sign Out
                  </button>
                </div>
              )}
            </div>
          </div>
        </header>

        {/* Content Area - Render child routes */}
        <div className="p-8 max-w-7xl mx-auto">
          <Outlet />
        </div>
      </main>
    </div>
  );
};

export default App;