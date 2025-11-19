<script lang="ts">
  import { onMount } from 'svelte'
  import { deploymentApi, pxeApi, type PXEDeployment, type DeploymentStats } from '../lib/api'
  
  let deployments: PXEDeployment[] = []
  let stats: DeploymentStats | null = null
  let loading = true
  let error: string | null = null
  let selectedStatus: string | null = null
  
  const statusOptions = [
    { value: null, label: '全部' },
    { value: 'pending', label: '待部署' },
    { value: 'in_progress', label: '进行中' },
    { value: 'completed', label: '已完成' },
    { value: 'failed', label: '失败' }
  ]
  
  const statusColors: Record<string, string> = {
    pending: '#ff9500',
    in_progress: '#007aff',
    completed: '#34c759',
    failed: '#ff3b30'
  }
  
  const statusLabels: Record<string, string> = {
    pending: '待部署',
    in_progress: '进行中',
    completed: '已完成',
    failed: '失败'
  }
  
  onMount(async () => {
    await Promise.all([loadDeployments(), loadStats()])
  })
  
  async function loadDeployments() {
    try {
      loading = true
      error = null
      deployments = await deploymentApi.list({
        status: selectedStatus || undefined,
        limit: 50
      })
    } catch (err) {
      error = err instanceof Error ? err.message : '加载部署记录失败'
      console.error('Failed to load deployments:', err)
    } finally {
      loading = false
    }
  }
  
  async function loadStats() {
    try {
      stats = await deploymentApi.getStats()
    } catch (err) {
      console.error('Failed to load stats:', err)
    }
  }
  
  async function handleStatusFilter(status: string | null) {
    selectedStatus = status
    await loadDeployments()
  }
  
  function formatDate(dateString: string) {
    const date = new Date(dateString)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  }
  
  function formatDuration(started: string, completed: string | undefined) {
    if (!completed) return '—'
    const start = new Date(started).getTime()
    const end = new Date(completed).getTime()
    const seconds = Math.floor((end - start) / 1000)
    if (seconds < 60) return `${seconds}秒`
    const minutes = Math.floor(seconds / 60)
    if (minutes < 60) return `${minutes}分钟`
    const hours = Math.floor(minutes / 60)
    return `${hours}小时${minutes % 60}分钟`
  }
</script>

