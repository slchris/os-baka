<script lang="ts">
  import { onMount } from 'svelte'
  import { pxeApi, type PXEConfig } from '../lib/api'
  import Button from '../components/Button.svelte'
  import PXEConfigForm from '../components/PXEConfigForm.svelte'
  
  let configs: PXEConfig[] = []
  let loading = true
  let error: string | null = null
  let showCreateModal = false
  let editingConfig: PXEConfig | null = null
  let serviceStatus: any = null
  let loadingAction = false
  
  onMount(async () => {
    await Promise.all([loadConfigs(), loadServiceStatus()])
  })
  
  async function loadConfigs() {
    try {
      loading = true
      error = null
      const response = await pxeApi.list()
      configs = response.items
    } catch (err) {
      error = err instanceof Error ? err.message : '加载PXE配置失败'
      console.error('Failed to load PXE configs:', err)
    } finally {
      loading = false
    }
  }
  
  async function loadServiceStatus() {
    try {
      serviceStatus = await pxeApi.getServiceStatus()
    } catch (err) {
      console.error('Failed to load service status:', err)
    }
  }
  
  function handleCreateClick() {
    editingConfig = null
    showCreateModal = true
  }
  
  function handleEdit(config: PXEConfig) {
    editingConfig = config
    showCreateModal = true
  }
  
  async function handleDelete(id: number) {
    if (!confirm('确定要删除这个PXE配置吗？这将移除对应的dnsmasq配置文件。')) return
    
    try {
      await pxeApi.delete(id)
      await loadConfigs()
    } catch (err) {
      alert('删除失败: ' + (err instanceof Error ? err.message : '未知错误'))
    }
  }
  
  async function handleToggleEnabled(config: PXEConfig) {
    try {
      await pxeApi.update(config.id, { enabled: !config.enabled })
      await loadConfigs()
    } catch (err) {
      alert('更新失败: ' + (err instanceof Error ? err.message : '未知错误'))
    }
  }
  
  async function handleGenerateAll() {
    if (!confirm('确定要重新生成所有dnsmasq配置文件吗？')) return
    
    loadingAction = true
    try {
      const result = await pxeApi.generateAll()
      alert(`成功生成 ${result.details.length} 个配置文件`)
      await loadConfigs()
    } catch (err) {
      alert('生成失败: ' + (err instanceof Error ? err.message : '未知错误'))
    } finally {
      loadingAction = false
    }
  }
  
  async function handleValidate() {
    loadingAction = true
    try {
      const result = await pxeApi.validate()
      if (result.valid) {
        alert('✅ dnsmasq配置验证通过')
      } else {
        alert('❌ dnsmasq配置验证失败:\n' + result.errors.join('\n'))
      }
    } catch (err) {
      alert('验证失败: ' + (err instanceof Error ? err.message : '未知错误'))
    } finally {
      loadingAction = false
    }
  }
  
  async function handleRestartService() {
    if (!confirm('确定要重启dnsmasq服务吗？这可能会短暂中断网络引导。')) return
    
    loadingAction = true
    try {
      const result = await pxeApi.restartService()
      if (result.success) {
        alert('✅ dnsmasq服务重启成功')
        await loadServiceStatus()
      } else {
        alert('❌ dnsmasq服务重启失败: ' + result.message)
      }
    } catch (err) {
      alert('重启失败: ' + (err instanceof Error ? err.message : '未知错误'))
    } finally {
      loadingAction = false
    }
  }
  
  function handleFormSuccess() {
    loadConfigs()
  }
</script>

