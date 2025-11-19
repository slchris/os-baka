<script lang="ts">
  import type { Asset } from '../lib/api'
  import { createEventDispatcher } from 'svelte'
  
  export let asset: Asset
  
  const dispatch = createEventDispatcher()
  
  function handleDelete() {
    dispatch('delete')
  }
  
  function getStatusColor(status: string): string {
    const colors: Record<string, string> = {
      active: 'var(--successColor)',
      inactive: 'var(--systemTertiary)',
      installing: 'var(--keyColor)',
      maintenance: 'var(--warningColor)',
      error: 'var(--errorColor)',
    }
    return colors[status] || 'var(--systemTertiary)'
  }
  
  function getOSIcon(osType?: string): string {
    // Return SVG path data for different OS types
    const icons: Record<string, string> = {
      macos: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8z',
      linux: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z',
      windows: 'M3 3h8v8H3V3zm10 0h8v8h-8V3zM3 13h8v8H3v-8zm10 0h8v8h-8v-8z',
    }
    return icons[osType || 'macos'] || icons.macos
  }
</script>

<div class="asset-card">
  <div class="card-header">
    <div class="header-left">
      <svg class="os-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="{getOSIcon(asset.os_type)}"></path>
      </svg>
      <h3 class="asset-name">{asset.hostname}</h3>
    </div>
    <span class="status-badge" style="--status-color: {getStatusColor(asset.status)}">
      {asset.status}
    </span>
  </div>
  
  <div class="card-body">
    <div class="info-row">
      <span class="info-label">IP 地址</span>
      <span class="info-value">{asset.ip_address}</span>
    </div>
    <div class="info-row">
      <span class="info-label">MAC 地址</span>
      <span class="info-value">{asset.mac_address}</span>
    </div>
    <div class="info-row">
      <span class="info-label">资产标签</span>
      <span class="info-value">{asset.asset_tag}</span>
    </div>
    {#if asset.encryption_enabled}
      <div class="info-row">
        <span class="info-label">加密</span>
        <span class="info-value encryption-enabled">
          <svg class="lock-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
          </svg>
          已启用
        </span>
      </div>
    {/if}
  </div>
  
  <div class="card-footer">
    <button class="action-btn" on:click={handleDelete}>
      删除
    </button>
    <button class="action-btn primary">
      详情
    </button>
  </div>
</div>

<style lang="scss">
  .asset-card {
    background: var(--defaultBackground);
    border-radius: var(--radius-md);
    padding: var(--spacing-lg);
    box-shadow: var(--shadow-card);
    transition: all var(--transition-standard);
    
    &:hover {
      box-shadow: var(--shadow-card-hover);
      transform: translateY(-4px);
    }
  }
  
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: var(--spacing-md);
    
    .header-left {
      display: flex;
      align-items: center;
      gap: var(--spacing-sm);
      
      .os-icon {
        width: 24px;
        height: 24px;
        color: var(--systemSecondary);
      }
      
      .asset-name {
        font: var(--title-3);
        color: var(--systemPrimary);
        margin: 0;
      }
    }
    
    .status-badge {
      padding: 4px 12px;
      border-radius: var(--radius-full);
      font: var(--footnote-emphasized);
      background: color-mix(in srgb, var(--status-color) 15%, transparent);
      color: var(--status-color);
      text-transform: capitalize;
    }
  }
  
  .card-body {
    margin-bottom: var(--spacing-lg);
    
    .info-row {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: var(--spacing-sm) 0;
      border-bottom: 1px solid rgba(0, 0, 0, 0.05);
      
      &:last-child {
        border-bottom: none;
      }
      
      .info-label {
        font: var(--footnote);
        color: var(--systemSecondary);
      }
      
      .info-value {
        font: var(--footnote-emphasized);
        color: var(--systemPrimary);
        font-family: 'SF Mono', 'Monaco', 'Courier New', monospace;
        
        &.encryption-enabled {
          display: flex;
          align-items: center;
          gap: var(--spacing-xs);
          color: var(--successColor);
          
          .lock-icon {
            width: 14px;
            height: 14px;
          }
        }
      }
    }
  }
  
  .card-footer {
    display: flex;
    gap: var(--spacing-sm);
    
    .action-btn {
      flex: 1;
      padding: var(--spacing-sm) var(--spacing-md);
      border: 1px solid rgba(0, 0, 0, 0.1);
      border-radius: var(--radius-sm);
      background: transparent;
      color: var(--systemPrimary);
      font: var(--footnote-emphasized);
      cursor: pointer;
      transition: all var(--transition-fast);
      
      &:hover {
        background: rgba(0, 0, 0, 0.05);
      }
      
      &.primary {
        background: var(--keyColor);
        color: white;
        border-color: transparent;
        
        &:hover {
          background: var(--keyColorHover);
        }
      }
    }
  }
</style>
