<script lang="ts">
  import { onMount } from 'svelte'
  import { usbKeyApi, assetApi, type USBKey, type Asset } from '../lib/api'
  
  let keys: USBKey[] = []
  let assets: Asset[] = []
  let loading = true
  let error: string | null = null
  let isAdmin = false
  
  // Modals
  let showInitModal = false
  let showBindModal = false
  let showBackupModal = false
  let showRestoreModal = false
  let showRebuildModal = false
  
  // Selected key for operations
  let selectedKey: USBKey | null = null
  
  // Form data
  let initLabel = ''
  let initDescription = ''
  let bindAssetId: number | null = null
  let restoreBackupUuid = ''
  let restoreBackupPassword = ''
  let rebuildRecoveryCode = ''
  
  // Response data (for one-time display)
  let initResponse: any = null
  let backupResponse: any = null
  let rebuildResponse: any = null
  
  onMount(async () => {
    await checkAdmin()
    await Promise.all([loadKeys(), loadAssets()])
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
  
  async function loadKeys() {
    try {
      loading = true
      error = null
      const response = await usbKeyApi.list()
      keys = response.items
    } catch (err) {
      error = err instanceof Error ? err.message : '加载失败'
      console.error('Failed to load keys:', err)
    } finally {
      loading = false
    }
  }
  
  async function loadAssets() {
    try {
      const response = await assetApi.list({ limit: 1000 })
      assets = response.items
    } catch (err) {
      console.error('Failed to load assets:', err)
    }
  }
  
  async function handleInit() {
    try {
      const result = await usbKeyApi.initialize({ label: initLabel, description: initDescription })
      initResponse = result
      await loadKeys()
      initLabel = ''
      initDescription = ''
    } catch (err) {
      alert('初始化失败: ' + (err instanceof Error ? err.message : '未知错误'))
    }
  }
  
  async function handleBind() {
    if (!selectedKey || !bindAssetId) return
    try {
      await usbKeyApi.bind(selectedKey.id, bindAssetId)
      await loadKeys()
      closeModals()
      alert('绑定成功')
    } catch (err) {
      alert('绑定失败: ' + (err instanceof Error ? err.message : '未知错误'))
    }
  }
  
  async function handleUnbind(key: USBKey) {
    if (!confirm('确定要解绑吗？')) return
    try {
      await usbKeyApi.unbind(key.id)
      await loadKeys()
      alert('解绑成功')
    } catch (err) {
      alert('解绑失败: ' + (err instanceof Error ? err.message : '未知错误'))
    }
  }
  
  async function handleCreateBackup(key: USBKey) {
    try {
      const result = await usbKeyApi.createBackup(key.id)
      backupResponse = result
      await loadKeys()
      showBackupModal = true
    } catch (err) {
      alert('备份失败: ' + (err instanceof Error ? err.message : '未知错误'))
    }
  }
  
  async function handleDownloadBackup(key: USBKey) {
    try {
      const blob = await usbKeyApi.downloadBackup(key.id)
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `usbkey_${key.uuid}_backup.enc`
      a.click()
      window.URL.revokeObjectURL(url)
    } catch (err) {
      alert('下载失败: ' + (err instanceof Error ? err.message : '未知错误'))
    }
  }
  
  async function handleRestore() {
    try {
      await usbKeyApi.restore(restoreBackupUuid, restoreBackupPassword)
      await loadKeys()
      closeModals()
      alert('恢复成功')
    } catch (err) {
      alert('恢复失败: ' + (err instanceof Error ? err.message : '未知错误'))
    }
  }
  
  async function handleRebuild() {
    if (!selectedKey) return
    try {
      const result = await usbKeyApi.rebuild(selectedKey.id, rebuildRecoveryCode)
      rebuildResponse = result
      await loadKeys()
      rebuildRecoveryCode = ''
    } catch (err) {
      alert('重建失败: ' + (err instanceof Error ? err.message : '未知错误'))
    }
  }
  
  async function handleRevoke(key: USBKey) {
    const reason = prompt('请输入吊销原因（可选）:')
    if (reason === null) return
    try {
      await usbKeyApi.revoke(key.id, reason || undefined)
      await loadKeys()
      alert('吊销成功')
    } catch (err) {
      alert('吊销失败: ' + (err instanceof Error ? err.message : '未知错误'))
    }
  }
  
  function openBindModal(key: USBKey) {
    selectedKey = key
    bindAssetId = null
    showBindModal = true
  }
  
  function openRebuildModal(key: USBKey) {
    selectedKey = key
    rebuildRecoveryCode = ''
    rebuildResponse = null
    showRebuildModal = true
  }
  
  function closeModals() {
    showInitModal = false
    showBindModal = false
    showBackupModal = false
    showRestoreModal = false
    showRebuildModal = false
    selectedKey = null
    initResponse = null
    backupResponse = null
    rebuildResponse = null
  }
  
  function closeInitResponse() {
    initResponse = null
    showInitModal = false
  }
  
  function getStatusColor(status: string): string {
    const colors: Record<string, string> = {
      pending: '#ff9500',
      active: '#34c759',
      bound: '#007aff',
      revoked: '#ff3b30',
      lost: '#8e8e93'
    }
    return colors[status] || '#8e8e93'
  }
  
  function getBoundAssetName(assetId: number | undefined): string {
    if (!assetId) return '—'
    const asset = assets.find(a => a.id === assetId)
    return asset ? asset.hostname : `Asset #${assetId}`
  }
</script>

<div class="usbkeys-page">
  <div class="page-container">
    <header class="page-header">
      <div>
        <h1>USB Key 管理</h1>
        <p class="subtitle">管理加密USB密钥的生命周期</p>
      </div>
      {#if isAdmin}
        <div class="header-actions">
          <button class="btn-primary" on:click={() => showInitModal = true}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
            初始化USB Key
          </button>
          <button class="btn-secondary" on:click={() => showRestoreModal = true}>
            恢复备份
          </button>
        </div>
      {/if}
    </header>
    
    {#if loading}
      <div class="loading">加载中...</div>
    {:else if error}
      <div class="error">{error}</div>
    {:else if keys.length === 0}
      <div class="empty">暂无USB Key</div>
    {:else}
      <div class="keys-grid">
        {#each keys as key}
          <div class="key-card">
            <div class="key-header">
              <div>
                <h3>{key.label}</h3>
                <span class="key-uuid">{key.uuid}</span>
              </div>
              <span class="status-badge" style="background-color: {getStatusColor(key.status)}20; color: {getStatusColor(key.status)}">
                {key.status}
              </span>
            </div>
            
            <div class="key-info">
              <div class="info-row">
                <span class="label">绑定机器:</span>
                <span class="value">{getBoundAssetName(key.asset_id)}</span>
              </div>
              <div class="info-row">
                <span class="label">使用次数:</span>
                <span class="value">{key.use_count}</span>
              </div>
              <div class="info-row">
                <span class="label">备份次数:</span>
                <span class="value">{key.backup_count}</span>
              </div>
              {#if key.last_used_at}
                <div class="info-row">
                  <span class="label">最后使用:</span>
                  <span class="value">{new Date(key.last_used_at).toLocaleString('zh-CN')}</span>
                </div>
              {/if}
            </div>
            
            {#if isAdmin}
              <div class="key-actions">
                {#if !key.asset_id && key.status === 'active'}
                  <button class="btn-action" on:click={() => openBindModal(key)}>绑定</button>
                {/if}
                {#if key.asset_id}
                  <button class="btn-action" on:click={() => handleUnbind(key)}>解绑</button>
                {/if}
                <button class="btn-action" on:click={() => handleCreateBackup(key)}>备份</button>
                {#if key.backup_count > 0}
                  <button class="btn-action" on:click={() => handleDownloadBackup(key)}>下载</button>
                {/if}
                <button class="btn-action" on:click={() => openRebuildModal(key)}>重建</button>
                {#if key.status !== 'revoked'}
                  <button class="btn-danger" on:click={() => handleRevoke(key)}>吊销</button>
                {/if}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<!-- Initialize Modal -->
{#if showInitModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="modal-overlay" on:click={closeModals}>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="modal" on:click|stopPropagation>
      <h2>初始化USB Key</h2>
      {#if !initResponse}
        <form on:submit|preventDefault={handleInit}>
          <div class="form-group">
            <label for="initLabel">标签 *</label>
            <input id="initLabel" type="text" bind:value={initLabel} required />
          </div>
          <div class="form-group">
            <label for="initDescription">描述</label>
            <textarea id="initDescription" bind:value={initDescription} rows="3"></textarea>
          </div>
          <div class="modal-actions">
            <button type="button" class="btn-secondary" on:click={closeModals}>取消</button>
            <button type="submit" class="btn-primary">初始化</button>
          </div>
        </form>
      {:else}
        <div class="success-box">
          <p class="warning">⚠️ 请妥善保存以下信息，关闭后将无法再次查看！</p>
          <div class="code-block">
            <strong>UUID:</strong>
            <code>{initResponse.uuid}</code>
          </div>
          <div class="code-block">
            <strong>密钥材料 (写入USB Key):</strong>
            <code>{initResponse.key_material}</code>
          </div>
          <div class="code-block">
            <strong>恢复代码 (安全保存):</strong>
            <code>{initResponse.recovery_code}</code>
          </div>
          <button class="btn-primary" on:click={closeInitResponse}>我已保存</button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- Bind Modal -->
{#if showBindModal && selectedKey}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="modal-overlay" on:click={closeModals}>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="modal" on:click|stopPropagation>
      <h2>绑定到资产</h2>
      <form on:submit|preventDefault={handleBind}>
        <div class="form-group">
          <label for="bindAssetId">选择资产 *</label>
          <select id="bindAssetId" bind:value={bindAssetId} required>
            <option value={null}>-- 选择资产 --</option>
            {#each assets as asset}
              <option value={asset.id}>{asset.hostname} ({asset.ip_address})</option>
            {/each}
          </select>
        </div>
        <div class="modal-actions">
          <button type="button" class="btn-secondary" on:click={closeModals}>取消</button>
          <button type="submit" class="btn-primary">绑定</button>
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- Backup Response Modal -->
{#if showBackupModal && backupResponse}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="modal-overlay" on:click={closeModals}>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="modal" on:click|stopPropagation>
      <h2>备份成功</h2>
      <div class="success-box">
        <p class="warning">⚠️ 请妥善保存备份密码！</p>
        <div class="code-block">
          <strong>备份UUID:</strong>
          <code>{backupResponse.backup_uuid}</code>
        </div>
        <div class="code-block">
          <strong>备份密码:</strong>
          <code>{backupResponse.backup_password}</code>
        </div>
        <div class="code-block">
          <strong>校验和:</strong>
          <code>{backupResponse.checksum}</code>
        </div>
        <button class="btn-primary" on:click={closeModals}>关闭</button>
      </div>
    </div>
  </div>
{/if}

<!-- Restore Modal -->
{#if showRestoreModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="modal-overlay" on:click={closeModals}>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="modal" on:click|stopPropagation>
      <h2>从备份恢复</h2>
      <form on:submit|preventDefault={handleRestore}>
        <div class="form-group">
          <label for="restoreBackupUuid">备份UUID *</label>
          <input id="restoreBackupUuid" type="text" bind:value={restoreBackupUuid} required />
        </div>
        <div class="form-group">
          <label for="restoreBackupPassword">备份密码 *</label>
          <input id="restoreBackupPassword" type="password" bind:value={restoreBackupPassword} required />
        </div>
        <div class="modal-actions">
          <button type="button" class="btn-secondary" on:click={closeModals}>取消</button>
          <button type="submit" class="btn-primary">恢复</button>
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- Rebuild Modal -->
{#if showRebuildModal && selectedKey}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="modal-overlay" on:click={closeModals}>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="modal" on:click|stopPropagation>
      <h2>重建USB Key</h2>
      {#if !rebuildResponse}
        <form on:submit|preventDefault={handleRebuild}>
          <div class="form-group">
            <label for="rebuildRecoveryCode">恢复代码 *</label>
            <input id="rebuildRecoveryCode" type="password" bind:value={rebuildRecoveryCode} required />
          </div>
          <div class="modal-actions">
            <button type="button" class="btn-secondary" on:click={closeModals}>取消</button>
            <button type="submit" class="btn-primary">重建</button>
          </div>
        </form>
      {:else}
        <div class="success-box">
          <p class="warning">⚠️ 请将新密钥材料写入新USB Key！</p>
          <div class="code-block">
            <strong>UUID:</strong>
            <code>{rebuildResponse.uuid}</code>
          </div>
          <div class="code-block">
            <strong>新密钥材料:</strong>
            <code>{rebuildResponse.key_material}</code>
          </div>
          <button class="btn-primary" on:click={closeModals}>关闭</button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style lang="scss">
  .usbkeys-page {
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
    align-items: center;
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
  
  .header-actions {
    display: flex;
    gap: var(--spacing-sm);
  }
  
  .btn-primary, .btn-secondary, .btn-action, .btn-danger {
    padding: var(--spacing-sm) var(--spacing-lg);
    border: none;
    border-radius: var(--radius-sm);
    font: var(--body);
    cursor: pointer;
    transition: all var(--transition-fast);
    display: flex;
    align-items: center;
    gap: var(--spacing-xs);
    
    svg {
      width: 18px;
      height: 18px;
    }
  }
  
  .btn-primary {
    background: var(--keyColor);
    color: white;
    
    &:hover {
      opacity: 0.85;
    }
  }
  
  .btn-secondary {
    background: transparent;
    color: var(--systemSecondary);
    border: 1px solid rgba(0, 0, 0, 0.1);
    
    &:hover {
      background: rgba(0, 0, 0, 0.03);
    }
  }
  
  .btn-action {
    background: rgba(0, 122, 255, 0.1);
    color: #007aff;
    font-size: 14px;
    padding: 6px 12px;
    
    &:hover {
      background: rgba(0, 122, 255, 0.2);
    }
  }
  
  .btn-danger {
    background: rgba(255, 59, 48, 0.1);
    color: #ff3b30;
    font-size: 14px;
    padding: 6px 12px;
    
    &:hover {
      background: rgba(255, 59, 48, 0.2);
    }
  }
  
  .keys-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
    gap: var(--spacing-lg);
  }
  
  .key-card {
    background: var(--defaultBackground);
    border-radius: var(--radius-md);
    padding: var(--spacing-lg);
    box-shadow: var(--shadow-card);
  }
  
  .key-header {
    display: flex;
    justify-content: space-between;
    align-items: start;
    margin-bottom: var(--spacing-md);
    
    h3 {
      font: var(--title-3);
      margin: 0 0 4px;
      color: var(--systemPrimary);
    }
    
    .key-uuid {
      font: var(--caption-1);
      color: var(--systemTertiary);
      font-family: monospace;
    }
  }
  
  .status-badge {
    padding: 4px 12px;
    border-radius: 12px;
    font: var(--caption-1);
    font-weight: 600;
  }
  
  .key-info {
    margin-bottom: var(--spacing-md);
    
    .info-row {
      display: flex;
      justify-content: space-between;
      padding: var(--spacing-xs) 0;
      border-bottom: 1px solid rgba(0, 0, 0, 0.05);
      
      &:last-child {
        border-bottom: none;
      }
      
      .label {
        font: var(--callout);
        color: var(--systemSecondary);
      }
      
      .value {
        font: var(--callout);
        color: var(--systemPrimary);
        font-weight: 500;
      }
    }
  }
  
  .key-actions {
    display: flex;
    gap: var(--spacing-xs);
    flex-wrap: wrap;
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
    max-width: 600px;
    width: 90%;
    max-height: 90vh;
    overflow-y: auto;
    
    h2 {
      font: var(--title-2);
      margin: 0 0 var(--spacing-lg);
      color: var(--systemPrimary);
    }
  }
  
  .form-group {
    margin-bottom: var(--spacing-lg);
    
    label {
      display: block;
      font: var(--callout);
      font-weight: 600;
      margin-bottom: var(--spacing-xs);
      color: var(--systemPrimary);
    }
    
    input, select, textarea {
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
    
    textarea {
      resize: vertical;
    }
  }
  
  .modal-actions {
    display: flex;
    gap: var(--spacing-sm);
    justify-content: flex-end;
    margin-top: var(--spacing-lg);
  }
  
  .success-box {
    .warning {
      background: rgba(255, 149, 0, 0.1);
      color: #ff9500;
      padding: var(--spacing-md);
      border-radius: var(--radius-sm);
      margin-bottom: var(--spacing-lg);
      font: var(--callout);
    }
  }
  
  .code-block {
    margin-bottom: var(--spacing-md);
    
    strong {
      display: block;
      font: var(--callout);
      margin-bottom: var(--spacing-xs);
      color: var(--systemPrimary);
    }
    
    code {
      display: block;
      background: rgba(0, 0, 0, 0.03);
      padding: var(--spacing-sm) var(--spacing-md);
      border-radius: var(--radius-sm);
      font-family: 'SF Mono', Monaco, monospace;
      font-size: 13px;
      color: var(--systemPrimary);
      word-break: break-all;
      
      @media (prefers-color-scheme: dark) {
        background: rgba(255, 255, 255, 0.05);
      }
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