<div class="pxe-page">
  <div class="page-container">
    <header class="page-header">
      <div class="header-content">
        <h1 class="page-title">PXE 配置管理</h1>
        <p class="page-subtitle">管理网络启动配置和dnsmasq服务</p>
      </div>
      
      <div class="header-actions">
        <div class="service-status">
          {#if serviceStatus}
            <span class="status-indicator" class:active={serviceStatus.running}></span>
            <span class="status-text">
              {serviceStatus.running ? 'dnsmasq运行中' : 'dnsmasq已停止'}
            </span>
          {/if}
        </div>
        <Button on:click={handleCreateClick} variant="primary">
          + 添加配置
        </Button>
      </div>
    </header>
    
    <div class="action-bar">
      <Button on:click={handleGenerateAll} variant="secondary" disabled={loadingAction}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 16px; height: 16px; margin-right: 6px;">
          <polyline points="23 4 23 10 17 10"></polyline>
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
        </svg>
        重新生成配置
      </Button>
      <Button on:click={handleValidate} variant="secondary" disabled={loadingAction}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 16px; height: 16px; margin-right: 6px;">
          <polyline points="20 6 9 17 4 12"></polyline>
        </svg>
        验证配置
      </Button>
      <Button on:click={handleRestartService} variant="secondary" disabled={loadingAction}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 16px; height: 16px; margin-right: 6px;">
          <polyline points="23 4 23 10 17 10"></polyline>
          <polyline points="1 20 1 14 7 14"></polyline>
          <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
        </svg>
        重启服务
      </Button>
    </div>
    
    {#if loading}
      <div class="loading-state">
        <div class="spinner"></div>
        <p>加载中...</p>
      </div>
    {:else if error}
      <div class="error-state">
        <svg class="error-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="12" y1="8" x2="12" y2="12"></line>
          <line x1="12" y1="16" x2="12.01" y2="16"></line>
        </svg>
        <p class="error-message">{error}</p>
        <Button on:click={loadConfigs} variant="secondary">重试</Button>
      </div>
    {:else if configs.length === 0}
      <div class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
          <line x1="8" y1="21" x2="16" y2="21"></line>
          <line x1="12" y1="17" x2="12" y2="21"></line>
        </svg>
        <h3>暂无PXE配置</h3>
        <p>开始添加您的第一个PXE启动配置</p>
        <Button on:click={handleCreateClick} variant="primary">
          添加配置
        </Button>
      </div>
    {:else}
      <div class="config-table">
        <table>
          <thead>
            <tr>
              <th>状态</th>
              <th>主机名</th>
              <th>MAC地址</th>
              <th>IP地址</th>
              <th>操作系统</th>
              <th>已部署</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {#each configs as config}
              <tr class:disabled={!config.enabled}>
                <td>
                  <button
                    class="status-toggle"
                    class:enabled={config.enabled}
                    on:click={() => handleToggleEnabled(config)}
                    title={config.enabled ? '点击禁用' : '点击启用'}
                  >
                    {config.enabled ? '●' : '○'}
                  </button>
                </td>
                <td class="hostname">{config.hostname}</td>
                <td class="mac">{config.mac_address}</td>
                <td>{config.ip_address}</td>
                <td>{config.os_type}</td>
                <td>
                  <span class="deploy-badge" class:deployed={config.deployed}>
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 14px; height: 14px;">
                      {#if config.deployed}
                        <polyline points="20 6 9 17 4 12"></polyline>
                      {:else}
                        <line x1="5" y1="12" x2="19" y2="12"></line>
                      {/if}
                    </svg>
                  </span>
                </td>
                <td>
                  <div class="action-buttons">
                    <button class="action-btn edit" on:click={() => handleEdit(config)} title="编辑">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                      </svg>
                    </button>
                    <button class="action-btn delete" on:click={() => handleDelete(config.id)} title="删除">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6"></polyline>
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                      </svg>
                    </button>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>

<PXEConfigForm 
  bind:show={showCreateModal} 
  config={editingConfig}
  on:success={handleFormSuccess} 
/>

<style lang="scss">
  .pxe-page {
    min-height: 100vh;
    background: var(--secondaryBackground);
  }
  
  .page-container {
    max-width: 1400px;
    margin: 0 auto;
    padding: var(--spacing-3xl) var(--spacing-lg);
  }
  
  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: var(--spacing-xl);
    flex-wrap: wrap;
    gap: var(--spacing-lg);
  }
  
  .header-content {
    .page-title {
      font: var(--title-1);
      color: var(--systemPrimary);
      margin: 0 0 var(--spacing-xs);
    }
    
    .page-subtitle {
      font: var(--body);
      color: var(--systemSecondary);
      margin: 0;
    }
  }
  
  .header-actions {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
  }
  
  .service-status {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
    padding: var(--spacing-sm) var(--spacing-md);
    background: var(--defaultBackground);
    border-radius: var(--radius-sm);
    font: var(--callout);
  }
  
  .status-indicator {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--systemTertiary);
    
    &.active {
      background: #34c759;
      box-shadow: 0 0 8px rgba(52, 199, 89, 0.4);
    }
  }
  
  .action-bar {
    display: flex;
    gap: var(--spacing-md);
    margin-bottom: var(--spacing-xl);
  }
  
  .config-table {
    background: var(--defaultBackground);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-card);
    overflow: hidden;
    
    table {
      width: 100%;
      border-collapse: collapse;
    }
    
    thead {
      background: rgba(0, 0, 0, 0.02);
      
      @media (prefers-color-scheme: dark) {
        background: rgba(255, 255, 255, 0.05);
      }
      
      th {
        text-align: left;
        padding: var(--spacing-md) var(--spacing-lg);
        font: var(--callout);
        font-weight: 600;
        color: var(--systemSecondary);
        border-bottom: 1px solid rgba(0, 0, 0, 0.1);
        
        @media (prefers-color-scheme: dark) {
          border-bottom-color: rgba(255, 255, 255, 0.1);
        }
      }
    }
    
    tbody {
      tr {
        border-bottom: 1px solid rgba(0, 0, 0, 0.05);
        transition: background var(--transition-fast);
        
        &:hover {
          background: rgba(0, 0, 0, 0.02);
          
          @media (prefers-color-scheme: dark) {
            background: rgba(255, 255, 255, 0.03);
          }
        }
        
        &.disabled {
          opacity: 0.5;
        }
        
        @media (prefers-color-scheme: dark) {
          border-bottom-color: rgba(255, 255, 255, 0.05);
        }
      }
      
      td {
        padding: var(--spacing-md) var(--spacing-lg);
        font: var(--body);
        color: var(--systemPrimary);
        
        &.hostname {
          font-weight: 500;
        }
        
        &.mac {
          font-family: 'SF Mono', Monaco, 'Courier New', monospace;
          font-size: 13px;
          color: var(--systemSecondary);
        }
      }
    }
  }
  
  .status-toggle {
    background: none;
    border: none;
    font-size: 20px;
    cursor: pointer;
    padding: 0;
    color: var(--systemTertiary);
    transition: color var(--transition-fast);
    
    &.enabled {
      color: #34c759;
    }
    
    &:hover {
      transform: scale(1.2);
    }
  }
  
  .deploy-badge {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
    background: rgba(0, 0, 0, 0.05);
    color: var(--systemTertiary);
    
    &.deployed {
      background: rgba(52, 199, 89, 0.1);
      color: #34c759;
    }
  }
  
  .action-buttons {
    display: flex;
    gap: var(--spacing-xs);
  }
  
  .action-btn {
    background: none;
    border: 1px solid rgba(0, 0, 0, 0.1);
    border-radius: var(--radius-sm);
    padding: var(--spacing-xs);
    cursor: pointer;
    color: var(--systemSecondary);
    transition: all var(--transition-fast);
    
    svg {
      width: 16px;
      height: 16px;
      display: block;
    }
    
    &:hover {
      background: rgba(0, 0, 0, 0.05);
    }
    
    &.edit:hover {
      border-color: var(--keyColor);
      color: var(--keyColor);
    }
    
    &.delete:hover {
      border-color: #ff3b30;
      color: #ff3b30;
      background: rgba(255, 59, 48, 0.05);
    }
  }
  
  .loading-state,
  .error-state,
  .empty-state {
    text-align: center;
    padding: var(--spacing-3xl);
    background: var(--defaultBackground);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-card);
  }
  
  .loading-state {
    .spinner {
      width: 40px;
      height: 40px;
      margin: 0 auto var(--spacing-md);
      border: 4px solid rgba(0, 0, 0, 0.1);
      border-top-color: var(--keyColor);
      border-radius: 50%;
      animation: spin 1s linear infinite;
    }
  }
  
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
  
  .error-state {
    .error-icon {
      width: 48px;
      height: 48px;
      margin: 0 auto var(--spacing-md);
      color: #ff3b30;
    }
  }
  
  .empty-state {
    .empty-icon {
      width: 80px;
      height: 80px;
      margin: 0 auto var(--spacing-lg);
      color: var(--systemTertiary);
    }
    
    h3 {
      font: var(--title-2);
      color: var(--systemPrimary);
      margin: 0 0 var(--spacing-sm);
    }
    
    p {
      font: var(--body);
      color: var(--systemSecondary);
      margin: 0 0 var(--spacing-lg);
    }
  }
</style>
