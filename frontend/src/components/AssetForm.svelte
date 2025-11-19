<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import { assetApi, type AssetCreate } from '../lib/api'
  import Button from './Button.svelte'
  
  export let show = false
  export let asset: any = null // 如果编辑现有资产
  
  const dispatch = createEventDispatcher()
  
  let formData: AssetCreate = {
    hostname: asset?.hostname || '',
    ip_address: asset?.ip_address || '',
    mac_address: asset?.mac_address || '',
    asset_tag: asset?.asset_tag || '',
    os_type: asset?.os_type || 'ubuntu',
    status: asset?.status || 'pending',
    encryption_enabled: asset?.encryption_enabled || false,
    usb_key: asset?.usb_key || ''
  }
  
  let loading = false
  let error: string | null = null
  
  const osTypes = [
    { value: 'ubuntu', label: 'Ubuntu' },
    { value: 'debian', label: 'Debian' },
    { value: 'centos', label: 'CentOS' },
    { value: 'rhel', label: 'RHEL' },
    { value: 'rocky', label: 'Rocky Linux' },
    { value: 'alma', label: 'AlmaLinux' },
    { value: 'windows', label: 'Windows' },
    { value: 'macos', label: 'macOS' }
  ]
  
  const statusOptions = [
    { value: 'pending', label: '待部署' },
    { value: 'active', label: '运行中' },
    { value: 'maintenance', label: '维护中' },
    { value: 'offline', label: '离线' }
  ]
  
  async function handleSubmit() {
    error = null
    loading = true
    
    try {
      if (asset?.id) {
        await assetApi.update(asset.id, formData)
      } else {
        await assetApi.create(formData)
      }
      dispatch('success')
      handleClose()
    } catch (err) {
      error = err instanceof Error ? err.message : '保存失败'
      console.error('Failed to save asset:', err)
    } finally {
      loading = false
    }
  }
  
  function handleClose() {
    show = false
    dispatch('close')
    resetForm()
  }
  
  function resetForm() {
    formData = {
      hostname: '',
      ip_address: '',
      mac_address: '',
      asset_tag: '',
      os_type: 'ubuntu',
      status: 'pending',
      encryption_enabled: false,
      usb_key: ''
    }
    error = null
  }
  
  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      handleClose()
    }
  }

  function handleBackdropKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      handleClose()
    }
  }
</script>

