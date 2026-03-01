
import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { BackendService } from '../services/backendService';
import { ShieldCheck, ArrowRight, Loader2 } from 'lucide-react';

export const Login: React.FC = () => {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');

    try {
      await BackendService.login(username, password);
      navigate('/dashboard');
    } catch (err) {
      setError('用户名或密码错误，请重试。默认凭据：admin / admin123');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-[#f5f5f7] dark:bg-[#121212] p-4 transition-colors duration-300">
      <div className="w-full max-w-md bg-white dark:bg-gray-800 rounded-3xl shadow-xl overflow-hidden border border-gray-100 dark:border-gray-700">
        <div className="p-8 md:p-12">
          <div className="flex justify-center mb-8">
            <div className="w-12 h-12 bg-black dark:bg-white rounded-xl flex items-center justify-center text-white dark:text-black">
              <ShieldCheck className="w-7 h-7" />
            </div>
          </div>
          
          <h2 className="text-2xl font-bold text-center text-gray-900 dark:text-white mb-2">OS-Baka</h2>
          <p className="text-center text-gray-500 dark:text-gray-400 mb-8">Secure Deployment Management</p>

          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">用户名</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="admin"
                className="w-full px-4 py-3 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-black/10 dark:focus:ring-white/10 transition-all placeholder-gray-400 dark:placeholder-gray-600"
                required
                autoComplete="username"
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">密码</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                className="w-full px-4 py-3 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-black/10 dark:focus:ring-white/10 transition-all placeholder-gray-400 dark:placeholder-gray-600"
                required
                autoComplete="current-password"
              />
            </div>

            {error && (
              <p className="text-sm text-red-500 dark:text-red-400 text-center bg-red-50 dark:bg-red-900/20 py-2 rounded-lg">{error}</p>
            )}

            <button
              type="submit"
              disabled={isLoading}
              className="w-full bg-black dark:bg-white text-white dark:text-black py-3 rounded-xl font-medium hover:bg-gray-800 dark:hover:bg-gray-200 transition-colors flex items-center justify-center gap-2 disabled:opacity-70"
            >
              {isLoading ? <Loader2 className="w-5 h-5 animate-spin" /> : '登录'}
              {!isLoading && <ArrowRight className="w-4 h-4" />}
            </button>
          </form>
          
          <div className="mt-4 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-100 dark:border-blue-800">
            <p className="text-xs text-blue-600 dark:text-blue-400 text-center">
              💡 默认凭据：<span className="font-mono font-semibold">admin</span> / <span className="font-mono font-semibold">admin123</span>
            </p>
          </div>
        </div>
        <div className="bg-gray-50 dark:bg-gray-900 p-4 text-center">
          <p className="text-xs text-gray-400 dark:text-gray-500">Protected by OS-Baka Integrity Shield</p>
        </div>
      </div>
    </div>
  );
};