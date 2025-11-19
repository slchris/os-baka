<script lang="ts">
  import { onMount } from 'svelte'
  import { pxeServiceApi, type PXEServiceConfig } from '../lib/api'
  
  let configs: PXEServiceConfig[] = []
  let loading = true
  let error: string | null = null
  let isAdmin = false
  
  // Modals
  let showConfigModal = false
  let showLogsModal = false
  let selectedConfig: PXEServiceConfig | null = null
  let containerLogs = ''
  let loadingAction = false
  
  // Edit form
  let editForm: Partial<PXEServiceConfig> = {}
  
  onMount(async () => {
    await checkAdmin()
    await loadConfigs()
  })
  
  async function checkAdmin() {
    try {
      const token = localStorage.getItem('token')
      if (!token) return
      const payload = JSON.parse(atob(token.split('.')[1]))
      isAdmin = payload.is_superuser === true
    } catch (err) {
      console.error('Failed to check admin:', err)
    }
  }
  
  async function loadConfigs() {
    try {
      loading = true
      error = null
      const data = await pxeServiceApi.getConfig()
      // Backend returns single config, wrap in array for list view
      configs = data ? [data] : []
    } catch (err) {
      error = err instanceof Error ? err.message : '加载失败'
      console.error('Failed to load configs:', err)
    } finally {
      loading = false
    }
  }
  
  async function handleStart(config: PXEServiceConfig) {
    loadingAction = true
    try {
      await pxeServiceApi.start()
      await loadConfigs()
      alert('PXE服务启动成功')
    } catch (err) {
      alert('启动失败: ' + (err instanceof Error ? err.message : '未知错误'))
    } finally {
      loadingAction = false
    }
  }
  
  async function handleStop(config: PXEServiceConfig) {
    if (!confirm('确定要停止PXE服务吗？')) return
    loadingAction = true
    try {
      await pxeServiceApi.stop()
      await loadConfigs()
      alert('PXE服务已停止')
    } catch (err) {
      alert('停止失败: ' + (err instanceof Error ? err.message : '未知错误'))
    } finally {
      loadingAction = false
    }
  }
  
  async function handleRestart(config: PXEServiceConfig) {
    loadingAction = true
    try {
      await pxeServiceApi.restart()
      await loadConfigs()
      alert('PXE服务已重启')
    } catch (err) {
      alert('重启失败: ' + (err instanceof Error ? err.message : '未知错误'))
    } finally {
      loadingAction = false
    }
  }
  
  async function openConfigModal(config: PXEServiceConfig) {
    selectedConfig = config
    editForm = {
      server_ip: config.server_ip,
      dhcp_range_start: config.dhcp_range_start,
      dhcp_range_end: config.dhcp_range_end,
      netmask: config.netmask,
      gateway: config.gateway,
      dns_servers: config.dns_servers,
      enable_bios: config.enable_bios,
      enable_uefi: config.enable_uefi,
      enable_encrypted: config.enable_encrypted,
      enable_unencrypted: config.enable_unencrypted,
      bios_boot_file: config.bios_boot_file,
      uefi_boot_file: config.uefi_boot_file,
      debian_version: config.debian_version,
      debian_mirror: config.debian_mirror,
      default_encryption: config.default_encryption,
      default_username: config.default_username,
      tftp_root: config.tftp_root,
      container_image: config.container_image
    }
    showConfigModal = true
  }
  
  async function handleSaveConfig() {
    if (!selectedConfig) return
    loadingAction = true
    try {
      await pxeServiceApi.updateConfig(editForm)
      await loadConfigs()
      showConfigModal = false
      alert('配置保存成功')
    } catch (err) {
      alert('保存失败: ' + (err instanceof Error ? err.message : '未知错误'))
    } finally {
      loadingAction = false
    }
  }
  
  async function openLogsModal(config: PXEServiceConfig) {
    selectedConfig = config
    showLogsModal = true
    containerLogs = '正在加载日志...'
    try {
      const response = await pxeServiceApi.getLogs()
      containerLogs = typeof response === 'string' ? response : (response.logs || '无日志输出')
    } catch (err) {
      containerLogs = '获取日志失败: ' + (err instanceof Error ? err.message : '未知错误')
    }
  }
  
  function closeModals() {
    showConfigModal = false
    showLogsModal = false
    selectedConfig = null
  }
  
  function getStatusColor(status: string | null): string {
    if (!status) return '#8e8e93'
    if (status.includes('running')) return '#34c759'
    if (status.includes('exited')) return '#ff9500'
    if (status.includes('created')) return '#007aff'
    return '#8e8e93'
  }
  
  function getStatusText(status: string | null): string {
    if (!status) return '未部署'
    if (status.includes('running')) return '运行中'
    if (status.includes('exited')) return '已停止'
    if (status.includes('created')) return '已创建'
    return status
  }