{#if show}
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div 
    class="modal-backdrop" 
    on:click={handleBackdropClick}
    on:keydown={handleBackdropKeydown}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div class="modal-content">
      <div class="modal-header">
        <h2>{asset?.id ? '编辑资产' : '添加资产'}</h2>
        <button class="close-button" on:click={handleClose} type="button">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>
      
      <form class="modal-body" on:submit|preventDefault={handleSubmit}>
        {#if error}
          <div class="error-banner">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="12" y1="8" x2="12" y2="12"></line>
              <line x1="12" y1="16" x2="12.01" y2="16"></line>
            </svg>
            <span>{error}</span>
          </div>
        {/if}
        
        <div class="form-grid">
          <div class="form-group">
            <label for="hostname">主机名 *</label>
            <input
              id="hostname"
              type="text"
              bind:value={formData.hostname}
              placeholder="infra-node1"
              required
            />
          </div>
          
          <div class="form-group">
            <label for="ip_address">IP 地址 *</label>
            <input
              id="ip_address"
              type="text"
              bind:value={formData.ip_address}
              placeholder="10.10.10.12"
              required
            />
          </div>
          
          <div class="form-group">
            <label for="mac_address">MAC 地址 *</label>
            <input
              id="mac_address"
              type="text"
              bind:value={formData.mac_address}
              placeholder="00:1A:2B:3C:4D:5E"
              pattern="^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$"
              required
            />
          </div>
          
          <div class="form-group">
            <label for="asset_tag">资产标签</label>
            <input
              id="asset_tag"
              type="text"
              bind:value={formData.asset_tag}
              placeholder="AST-001"
            />
          </div>
          
          <div class="form-group">
            <label for="os_type">操作系统 *</label>
            <select id="os_type" bind:value={formData.os_type} required>
              {#each osTypes as osType}
                <option value={osType.value}>{osType.label}</option>
              {/each}
            </select>
          </div>
          
          <div class="form-group">
            <label for="status">状态</label>
            <select id="status" bind:value={formData.status}>
              {#each statusOptions as status}
                <option value={status.value}>{status.label}</option>
              {/each}
            </select>
          </div>
          
          <div class="form-group checkbox-group">
            <label class="checkbox-label">
              <input
                type="checkbox"
                bind:checked={formData.encryption_enabled}
              />
              <span>启用加密</span>
            </label>
          </div>
          
          {#if formData.encryption_enabled}
            <div class="form-group">
              <label for="usb_key">USB 密钥 ID</label>
              <input
                id="usb_key"
                type="text"
                bind:value={formData.usb_key}
                placeholder="USB-KEY-001"
              />
            </div>
          {/if}
        </div>
        
        <div class="modal-footer">
          <Button type="button" variant="secondary" on:click={handleClose} disabled={loading}>
            取消
          </Button>
          <Button type="submit" variant="primary" disabled={loading}>
            {loading ? '保存中...' : (asset?.id ? '更新' : '添加')}
          </Button>
        </div>
      </form>
    </div>
  </div>
{/if}

<style lang="scss">
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: var(--spacing-lg);
  }
  
  .modal-content {
    background: var(--defaultBackground);
    border-radius: var(--radius-lg);
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
    max-width: 600px;
    width: 100%;
    max-height: 90vh;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }
  
  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--spacing-xl);
    border-bottom: 1px solid rgba(0, 0, 0, 0.1);
    
    @media (prefers-color-scheme: dark) {
      border-bottom-color: rgba(255, 255, 255, 0.1);
    }
    
    h2 {
      font: var(--title-2);
      color: var(--systemPrimary);
      margin: 0;
    }
  }
  
  .close-button {
    background: none;
    border: none;
    padding: var(--spacing-sm);
    cursor: pointer;
    color: var(--systemSecondary);
    transition: color var(--transition-fast);
    
    svg {
      width: 24px;
      height: 24px;
    }
    
    &:hover {
      color: var(--systemPrimary);
    }
  }
  
  .modal-body {
    padding: var(--spacing-xl);
    overflow-y: auto;
  }
  
  .error-banner {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
    padding: var(--spacing-md);
    background: rgba(255, 59, 48, 0.1);
    border: 1px solid rgba(255, 59, 48, 0.3);
    border-radius: var(--radius-sm);
    color: #ff3b30;
    margin-bottom: var(--spacing-lg);
    
    svg {
      width: 20px;
      height: 20px;
      flex-shrink: 0;
    }
    
    span {
      font: var(--callout);
    }
  }
  
  .form-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--spacing-lg);
    
    @media (max-width: 600px) {
      grid-template-columns: 1fr;
    }
  }
  
  .form-group {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-xs);
    
    &.checkbox-group {
      grid-column: 1 / -1;
    }
    
    label {
      font: var(--callout);
      color: var(--systemPrimary);
      font-weight: 500;
    }
    
    input[type="text"],
    select {
      padding: var(--spacing-md);
      border: 1px solid rgba(0, 0, 0, 0.2);
      border-radius: var(--radius-sm);
      font: var(--body);
      color: var(--systemPrimary);
      background: var(--defaultBackground);
      transition: all var(--transition-fast);
      
      &:focus {
        outline: none;
        border-color: var(--keyColor);
        box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
      }
      
      @media (prefers-color-scheme: dark) {
        border-color: rgba(255, 255, 255, 0.2);
      }
    }
  }
  
  .checkbox-label {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
    cursor: pointer;
    
    input[type="checkbox"] {
      width: 20px;
      height: 20px;
      cursor: pointer;
    }
    
    span {
      font: var(--body);
      color: var(--systemPrimary);
    }
  }
  
  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--spacing-md);
    padding: var(--spacing-xl);
    padding-top: 0;
  }
</style>
