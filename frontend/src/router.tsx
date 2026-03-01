import React from 'react';
import { createBrowserRouter, Navigate } from 'react-router-dom';
import App from './App';
import { Login } from './pages/Login';
import { Dashboard } from './pages/Dashboard';
import { Nodes } from './pages/Nodes';
import { KeyVault } from './pages/KeyVault';
import { Settings } from './pages/Settings';
import { DHCPSettings } from './pages/DHCPSettings';
import { WebShell } from './pages/WebShell';
import { Notifications } from './pages/Notifications';
import { Documentation } from './pages/Documentation';

/**
 * Protected Route Component
 * 检查用户是否已登录，未登录则重定向到登录页
 */
interface ProtectedRouteProps {
  children: React.ReactNode;
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ children }) => {
  const token = localStorage.getItem('access_token');

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

/**
 * Router Configuration
 * 路由配置 - 定义所有页面路径和组件映射
 */
export const router = createBrowserRouter([
  {
    path: '/login',
    element: <Login />,
  },
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <App />
      </ProtectedRoute>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="/dashboard" replace />,
      },
      {
        path: 'dashboard',
        element: <Dashboard />,
      },
      {
        path: 'nodes',
        element: <Nodes />,
      },
      {
        path: 'keyvault',
        element: <KeyVault />,
      },
      {
        path: 'webshell',
        element: <WebShell />,
      },
      {
        path: 'notifications',
        element: <Notifications />,
      },
      {
        path: 'documentation',
        element: <Documentation />,
      },
      {
        path: 'settings',
        element: <Settings />,
      },
      {
        path: 'dhcp',
        element: <DHCPSettings />,
      },
      {
        path: '*',
        element: <Navigate to="/dashboard" replace />,
      },
    ],
  },
  {
    path: '*',
    element: <Navigate to="/login" replace />,
  },
]);

export default router;