</script>

<div class="pxe-page">
  <div class="page-container">
    <header class="page-header">
      <div>
        <h1>PXE 服务管理</h1>
        <p class="subtitle">管理PXE网络启动服务和部署配置</p>
      </div>
    </header>
    
    {#if loading}
      <div class="loading">加载中...</div>
    {:else if error}
      <div class="error">{error}</div>
    {:else if configs.length === 0}
      <div class="empty">暂无PXE服务配置</div>
    {:else}
      <div class="services-list">
        {#each configs as config}
          <div class="service-card">
            <div class="service-header">
              <div class="service-title">
                <h3>PXE Service</h3>
                <span class="container-name">{config.container_name}</span>
              </div>
              <span 
                class="status-badge" 
                style="background-color: {getStatusColor(config.container_status)}20; color: {getStatusColor(config.container_status)}"
              >
                {getStatusText(config.container_status)}
              </span>
            </div>
            
            <div class="service-info">
              <div class="info-grid">
                <div class="info-item">
                  <span class="label">服务器IP</span>
                  <span class="value">{config.server_ip}</span>
                </div>
                <div class="info-item">
                  <span class="label">DHCP范围</span>
                  <span class="value">{config.dhcp_range_start} - {config.dhcp_range_end}</span>
                </div>
                <div class="info-item">
                  <span class="label">启动模式</span>
                  <span class="value">
                    {#if config.enable_bios && config.enable_uefi}
                      BIOS + UEFI
                    {:else if config.enable_bios}
                      BIOS
                    {:else if config.enable_uefi}
                      UEFI
                    {:else}
                      未启用
                    {/if}
                  </span>
                </div>
                <div class="info-item">
                  <span class="label">安装类型</span>
                  <span class="value">
                    {#if config.enable_encrypted && config.enable_unencrypted}
                      加密 + 非加密
                    {:else if config.enable_encrypted}
                      加密
                    {:else if config.enable_unencrypted}
                      非加密
                    {:else}
                      未配置
                    {/if}
                  </span>
                </div>
                <div class="info-item">
                  <span class="label">Debian版本</span>
                  <span class="value">{config.debian_version}</span>
                </div>
                <div class="info-item">
                  <span class="label">最后更新</span>
                  <span class="value">{new Date(config.updated_at).toLocaleString('zh-CN')}</span>
                </div>
              </div>
            </div>
            
            {#if isAdmin}
              <div class="service-actions">
                <button class="btn-action" on:click={() => openConfigModal(config)} disabled={loadingAction}>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                  </svg>
                  查看配置
                </button>
                
                {#if config.container_status?.includes('running')}
                  <button class="btn-warning" on:click={() => handleStop(config)} disabled={loadingAction}>
                    <svg viewBox="0 0 24 24" fill="currentColor">
                      <rect x="6" y="6" width="12" height="12" rx="2"></rect>
                    </svg>
                    停止
                  </button>
                  <button class="btn-action" on:click={() => handleRestart(config)} disabled={loadingAction}>
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="23 4 23 10 17 10"></polyline>
                      <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
                    </svg>
                    重启
                  </button>
                {:else}
                  <button class="btn-success" on:click={() => handleStart(config)} disabled={loadingAction}>
                    <svg viewBox="0 0 24 24" fill="currentColor">
                      <polygon points="5 3 19 12 5 21 5 3"></polygon>
                    </svg>
                    启动
                  </button>
                {/if}
                
                <button class="btn-action" on:click={() => openLogsModal(config)} disabled={loadingAction}>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                    <polyline points="14 2 14 8 20 8"></polyline>
                    <line x1="16" y1="13" x2="8" y2="13"></line>
                    <line x1="16" y1="17" x2="8" y2="17"></line>
                    <polyline points="10 9 9 9 8 9"></polyline>
                  </svg>
                  查看日志
                </button>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<!-- Config Modal -->
{#if showConfigModal && selectedConfig}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="modal-overlay" on:click={closeModals}>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="modal large" on:click|stopPropagation>
      <h2>PXE 服务配置</h2>
      <form on:submit|preventDefault={handleSaveConfig}>
        <div class="form-section">
          <h3>网络配置</h3>
          <div class="form-row">
            <div class="form-group">
              <label for="server_ip">服务器IP *</label>
              <input id="server_ip" type="text" bind:value={editForm.server_ip} required />
            </div>
          </div>
          <div class="form-row">
            <div class="form-group">
              <label for="dhcp_start">DHCP起始IP *</label>
              <input id="dhcp_start" type="text" bind:value={editForm.dhcp_range_start} required />
            </div>
            <div class="form-group">
              <label for="dhcp_end">DHCP结束IP *</label>
              <input id="dhcp_end" type="text" bind:value={editForm.dhcp_range_end} required />
            </div>
          </div>
          <div class="form-row">
            <div class="form-group">
              <label for="netmask">子网掩码 *</label>
              <input id="netmask" type="text" bind:value={editForm.netmask} required />
            </div>
            <div class="form-group">
              <label for="gateway">网关</label>
              <input id="gateway" type="text" bind:value={editForm.gateway} />
            </div>
          </div>
          <div class="form-group">
            <label for="dns_servers">DNS服务器 (逗号分隔)</label>
            <input id="dns_servers" type="text" bind:value={editForm.dns_servers} placeholder="8.8.8.8,8.8.4.4" />
          </div>
        </div>
        
        <div class="form-section">
          <h3>启动选项</h3>
          <div class="checkbox-group">
            <label class="checkbox-label">
              <input type="checkbox" bind:checked={editForm.enable_bios} />
              <span>启用 BIOS 启动</span>
            </label>
            <label class="checkbox-label">
              <input type="checkbox" bind:checked={editForm.enable_uefi} />
              <span>启用 UEFI 启动</span>
            </label>
          </div>
          {#if editForm.enable_bios}
            <div class="form-group">
              <label for="bios_boot">BIOS启动文件</label>
              <input id="bios_boot" type="text" bind:value={editForm.bios_boot_file} />
            </div>
          {/if}
          {#if editForm.enable_uefi}
            <div class="form-group">
              <label for="uefi_boot">UEFI启动文件</label>
              <input id="uefi_boot" type="text" bind:value={editForm.uefi_boot_file} />
            </div>
          {/if}
        </div>
        
        <div class="form-section">
          <h3>安装选项</h3>
          <div class="checkbox-group">
            <label class="checkbox-label">
              <input type="checkbox" bind:checked={editForm.enable_encrypted} />
              <span>启用加密安装 (LUKS)</span>
            </label>
            <label class="checkbox-label">
              <input type="checkbox" bind:checked={editForm.enable_unencrypted} />
              <span>启用非加密安装</span>
            </label>
            <label class="checkbox-label">
              <input type="checkbox" bind:checked={editForm.default_encryption} />
              <span>默认使用加密</span>
            </label>
          </div>
        </div>
        
        <div class="form-section">
          <h3>系统配置</h3>
          <div class="form-row">
            <div class="form-group">
              <label for="debian_version">Debian版本</label>
              <input id="debian_version" type="text" bind:value={editForm.debian_version} />
            </div>
            <div class="form-group">
              <label for="default_username">默认用户名</label>
              <input id="default_username" type="text" bind:value={editForm.default_username} />
            </div>
          </div>
          <div class="form-group">
            <label for="debian_mirror">Debian镜像源</label>
            <input id="debian_mirror" type="text" bind:value={editForm.debian_mirror} />
          </div>
          <div class="form-group">
            <label for="tftp_root">TFTP根目录</label>
            <input id="tftp_root" type="text" bind:value={editForm.tftp_root} />
          </div>
        </div>
        
        <div class="modal-actions">
          <button type="button" class="btn-secondary" on:click={closeModals}>取消</button>
          <button type="submit" class="btn-primary" disabled={loadingAction}>
            {loadingAction ? '保存中...' : '保存配置'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- Logs Modal -->
{#if showLogsModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="modal-overlay" on:click={closeModals}>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="modal large" on:click|stopPropagation>
      <h2>容器日志</h2>
      <div class="logs-container">
        <pre>{containerLogs}</pre>
      </div>
      <div class="modal-actions">
        <button class="btn-primary" on:click={closeModals}>关闭</button>
      </div>
    </div>
  </div>
{/if}

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
    margin-bottom: var(--spacing-xl);
    
    h1 {
      font: var(--title-1);
      color: var(--systemPrimary);
      margin: 0 0 var(--spacing-xs);
    }
    
    .subtitle {
      font: var(--body);
      color: var(--systemSecondary);
      margin: 0;
    }
  }
  
  .services-list {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-lg);
  }
  
  .service-card {
    background: var(--defaultBackground);
    border-radius: var(--radius-md);
    padding: var(--spacing-xl);
    box-shadow: var(--shadow-card);
  }
  
  .service-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: var(--spacing-lg);
    padding-bottom: var(--spacing-md);
    border-bottom: 1px solid rgba(0, 0, 0, 0.08);
    
    .service-title {
      h3 {
        font: var(--title-2);
        margin: 0 0 4px;
        color: var(--systemPrimary);
      }
      
      .container-name {
        font: var(--caption-1);
        color: var(--systemTertiary);
        font-family: monospace;
      }
    }
  }
  
  .status-badge {
    padding: 6px 16px;
    border-radius: 16px;
    font: var(--callout);
    font-weight: 600;
  }
  
  .service-info {
    margin-bottom: var(--spacing-lg);
  }
  
  .info-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: var(--spacing-md);
  }
  
  .info-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
    
    .label {
      font: var(--caption-1);
      color: var(--systemSecondary);
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }
    
    .value {
      font: var(--callout);
      color: var(--systemPrimary);
      font-weight: 500;
    }
  }
  
  .service-actions {
    display: flex;
    gap: var(--spacing-sm);
    flex-wrap: wrap;
    padding-top: var(--spacing-md);
    border-top: 1px solid rgba(0, 0, 0, 0.05);
  }
  
  .btn-action, .btn-success, .btn-warning, .btn-primary, .btn-secondary {
    padding: var(--spacing-sm) var(--spacing-lg);
    border: none;
    border-radius: var(--radius-sm);
    font: var(--callout);
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition-fast);
    display: flex;
    align-items: center;
    gap: var(--spacing-xs);
    
    svg {
      width: 16px;
      height: 16px;
    }
    
    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }
  
  .btn-action {
    background: rgba(0, 122, 255, 0.1);
    color: #007aff;
    
    &:hover:not(:disabled) {
      background: rgba(0, 122, 255, 0.2);
    }
  }
  
  .btn-success {
    background: rgba(52, 199, 89, 0.1);
    color: #34c759;
    
    &:hover:not(:disabled) {
      background: rgba(52, 199, 89, 0.2);
    }
  }
  
  .btn-warning {
    background: rgba(255, 149, 0, 0.1);
    color: #ff9500;
    
    &:hover:not(:disabled) {
      background: rgba(255, 149, 0, 0.2);
    }
  }
  
  .btn-primary {
    background: var(--keyColor);
    color: white;
    
    &:hover:not(:disabled) {
      opacity: 0.85;
    }
  }
  
  .btn-secondary {
    background: transparent;
    color: var(--systemSecondary);
    border: 1px solid rgba(0, 0, 0, 0.1);
    
    &:hover:not(:disabled) {
      background: rgba(0, 0, 0, 0.03);
    }
  }
  
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    backdrop-filter: blur(4px);
  }
  
  .modal {
    background: var(--defaultBackground);
    border-radius: var(--radius-lg);
    padding: var(--spacing-xl);
    width: 90%;
    max-width: 600px;
    max-height: 90vh;
    overflow-y: auto;
    
    &.large {
      max-width: 800px;
    }
    
    h2 {
      font: var(--title-2);
      margin: 0 0 var(--spacing-lg);
      color: var(--systemPrimary);
    }
    
    h3 {
      font: var(--title-3);
      margin: 0 0 var(--spacing-md);
      color: var(--systemPrimary);
    }
  }
  
  .form-section {
    margin-bottom: var(--spacing-xl);
    
    &:last-of-type {
      margin-bottom: var(--spacing-lg);
    }
  }
  
  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--spacing-md);
  }
  
  .form-group {
    margin-bottom: var(--spacing-md);
    
    label {
      display: block;
      font: var(--callout);
      font-weight: 600;
      margin-bottom: var(--spacing-xs);
      color: var(--systemPrimary);
    }
    
    input {
      width: 100%;
      padding: var(--spacing-sm) var(--spacing-md);
      border: 1px solid rgba(0, 0, 0, 0.1);
      border-radius: var(--radius-sm);
      font: var(--body);
      color: var(--systemPrimary);
      background: var(--defaultBackground);
      
      &:focus {
        outline: none;
        border-color: var(--keyColor);
      }
    }
  }
  
  .checkbox-group {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-sm);
    margin-bottom: var(--spacing-md);
  }
  
  .checkbox-label {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
    cursor: pointer;
    font: var(--callout);
    color: var(--systemPrimary);
    
    input[type="checkbox"] {
      width: 18px;
      height: 18px;
      cursor: pointer;
    }
  }
  
  .modal-actions {
    display: flex;
    gap: var(--spacing-sm);
    justify-content: flex-end;
    margin-top: var(--spacing-lg);
    padding-top: var(--spacing-lg);
    border-top: 1px solid rgba(0, 0, 0, 0.08);
  }
  
  .logs-container {
    background: rgba(0, 0, 0, 0.03);
    border-radius: var(--radius-sm);
    padding: var(--spacing-md);
    max-height: 500px;
    overflow-y: auto;
    margin-bottom: var(--spacing-lg);
    
    pre {
      margin: 0;
      font-family: 'SF Mono', Monaco, monospace;
      font-size: 12px;
      line-height: 1.5;
      color: var(--systemPrimary);
      white-space: pre-wrap;
      word-break: break-all;
    }
    
    @media (prefers-color-scheme: dark) {
      background: rgba(255, 255, 255, 0.05);
    }
  }
  
  .loading, .error, .empty {
    text-align: center;
    padding: var(--spacing-3xl);
    color: var(--systemSecondary);
  }
  
  .error {
    color: #ff3b30;
  }
</style>