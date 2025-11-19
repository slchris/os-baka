<script lang="ts">
  import { onMount } from 'svelte'
  import { assetApi, type Asset } from '../lib/api'
  import AssetCard from '../components/AssetCard.svelte'
  import AssetForm from '../components/AssetForm.svelte'
  import Button from '../components/Button.svelte'
  
  let assets: Asset[] = []
  let loading = true
  let error: string | null = null
  let showCreateModal = false
  
  onMount(async () => {
    await loadAssets()
  })
  
  async function loadAssets() {
    try {
      loading = true
      error = null
      const response = await assetApi.list()
      assets = response.items
    } catch (err) {
      error = err instanceof Error ? err.message : '加载资产失败'
      console.error('Failed to load assets:', err)
    } finally {
      loading = false
    }
  }
  
  function handleCreateClick() {
    showCreateModal = true
  }
  
  async function handleDeleteAsset(id: number) {
    if (!confirm('确定要删除这个资产吗？')) return
    
    try {
      await assetApi.delete(id)
      await loadAssets()
    } catch (err) {
      alert('删除失败: ' + (err instanceof Error ? err.message : '未知错误'))
    }
  }
  
  function handleFormSuccess() {
    loadAssets()
  }
</script>

<div class="asset-list-page">
  <div class="page-container">
    <header class="page-header">
      <div class="header-content">
        <h1 class="page-title">资产管理</h1>
        <p class="page-subtitle">管理和监控所有物理资产</p>
      </div>
      <Button on:click={handleCreateClick} variant="primary">
        + 添加资产
      </Button>
    </header>
    
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
        <Button on:click={loadAssets} variant="secondary">重试</Button>
      </div>
    {:else if assets.length === 0}
      <div class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path>
          <polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline>
          <line x1="12" y1="22.08" x2="12" y2="12"></line>
        </svg>
        <h3>暂无资产</h3>
        <p>开始添加您的第一个资产吧</p>
        <Button on:click={handleCreateClick} variant="primary">
          添加资产
        </Button>
      </div>
    {:else}
      <div class="asset-grid">
        {#each assets as asset (asset.id)}
          <AssetCard {asset} on:delete={() => handleDeleteAsset(asset.id)} />
        {/each}
      </div>
    {/if}
  </div>
</div>

<AssetForm bind:show={showCreateModal} on:success={handleFormSuccess} />

<style lang="scss">
  .asset-list-page {
    min-height: 100vh;
    background: var(--secondaryBackground);
  }
  
  .page-container {
    max-width: 1200px;
    margin: 0 auto;
    padding: var(--spacing-3xl) var(--spacing-lg);
  }
  
  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: var(--spacing-2xl);
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
  
  .asset-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: var(--spacing-lg);
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
    
    p {
      font: var(--body);
      color: var(--systemSecondary);
      margin: 0;
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
    
    .error-message {
      font: var(--body);
      color: var(--errorColor);
      margin: 0 0 var(--spacing-lg);
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
