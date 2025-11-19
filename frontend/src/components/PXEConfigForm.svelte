<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import { pxeApi, type PXEConfig, type PXEConfigCreate } from '../lib/api'
  import Button from './Button.svelte'
  
  export let show = false
  export let config: PXEConfig | null = null
  
  const dispatch = createEventDispatcher()
  
  let formData: PXEConfigCreate = {
    hostname: config?.hostname || '',
    mac_address: config?.mac_address || '',
    ip_address: config?.ip_address || '',
    netmask: config?.netmask || '255.255.255.0',
    gateway: config?.gateway || '',
    dns_servers: config?.dns_servers || '8.8.8.8,8.8.4.4',
    boot_image: config?.boot_image || '/pxelinux.0',
    boot_params: config?.boot_params || '',
    os_type: config?.os_type || 'ubuntu',
    enabled: config?.enabled ?? true,
    description: config?.description || ''
  }
  
  let loading = false
  let error: string | null = null
  
  const osTypes = [
    { value: 'ubuntu', label: 'Ubuntu' },
    { value: 'debian', label: 'Debian' },
    { value: 'centos', label: 'CentOS' },
    { value: 'rhel', label: 'RHEL' },
    { value: 'rocky', label: 'Rocky Linux' },
    { value: 'alma', label: 'AlmaLinux' }
  ]
  
  $: if (config) {
    formData = {
      hostname: config.hostname,
      mac_address: config.mac_address,
      ip_address: config.ip_address,
      netmask: config.netmask,
      gateway: config.gateway || '',
      dns_servers: config.dns_servers || '',
      boot_image: config.boot_image || '',
      boot_params: config.boot_params || '',
      os_type: config.os_type,
      enabled: config.enabled,
      description: config.description || ''
    }
  }
  
  async function handleSubmit() {
    error = null
    loading = true
    
    try {
      if (config?.id) {
        await pxeApi.update(config.id, formData)
      } else {
        await pxeApi.create(formData)
      }
      dispatch('success')
      handleClose()
    } catch (err: any) {
      error = err.response?.data?.detail || err.message || '保存失败'
      console.error('Failed to save PXE config:', err)
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
      mac_address: '',
      ip_address: '',
      netmask: '255.255.255.0',
      gateway: '',
      dns_servers: '8.8.8.8,8.8.4.4',
      boot_image: '/pxelinux.0',
      boot_params: '',
      os_type: 'ubuntu',
      enabled: true,
      description: ''
    }
    error = null
  }
  
  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      handleClose()
    }
  }
</script>

{#if show}
  <div 
    class="modal-backdrop" 
    on:click={handleBackdropClick}
    on:keydown={(e) => e.key === 'Escape' && handleClose()}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div class="modal-content">
      <div class="modal-header">
        <h2>{config?.id ? '编辑PXE配置' : '添加PXE配置'}</h2>
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
          <div class="form-group full-width">
            <label for="hostname">主机名 *</label>
            <input
              id="hostname"
              type="text"
              bind:value={formData.hostname}
              placeholder="infra-node1"
              required
            />
            <p class="field-hint">例如: infra-node1, web-server-01</p>
          </div>
          
          <div class="form-group">
            <label for="mac_address">MAC地址 *</label>
            <input
              id="mac_address"
              type="text"
              bind:value={formData.mac_address}
              placeholder="00:1A:2B:3C:4D:5E"
              required
            />
          </div>
          
          <div class="form-group">
            <label for="ip_address">IP地址 *</label>
            <input
              id="ip_address"
              type="text"
              bind:value={formData.ip_address}
              placeholder="10.10.10.12"
              required
            />
          </div>
          
          <div class="form-group">
            <label for="netmask">子网掩码</label>
            <input
              id="netmask"
              type="text"
              bind:value={formData.netmask}
              placeholder="255.255.255.0"
            />
          </div>
          
          <div class="form-group">
            <label for="gateway">网关</label>
            <input
              id="gateway"
              type="text"
              bind:value={formData.gateway}
              placeholder="10.10.10.1"
            />
          </div>
          
          <div class="form-group full-width">
            <label for="dns_servers">DNS服务器</label>
            <input
              id="dns_servers"
              type="text"
              bind:value={formData.dns_servers}
              placeholder="8.8.8.8,8.8.4.4"
            />
            <p class="field-hint">多个DNS服务器用逗号分隔</p>
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
            <label for="boot_image">启动镜像路径</label>
            <input
              id="boot_image"
              type="text"
              bind:value={formData.boot_image}
              placeholder="/pxelinux.0"
            />
          </div>
          
          <div class="form-group full-width">
            <label for="boot_params">启动参数</label>
            <textarea
              id="boot_params"
              bind:value={formData.boot_params}
              placeholder="APPEND initrd=initrd.img"
              rows="3"
            ></textarea>
            <p class="field-hint">自定义PXE启动参数(可选)</p>
          </div>
          
          <div class="form-group full-width">
            <label for="description">描述</label>
            <textarea
              id="description"
              bind:value={formData.description}
              placeholder="生产环境Web服务器"
              rows="2"
            ></textarea>
          </div>
          
          <div class="form-group checkbox-group">
            <label class="checkbox-label">
              <input
                type="checkbox"
                bind:checked={formData.enabled}
              />
              <span>启用此配置</span>
            </label>
          </div>
        </div>
        
        <div class="modal-footer">
          <Button type="button" variant="secondary" on:click={handleClose} disabled={loading}>
            取消
          </Button>
          <Button type="submit" variant="primary" disabled={loading}>
            {loading ? '保存中...' : (config?.id ? '更新' : '添加')}
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
    max-width: 700px;
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
    
    &.full-width {
      grid-column: 1 / -1;
    }
    
    &.checkbox-group {
      grid-column: 1 / -1;
    }
    
    label {
      font: var(--callout);
      color: var(--systemPrimary);
      font-weight: 500;
    }
    
    input[type="text"],
    select,
    textarea {
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
    
    textarea {
      resize: vertical;
      font-family: inherit;
    }
  }
  
  .field-hint {
    font: var(--caption-1);
    color: var(--systemSecondary);
    margin: 0;
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
