import React, { useState, useEffect, useRef } from 'react';
import {
  Terminal as TerminalIcon,
  Server,
  Lock,
  ChevronRight,
  ChevronDown,
  X,
  AlertCircle,
  CheckCircle2,
  Loader2,
  Eye,
  EyeOff,
  Folder,
  FileKey
} from 'lucide-react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { WebLinksAddon } from 'xterm-addon-web-links';
import 'xterm/css/xterm.css';
import { BackendService } from '../services/backendService';
import { NodeConfig, NodeStatus } from '../types';

type AuthMethod = 'password' | 'key';

interface ConnectionState {
  status: 'disconnected' | 'connecting' | 'connected' | 'error';
  message?: string;
}

export const WebShell: React.FC = () => {
  const [nodes, setNodes] = useState<NodeConfig[]>([]);
  const [selectedNode, setSelectedNode] = useState<NodeConfig | null>(null);
  const [showConnectionModal, setShowConnectionModal] = useState(false);

  const [connectionState, setConnectionState] = useState<ConnectionState>({ status: 'disconnected' });
  const [authMethod, setAuthMethod] = useState<AuthMethod>('password');

  const [username, setUsername] = useState('root');
  const [password, setPassword] = useState('');
  const [privateKey, setPrivateKey] = useState('');
  const [keyPassphrase, setKeyPassphrase] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showPassphrase, setShowPassphrase] = useState(false);
  const [sshPort, setSshPort] = useState(22);

  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set(['active', 'pending', 'offline']));

  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    loadNodes();
  }, []);

  const loadNodes = async () => {
    const loadedNodes = await BackendService.getNodes();
    setNodes(loadedNodes as NodeConfig[]);
  };

  const groupedNodes: Record<string, NodeConfig[]> = nodes.reduce((acc, node) => {
    const status = node.status.toLowerCase();
    if (!acc[status]) {
      acc[status] = [];
    }
    acc[status].push(node);
    return acc;
  }, {} as Record<string, NodeConfig[]>);

  const toggleGroup = (group: string) => {
    const newExpanded = new Set(expandedGroups);
    if (newExpanded.has(group)) {
      newExpanded.delete(group);
    } else {
      newExpanded.add(group);
    }
    setExpandedGroups(newExpanded);
  };

  const handleNodeSelect = (node: NodeConfig) => {
    setSelectedNode(node);
    setShowConnectionModal(true);

    if (node.ssh) {
      setUsername(node.ssh.username || 'root');
      setSshPort(node.ssh.port || 22);
      if (node.ssh.privateKey) {
        setAuthMethod('key');
        setPrivateKey(node.ssh.privateKey);
      }
    }
  };

  const initializeTerminal = () => {
    if (!terminalRef.current) return;

    if (xtermRef.current) {
      xtermRef.current.dispose();
    }

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
        cursor: '#00ff00',
        black: '#000000',
        red: '#cd3131',
        green: '#0dbc79',
        yellow: '#e5e510',
        blue: '#2472c8',
        magenta: '#bc3fbc',
        cyan: '#11a8cd',
        white: '#e5e5e5',
        brightBlack: '#666666',
        brightRed: '#f14c4c',
        brightGreen: '#23d18b',
        brightYellow: '#f5f543',
        brightBlue: '#3b8eea',
        brightMagenta: '#d670d6',
        brightCyan: '#29b8db',
        brightWhite: '#e5e5e5',
      },
    });

    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();

    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);
    term.open(terminalRef.current);

    fitAddon.fit();

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    term.onData((data) => {
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({
          type: 'input',
          data: data
        }));
      }
    });

    const handleResize = () => {
      if (fitAddonRef.current && xtermRef.current) {
        fitAddon.fit();
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
          wsRef.current.send(JSON.stringify({
            type: 'resize',
            cols: xtermRef.current.cols,
            rows: xtermRef.current.rows
          }));
        }
      }
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  };

  const connectToNode = async () => {
    if (!selectedNode) return;

    setConnectionState({ status: 'connecting', message: 'Connecting to node...' });

    initializeTerminal();

    try {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.hostname}:8000/api/v1/ws/ssh`;

      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        const authMessage = {
          type: 'auth',
          host: selectedNode.ipAddress,
          port: sshPort,
          username: username,
          method: authMethod,
          credential: authMethod === 'password' ? password : privateKey,
          ...(authMethod === 'key' && keyPassphrase ? { keyPassphrase } : {})
        };

        ws.send(JSON.stringify(authMessage));
      };

      ws.onmessage = (event) => {
        const message = JSON.parse(event.data);

        if (message.type === 'connected') {
          setConnectionState({ status: 'connected', message: 'Connected successfully' });
          setShowConnectionModal(false);

          setPassword('');
          setPrivateKey('');
          setKeyPassphrase('');

        } else if (message.type === 'output') {
          const output = atob(message.data);
          xtermRef.current?.write(output);

        } else if (message.type === 'error') {
          setConnectionState({ status: 'error', message: message.message });
          xtermRef.current?.writeln(`\r\n\x1b[31mError: ${message.message}\x1b[0m\r\n`);

        } else if (message.type === 'disconnected') {
          setConnectionState({ status: 'disconnected', message: 'Connection closed' });
          xtermRef.current?.writeln(`\r\n\x1b[33mConnection closed.\x1b[0m\r\n`);
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        setConnectionState({ status: 'error', message: 'WebSocket connection failed' });
      };

      ws.onclose = () => {
        setConnectionState({ status: 'disconnected', message: 'Connection closed' });
      };

    } catch (error) {
      setConnectionState({
        status: 'error',
        message: error instanceof Error ? error.message : 'Connection failed'
      });
    }
  };

  const disconnect = () => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    if (xtermRef.current) {
      xtermRef.current.dispose();
      xtermRef.current = null;
    }

    setConnectionState({ status: 'disconnected' });
    setSelectedNode(null);
  };

  const getStatusColor = (status: NodeStatus) => {
    switch (status) {
      case NodeStatus.ACTIVE:
        return 'text-green-500';
      case NodeStatus.PENDING:
        return 'text-yellow-500';
      case NodeStatus.OFFLINE:
      case NodeStatus.ERROR:
        return 'text-red-500';
      default:
        return 'text-gray-500';
    }
  };

  const getStatusIcon = (status: NodeStatus) => {
    const color = getStatusColor(status);
    return <div className={`w-2 h-2 rounded-full ${color.replace('text-', 'bg-')}`} />;
  };

  return (
    <div className="h-[calc(100vh-8rem)] flex gap-4">
      <div className="w-80 bg-white dark:bg-gray-800 rounded-2xl shadow-lg border border-gray-200 dark:border-gray-700 overflow-hidden flex flex-col">
        <div className="p-4 border-b border-gray-200 dark:border-gray-700">
          <h3 className="font-bold text-gray-900 dark:text-white flex items-center gap-2">
            <Server className="w-5 h-5" />
            Available Nodes
          </h3>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
            Select a node to connect
          </p>
        </div>

        <div className="flex-1 overflow-y-auto p-2">
          {Object.entries(groupedNodes).map(([group, groupNodes]) => (
            <div key={group} className="mb-2">
              <button
                onClick={() => toggleGroup(group)}
                className="w-full flex items-center gap-2 px-3 py-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors text-left"
              >
                {expandedGroups.has(group) ? (
                  <ChevronDown className="w-4 h-4 text-gray-500" />
                ) : (
                  <ChevronRight className="w-4 h-4 text-gray-500" />
                )}
                <Folder className="w-4 h-4 text-blue-500" />
                <span className="font-medium text-gray-700 dark:text-gray-300 capitalize">
                  {group}
                </span>
                <span className="ml-auto text-xs text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded-full">
                  {groupNodes.length}
                </span>
              </button>

              {expandedGroups.has(group) && (
                <div className="ml-6 mt-1 space-y-1">
                  {groupNodes.map((node) => (
                    <button
                      key={node.id}
                      onClick={() => handleNodeSelect(node)}
                      className={`w-full flex items-center gap-2 px-3 py-2 rounded-lg transition-all text-left ${selectedNode?.id === node.id
                        ? 'bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800'
                        : 'hover:bg-gray-50 dark:hover:bg-gray-700/50'
                        }`}
                    >
                      {getStatusIcon(node.status)}
                      <Server className="w-4 h-4 text-gray-500" />
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-sm text-gray-900 dark:text-white truncate">
                          {node.hostname}
                        </div>
                        <div className="text-xs text-gray-500 dark:text-gray-400 font-mono">
                          {node.ipAddress}
                        </div>
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </div>
          ))}

          {nodes.length === 0 && (
            <div className="text-center py-12 text-gray-500 dark:text-gray-400">
              <Server className="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p className="text-sm">No nodes available</p>
              <p className="text-xs mt-1">Add nodes in the Nodes page</p>
            </div>
          )}
        </div>
      </div>

      <div className="flex-1 bg-[#1e1e1e] rounded-2xl overflow-hidden flex flex-col shadow-2xl border border-gray-800">
        <div className="bg-[#2d2d2d] px-4 py-2 flex justify-between items-center border-b border-[#3e3e3e]">
          <div className="flex items-center gap-3">
            <TerminalIcon className="w-4 h-4 text-green-400" />
            <span className="text-gray-300 text-sm font-mono">
              {connectionState.status === 'connected' && selectedNode
                ? `${username}@${selectedNode.hostname}`
                : 'Not Connected'}
            </span>

            {connectionState.status === 'connected' && (
              <div className="flex items-center gap-1 text-green-400 text-xs">
                <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse" />
                <span>Connected</span>
              </div>
            )}

            {connectionState.status === 'connecting' && (
              <div className="flex items-center gap-1 text-yellow-400 text-xs">
                <Loader2 className="w-3 h-3 animate-spin" />
                <span>Connecting...</span>
              </div>
            )}
          </div>

          <div className="flex items-center gap-2">
            {connectionState.status === 'connected' && (
              <button
                onClick={disconnect}
                className="px-3 py-1 text-xs bg-red-500/20 text-red-400 rounded-lg hover:bg-red-500/30 transition-colors flex items-center gap-1"
              >
                <X className="w-3 h-3" />
                Disconnect
              </button>
            )}
            <div className="flex gap-2">
              <div className="w-3 h-3 rounded-full bg-yellow-500 hover:opacity-80 cursor-pointer" />
              <div className="w-3 h-3 rounded-full bg-green-500 hover:opacity-80 cursor-pointer" />
              <div className="w-3 h-3 rounded-full bg-red-500 hover:opacity-80 cursor-pointer" onClick={disconnect} />
            </div>
          </div>
        </div>

        <div className="flex-1 p-4 overflow-hidden">
          {connectionState.status === 'disconnected' ? (
            <div className="h-full flex items-center justify-center text-gray-500">
              <div className="text-center">
                <TerminalIcon className="w-16 h-16 mx-auto mb-4 opacity-50" />
                <p className="text-lg font-medium">Select a node to start SSH session</p>
                <p className="text-sm mt-2 opacity-75">Choose a node from the sidebar</p>
              </div>
            </div>
          ) : (
            <div ref={terminalRef} className="h-full" />
          )}
        </div>
      </div>

      {showConnectionModal && selectedNode && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
          <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl max-w-md w-full">
            <div className="p-5 border-b border-gray-200 dark:border-gray-700">
              <div className="flex justify-between items-center">
                <div>
                  <h3 className="font-bold text-gray-900 dark:text-white text-lg">
                    Connect to {selectedNode.hostname}
                  </h3>
                  <p className="text-sm text-gray-500 dark:text-gray-400 font-mono mt-1">
                    {selectedNode.ipAddress}
                  </p>
                </div>
                <button
                  onClick={() => setShowConnectionModal(false)}
                  className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
            </div>

            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Authentication Method
                </label>
                <div className="grid grid-cols-2 gap-3">
                  <button
                    onClick={() => setAuthMethod('password')}
                    className={`p-3 rounded-lg border flex flex-col items-center gap-2 transition-all ${authMethod === 'password'
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400'
                      : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:bg-gray-50 dark:hover:bg-gray-700'
                      }`}
                  >
                    <Lock className="w-5 h-5" />
                    <span className="text-xs font-bold">Password</span>
                  </button>
                  <button
                    onClick={() => setAuthMethod('key')}
                    className={`p-3 rounded-lg border flex flex-col items-center gap-2 transition-all ${authMethod === 'key'
                      ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20 text-purple-600 dark:text-purple-400'
                      : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:bg-gray-50 dark:hover:bg-gray-700'
                      }`}
                  >
                    <FileKey className="w-5 h-5" />
                    <span className="text-xs font-bold">SSH Key</span>
                  </button>
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                  Username
                </label>
                <input
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                  placeholder="root"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                  SSH Port
                </label>
                <input
                  type="number"
                  value={sshPort}
                  onChange={(e) => setSshPort(parseInt(e.target.value))}
                  className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                />
              </div>

              {authMethod === 'password' && (
                <div>
                  <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                    Password
                  </label>
                  <div className="relative">
                    <input
                      type={showPassword ? 'text' : 'password'}
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                      placeholder="Enter password"
                      autoComplete="new-password"
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                    >
                      {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </button>
                  </div>
                </div>
              )}

              {authMethod === 'key' && (
                <>
                  <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                      Private Key
                    </label>
                    <textarea
                      value={privateKey}
                      onChange={(e) => setPrivateKey(e.target.value)}
                      className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-xs dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500/20 font-mono"
                      rows={6}
                      placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                      Key Passphrase (Optional)
                    </label>
                    <div className="relative">
                      <input
                        type={showPassphrase ? 'text' : 'password'}
                        value={keyPassphrase}
                        onChange={(e) => setKeyPassphrase(e.target.value)}
                        className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-sm dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500/20"
                        placeholder="Leave empty if no passphrase"
                        autoComplete="new-password"
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassphrase(!showPassphrase)}
                        className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                      >
                        {showPassphrase ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                      </button>
                    </div>
                  </div>
                </>
              )}

              {connectionState.status === 'error' && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3 flex items-start gap-2">
                  <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
                  <div className="flex-1">
                    <p className="text-sm font-medium text-red-900 dark:text-red-300">Connection Failed</p>
                    <p className="text-xs text-red-700 dark:text-red-400 mt-1">{connectionState.message}</p>
                  </div>
                </div>
              )}

              <div className="pt-4 flex justify-end gap-2">
                <button
                  onClick={() => setShowConnectionModal(false)}
                  className="px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={connectToNode}
                  disabled={connectionState.status === 'connecting' || (authMethod === 'password' && !password) || (authMethod === 'key' && !privateKey)}
                  className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {connectionState.status === 'connecting' ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Connecting...
                    </>
                  ) : (
                    <>
                      <CheckCircle2 className="w-4 h-4" />
                      Connect
                    </>
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