<div class="deploy-page">
  <div class="page-container">
    <header class="page-header">
      <div class="header-content">
        <h1 class="page-title">部署管理</h1>
        <p class="page-subtitle">查看PXE部署历史和状态</p>
      </div>
    </header>
    
    {#if stats}
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-value">{stats.total}</div>
          <div class="stat-label">总计</div>
        </div>
        <div class="stat-card status-pending">
          <div class="stat-value">{stats.pending}</div>
          <div class="stat-label">待部署</div>
        </div>
        <div class="stat-card status-in-progress">
          <div class="stat-value">{stats.in_progress}</div>
          <div class="stat-label">进行中</div>
        </div>
        <div class="stat-card status-completed">
          <div class="stat-value">{stats.completed}</div>
          <div class="stat-label">已完成</div>
        </div>
        <div class="stat-card status-failed">
          <div class="stat-value">{stats.failed}</div>
          <div class="stat-label">失败</div>
        </div>
      </div>
    {/if}
    
    <div class="filters">
      {#each statusOptions as option}
        <button
          class="filter-btn"
          class:active={selectedStatus === option.value}
          on:click={() => handleStatusFilter(option.value)}
        >
          {option.label}
        </button>
      {/each}
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
      </div>
    {:else if deployments.length === 0}
      <div class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <polygon points="10 8 16 12 10 16 10 8"></polygon>
        </svg>
        <h3>暂无部署记录</h3>
        <p>还没有进行过任何部署</p>
      </div>
    {:else}
      <div class="deployment-list">
        {#each deployments as deployment}
          <div class="deployment-card">
            <div class="deployment-header">
              <div class="deployment-info">
                <h3>部署 #{deployment.id}</h3>
                <span 
                  class="status-badge" 
                  style="background-color: {statusColors[deployment.status]}20; color: {statusColors[deployment.status]}"
                >
                  {statusLabels[deployment.status]}
                </span>
              </div>
              <div class="deployment-time">
                {formatDate(deployment.started_at)}
              </div>
            </div>
            
            <div class="deployment-body">
              <div class="detail-row">
                <span class="detail-label">PXE配置ID:</span>
                <span class="detail-value">{deployment.pxe_config_id}</span>
              </div>
              
              {#if deployment.completed_at}
                <div class="detail-row">
                  <span class="detail-label">持续时间:</span>
                  <span class="detail-value">{formatDuration(deployment.started_at, deployment.completed_at)}</span>
                </div>
              {/if}
              
              {#if deployment.error_message}
                <div class="error-box">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"></circle>
                    <line x1="12" y1="8" x2="12" y2="12"></line>
                    <line x1="12" y1="16" x2="12.01" y2="16"></line>
                  </svg>
                  <span>{deployment.error_message}</span>
                </div>
              {/if}
              
              {#if deployment.log_output}
                <details class="log-details">
                  <summary>查看日志</summary>
                  <pre class="log-output">{deployment.log_output}</pre>
                </details>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style lang="scss">
  .deploy-page {
    min-height: 100vh;
    background: var(--secondaryBackground);
  }
  
  .page-container {
    max-width: 1200px;
    margin: 0 auto;
    padding: var(--spacing-3xl) var(--spacing-lg);
  }
  
  .page-header {
    margin-bottom: var(--spacing-xl);
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
  
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: var(--spacing-lg);
    margin-bottom: var(--spacing-xl);
  }
  
  .stat-card {
    background: var(--defaultBackground);
    border-radius: var(--radius-md);
    padding: var(--spacing-xl);
    box-shadow: var(--shadow-card);
    text-align: center;
    
    &.status-pending { border-left: 4px solid #ff9500; }
    &.status-in-progress { border-left: 4px solid #007aff; }
    &.status-completed { border-left: 4px solid #34c759; }
    &.status-failed { border-left: 4px solid #ff3b30; }
  }
  
  .stat-value {
    font-size: 36px;
    font-weight: 700;
    color: var(--systemPrimary);
    margin-bottom: var(--spacing-xs);
  }
  
  .stat-label {
    font: var(--callout);
    color: var(--systemSecondary);
  }
  
  .filters {
    display: flex;
    gap: var(--spacing-sm);
    margin-bottom: var(--spacing-xl);
    flex-wrap: wrap;
  }
  
  .filter-btn {
    padding: var(--spacing-sm) var(--spacing-lg);
    border: 1px solid rgba(0, 0, 0, 0.1);
    border-radius: var(--radius-sm);
    background: var(--defaultBackground);
    color: var(--systemSecondary);
    font: var(--body);
    cursor: pointer;
    transition: all var(--transition-fast);
    
    &:hover {
      background: rgba(0, 0, 0, 0.03);
      border-color: rgba(0, 0, 0, 0.2);
    }
    
    &.active {
      background: var(--keyColor);
      color: white;
      border-color: var(--keyColor);
    }
  }
  
  .deployment-list {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-lg);
  }
  
  .deployment-card {
    background: var(--defaultBackground);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-card);
    overflow: hidden;
  }
  
  .deployment-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--spacing-lg);
    border-bottom: 1px solid rgba(0, 0, 0, 0.05);
    
    @media (prefers-color-scheme: dark) {
      border-bottom-color: rgba(255, 255, 255, 0.05);
    }
  }
  
  .deployment-info {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
    
    h3 {
      font: var(--title-3);
      color: var(--systemPrimary);
      margin: 0;
    }
  }
  
  .status-badge {
    padding: 4px 12px;
    border-radius: 12px;
    font: var(--caption-1);
    font-weight: 600;
  }
  
  .deployment-time {
    font: var(--callout);
    color: var(--systemSecondary);
  }
  
  .deployment-body {
    padding: var(--spacing-lg);
  }
  
  .detail-row {
    display: flex;
    gap: var(--spacing-sm);
    margin-bottom: var(--spacing-sm);
    
    .detail-label {
      font: var(--callout);
      color: var(--systemSecondary);
      font-weight: 500;
    }
    
    .detail-value {
      font: var(--callout);
      color: var(--systemPrimary);
    }
  }
  
  .error-box {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
    padding: var(--spacing-md);
    background: rgba(255, 59, 48, 0.1);
    border: 1px solid rgba(255, 59, 48, 0.3);
    border-radius: var(--radius-sm);
    color: #ff3b30;
    margin-top: var(--spacing-md);
    
    svg {
      width: 20px;
      height: 20px;
      flex-shrink: 0;
    }
  }
  
  .log-details {
    margin-top: var(--spacing-md);
    
    summary {
      cursor: pointer;
      font: var(--callout);
      color: var(--keyColor);
      padding: var(--spacing-sm);
      
      &:hover {
        text-decoration: underline;
      }
    }
  }
  
  .log-output {
    margin-top: var(--spacing-sm);
    padding: var(--spacing-md);
    background: rgba(0, 0, 0, 0.03);
    border-radius: var(--radius-sm);
    font-family: 'SF Mono', Monaco, 'Courier New', monospace;
    font-size: 12px;
    line-height: 1.6;
    color: var(--systemPrimary);
    overflow-x: auto;
    max-height: 300px;
    overflow-y: auto;
    
    @media (prefers-color-scheme: dark) {
      background: rgba(255, 255, 255, 0.05);
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
      margin: 0;
    }
  }
</style>
