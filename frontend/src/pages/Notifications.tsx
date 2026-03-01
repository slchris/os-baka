
import React, { useEffect, useState } from 'react';
import { Bell, Info, AlertTriangle, AlertCircle, Check, Trash2 } from 'lucide-react';
import { BackendService } from '../services/backendService';
import { Notification } from '../types';

export const Notifications: React.FC = () => {
  const [notifications, setNotifications] = useState<Notification[]>([]);

  useEffect(() => {
    setNotifications(BackendService.getNotifications());
  }, []);

  const getIcon = (type: string) => {
    switch (type) {
      case 'info': return <Info className="w-5 h-5 text-blue-500" />;
      case 'warning': return <AlertTriangle className="w-5 h-5 text-orange-500" />;
      case 'error': return <AlertCircle className="w-5 h-5 text-red-500" />;
      case 'success': return <Check className="w-5 h-5 text-green-500" />;
      default: return <Bell className="w-5 h-5 text-gray-500" />;
    }
  };

  return (
    <div className="space-y-6 max-w-4xl animate-fade-in">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">Message Center</h2>
          <p className="text-gray-500 dark:text-gray-400">System alerts, updates, and security notifications.</p>
        </div>
        <button className="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 flex items-center gap-2">
          <Trash2 className="w-4 h-4" /> Clear All
        </button>
      </div>

      <div className="space-y-3">
        {notifications.length > 0 ? (
          notifications.map((notification) => (
            <div
              key={notification.id}
              className={`bg-white dark:bg-gray-800 border rounded-xl p-5 shadow-sm flex gap-4 transition-all hover:shadow-md ${!notification.read ? 'border-blue-200 dark:border-blue-900 bg-blue-50/30 dark:bg-blue-900/10' : 'border-gray-200 dark:border-gray-700'
                }`}
            >
              <div className="p-2 bg-white dark:bg-gray-700 rounded-full shadow-sm h-fit shrink-0">
                {getIcon(notification.type)}
              </div>
              <div className="flex-1">
                <div className="flex justify-between items-start">
                  <h3 className={`font-semibold text-gray-900 dark:text-white ${!notification.read && 'text-blue-900 dark:text-blue-300'}`}>
                    {notification.title}
                  </h3>
                  <span className="text-xs text-gray-400 whitespace-nowrap ml-4">
                    {notification.timestamp}
                  </span>
                </div>
                <p className="text-gray-600 dark:text-gray-300 text-sm mt-1 leading-relaxed">
                  {notification.message}
                </p>
              </div>
            </div>
          ))
        ) : (
          <div className="text-center py-20 bg-white dark:bg-gray-800 rounded-2xl border border-dashed border-gray-300 dark:border-gray-700">
            <div className="w-12 h-12 bg-gray-50 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-3">
              <Bell className="w-6 h-6 text-gray-300 dark:text-gray-500" />
            </div>
            <p className="text-gray-500 dark:text-gray-400 font-medium">All caught up!</p>
            <p className="text-gray-400 dark:text-gray-500 text-sm">No new notifications.</p>
          </div>
        )}
      </div>
    </div>
  );
};