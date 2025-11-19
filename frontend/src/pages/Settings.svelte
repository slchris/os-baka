<script lang="ts">
  import { onMount } from 'svelte'
  import { settingsApi, type SystemSetting, type SettingCreate } from '../lib/api'
  
  let settings: SystemSetting[] = []
  let categories: string[] = []
  let selectedCategory: string | null = null
  let loading = true
  let error: string | null = null
  let isAdmin = false
  
  let showCreateModal = false
  let showEditModal = false
  let editingSetting: SystemSetting | null = null
  
  let formData: SettingCreate = {
    key: '',
    value: '',
    value_type: 'string',
    category: 'general',
    description: '',
    is_public: false
  }
  
  const valueTypes = [
    { value: 'string', label: '字符串' },
    { value: 'int', label: '整数' },
    { value: 'bool', label: '布尔值' },
    { value: 'json', label: 'JSON' }
  ]
  
  onMount(async () => {
    await checkAdmin()
    await Promise.all([loadSettings(), loadCategories()])
  })
  
  async function checkAdmin() {
    try {
      const token = localStorage.getItem('token')
      if (!token) return
      
      const payload = JSON.parse(atob(token.split('.')[1]))
      isAdmin = payload.is_superuser === true
    } catch (err) {
      console.error('Failed to check admin status:', err)
    }
  }
  
  async function loadSettings() {
    try {
      loading = true
      error = null
      const response = await settingsApi.list({ 
        category: selectedCategory || undefined 
      })
      settings = response.items
    } catch (err) {
      error = err instanceof Error ? err.message : '加载设置失败'
      console.error('Failed to load settings:', err)
    } finally {
      loading = false
    }
  }
  
  async function loadCategories() {
    try {
      categories = await settingsApi.listCategories()
    } catch (err) {
      console.error('Failed to load categories:', err)
    }
  }
  
  async function handleCategoryFilter(category: string | null) {
    selectedCategory = category
    await loadSettings()
  }
  
  function openCreateModal() {
    formData = {
      key: '',
      value: '',
      value_type: 'string',
      category: 'general',
      description: '',
      is_public: false
    }
    showCreateModal = true
  }
  
  function openEditModal(setting: SystemSetting) {
    editingSetting = setting
    formData = {
      key: setting.key,
      value: setting.value || '',
      value_type: setting.value_type,
      category: setting.category,
      description: setting.description || '',
      is_public: setting.is_public
    }
    showEditModal = true
  }
  
  function closeModals() {
    showCreateModal = false
    showEditModal = false
    editingSetting = null
  }
  
  async function handleCreate() {
    try {
      await settingsApi.create(formData)
      closeModals()
      await loadSettings()
    } catch (err) {
      alert(err instanceof Error ? err.message : '创建设置失败')
    }
  }
  
  async function handleUpdate() {
    if (!editingSetting) return
    try {
      const { key, ...updateData } = formData
      await settingsApi.update(editingSetting.key, updateData)
      closeModals()
      await loadSettings()
    } catch (err) {
      alert(err instanceof Error ? err.message : '更新设置失败')
    }
  }
  
  async function handleDelete(key: string) {
    if (!confirm('确定要删除此设置吗？')) return
    try {
      await settingsApi.delete(key)
      await loadSettings()
    } catch (err) {
      alert(err instanceof Error ? err.message : '删除设置失败')
    }
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
  
  function formatValue(value: string, type: string) {
    if (type === 'bool') {
      return value === 'true' ? '是' : '否'
    }
    if (type === 'json') {
      try {
        return JSON.stringify(JSON.parse(value), null, 2)
      } catch {
        return value
      }
    }
    return value
  }
</script>

<div class="settings-page">
  <div class="page-container">
    <header class="page-header">
      <div class="header-content">
        <h1 class="page-title">系统设置</h1>
        <p class="page-subtitle">管理系统配置参数</p>
      </div>
      {#if isAdmin}
        <button class="btn-primary" on:click={openCreateModal}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          新建设置
        </button>
      {/if}
    </header>
    
    <div class="filters">
      <button
        class="filter-btn"
        class:active={selectedCategory === null}
        on:click={() => handleCategoryFilter(null)}
      >
        全部
      </button>
      {#each categories as category}
        <button
          class="filter-btn"
          class:active={selectedCategory === category}
          on:click={() => handleCategoryFilter(category)}
        >
          {category}
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
    {:else if settings.length === 0}
      <div class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <path d="M12 6v6l4 2"></path>
        </svg>
        <h3>暂无设置</h3>
        <p>还没有创建任何设置项</p>
      </div>
    {:else}
      <div class="settings-table">
        <div class="table-header">
          <div class="col-key">键名</div>
          <div class="col-value">值</div>
          <div class="col-type">类型</div>
          <div class="col-category">分类</div>
          <div class="col-public">公开</div>
          <div class="col-updated">更新时间</div>
          {#if isAdmin}
            <div class="col-actions">操作</div>
          {/if}
        </div>
        {#each settings as setting}
          <div class="table-row">
            <div class="col-key">
              <span class="key-text">{setting.key}</span>
              {#if setting.description}
                <span class="description">{setting.description}</span>
              {/if}
            </div>
            <div class="col-value">
              <code class="value-code">{formatValue(setting.value || '', setting.value_type)}</code>
            </div>
            <div class="col-type">
              <span class="type-badge">{setting.value_type}</span>
            </div>
            <div class="col-category">
              <span class="category-badge">{setting.category}</span>
            </div>
            <div class="col-public">
              {#if setting.is_public}
                <svg class="icon-check" viewBox="0 0 24 24" fill="none" stroke="#34c759" stroke-width="2">
                  <polyline points="20 6 9 17 4 12"></polyline>
                </svg>
              {:else}
                <svg class="icon-x" viewBox="0 0 24 24" fill="none" stroke="#ff3b30" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"></line>
                  <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
              {/if}
            </div>
            <div class="col-updated">
              {formatDate(setting.updated_at)}
            </div>
            {#if isAdmin}
              <div class="col-actions">
                <button class="btn-icon" on:click={() => openEditModal(setting)} title="编辑">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                  </svg>
                </button>
                <button class="btn-icon btn-danger" on:click={() => handleDelete(setting.key)} title="删除">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"></polyline>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                  </svg>
                </button>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

{#if showCreateModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="modal-overlay" on:click={closeModals}>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="modal" on:click|stopPropagation>
      <div class="modal-header">
        <h2>新建设置</h2>
        <button class="btn-close" on:click={closeModals}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>
      <form class="modal-body" on:submit|preventDefault={handleCreate}>
        <div class="form-group">
          <label for="key">键名 *</label>
          <input id="key" type="text" bind:value={formData.key} required />
        </div>
        
        <div class="form-group">
          <label for="value">值 *</label>
          <textarea id="value" bind:value={formData.value} required rows="3"></textarea>
        </div>
        
        <div class="form-row">
          <div class="form-group">
            <label for="value_type">类型 *</label>
            <select id="value_type" bind:value={formData.value_type} required>
              {#each valueTypes as type}
                <option value={type.value}>{type.label}</option>
              {/each}
            </select>
          </div>
          
          <div class="form-group">
            <label for="category">分类 *</label>
            <select id="category" bind:value={formData.category} required>
              <option value="general">general</option>
              <option value="pxe">pxe</option>
              <option value="network">network</option>
              <option value="security">security</option>
            </select>
          </div>
        </div>
        
        <div class="form-group">
          <label for="description">描述</label>
          <input id="description" type="text" bind:value={formData.description} />
        </div>
        
        <div class="form-group checkbox-group">
          <label>
            <input type="checkbox" bind:checked={formData.is_public} />
            <span>公开访问（允许非管理员查看）</span>
          </label>
        </div>
        
        <div class="modal-footer">
          <button type="button" class="btn-secondary" on:click={closeModals}>取消</button>
          <button type="submit" class="btn-primary">创建</button>
        </div>
      </form>
    </div>
  </div>
{/if}

{#if showEditModal && editingSetting}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="modal-overlay" on:click={closeModals}>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="modal" on:click|stopPropagation>
      <div class="modal-header">
        <h2>编辑设置: {editingSetting.key}</h2>
        <button class="btn-close" on:click={closeModals}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>
      <form class="modal-body" on:submit|preventDefault={handleUpdate}>
        <div class="form-group">
          <label for="edit-value">值 *</label>
          <textarea id="edit-value" bind:value={formData.value} required rows="3"></textarea>
        </div>
        
        <div class="form-row">
          <div class="form-group">
            <label for="edit-value_type">类型 *</label>
            <select id="edit-value_type" bind:value={formData.value_type} required>
              {#each valueTypes as type}
                <option value={type.value}>{type.label}</option>
              {/each}
            </select>
          </div>
          
          <div class="form-group">
            <label for="edit-category">分类 *</label>
            <select id="edit-category" bind:value={formData.category} required>
              <option value="general">general</option>
              <option value="pxe">pxe</option>
              <option value="network">network</option>
              <option value="security">security</option>
            </select>
          </div>
        </div>
        
        <div class="form-group">
          <label for="edit-description">描述</label>
          <input id="edit-description" type="text" bind:value={formData.description} />
        </div>
        
        <div class="form-group checkbox-group">
          <label>
            <input type="checkbox" bind:checked={formData.is_public} />
            <span>公开访问（允许非管理员查看）</span>
          </label>
        </div>
        
        <div class="modal-footer">
          <button type="button" class="btn-secondary" on:click={closeModals}>取消</button>
          <button type="submit" class="btn-primary">保存</button>
        </div>
      </form>
    </div>
  </div>
{/if}

<style lang="scss">
  .settings-page {
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
  }
  
  .btn-primary {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
    padding: var(--spacing-sm) var(--spacing-lg);
    background: var(--keyColor);
    color: white;
    border: none;
    border-radius: var(--radius-sm);
    font: var(--body);
    font-weight: 600;
    cursor: pointer;
    transition: all var(--transition-fast);
    
    svg {
      width: 18px;
      height: 18px;
    }
    
    &:hover {
      opacity: 0.85;
    }
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
  
  .settings-table {
    background: var(--defaultBackground);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-card);
    overflow: hidden;
  }
  
  .table-header,
  .table-row {
    display: grid;
    grid-template-columns: 2fr 2fr 1fr 1fr 80px 1.5fr 100px;
    gap: var(--spacing-md);
    align-items: center;
    padding: var(--spacing-lg);
  }
  
  .table-header {
    background: rgba(0, 0, 0, 0.02);
    border-bottom: 1px solid rgba(0, 0, 0, 0.05);
    font: var(--caption-1);
    font-weight: 600;
    color: var(--systemSecondary);
    text-transform: uppercase;
    
    @media (prefers-color-scheme: dark) {
      background: rgba(255, 255, 255, 0.02);
      border-bottom-color: rgba(255, 255, 255, 0.05);
    }
  }
  
  .table-row {
    border-bottom: 1px solid rgba(0, 0, 0, 0.05);
    
    &:last-child {
      border-bottom: none;
    }
    
    @media (prefers-color-scheme: dark) {
      border-bottom-color: rgba(255, 255, 255, 0.05);
    }
  }
  
  .col-key {
    .key-text {
      display: block;
      font: var(--callout);
      font-weight: 600;
      color: var(--systemPrimary);
    }
    
    .description {
      display: block;
      font: var(--caption-1);
      color: var(--systemTertiary);
      margin-top: 2px;
    }
  }
  
  .col-value {
    .value-code {
      font-family: 'SF Mono', Monaco, 'Courier New', monospace;
      font-size: 12px;
      color: var(--systemPrimary);
      background: rgba(0, 0, 0, 0.03);
      padding: 4px 8px;
      border-radius: 4px;
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: pre-wrap;
      word-break: break-all;
      
      @media (prefers-color-scheme: dark) {
        background: rgba(255, 255, 255, 0.05);
      }
    }
  }
  
  .type-badge,
  .category-badge {
    display: inline-block;
    padding: 4px 10px;
    border-radius: 12px;
    font: var(--caption-2);
    font-weight: 600;
  }
  
  .type-badge {
    background: rgba(0, 122, 255, 0.1);
    color: #007aff;
  }
  
  .category-badge {
    background: rgba(52, 199, 89, 0.1);
    color: #34c759;
  }
  
  .icon-check,
  .icon-x {
    width: 20px;
    height: 20px;
  }
  
  .col-actions {
    display: flex;
    gap: var(--spacing-xs);
  }
  
  .btn-icon {
    padding: var(--spacing-xs);
    background: transparent;
    border: 1px solid rgba(0, 0, 0, 0.1);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--transition-fast);
    
    svg {
      width: 16px;
      height: 16px;
      color: var(--systemSecondary);
    }
    
    &:hover {
      background: rgba(0, 0, 0, 0.03);
      border-color: rgba(0, 0, 0, 0.2);
      
      svg {
        color: var(--systemPrimary);
      }
    }
    
    &.btn-danger:hover {
      background: rgba(255, 59, 48, 0.1);
      border-color: #ff3b30;
      
      svg {
        color: #ff3b30;
      }
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
  
  .error-state .error-icon,
  .empty-state .empty-icon {
    margin: 0 auto var(--spacing-lg);
    color: var(--systemTertiary);
  }
  
  .error-state .error-icon {
    width: 48px;
    height: 48px;
    color: #ff3b30;
  }
  
  .empty-state {
    .empty-icon {
      width: 80px;
      height: 80px;
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
  
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
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
    box-shadow: var(--shadow-modal);
    max-width: 600px;
    width: 90%;
    max-height: 90vh;
    overflow-y: auto;
  }
  
  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--spacing-lg);
    border-bottom: 1px solid rgba(0, 0, 0, 0.05);
    
    h2 {
      font: var(--title-2);
      color: var(--systemPrimary);
      margin: 0;
    }
    
    @media (prefers-color-scheme: dark) {
      border-bottom-color: rgba(255, 255, 255, 0.05);
    }
  }
  
  .btn-close {
    padding: var(--spacing-xs);
    background: transparent;
    border: none;
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
    padding: var(--spacing-lg);
  }
  
  .form-group {
    margin-bottom: var(--spacing-lg);
    
    label {
      display: block;
      font: var(--callout);
      font-weight: 600;
      color: var(--systemPrimary);
      margin-bottom: var(--spacing-xs);
    }
    
    input[type="text"],
    textarea,
    select {
      width: 100%;
      padding: var(--spacing-sm) var(--spacing-md);
      border: 1px solid rgba(0, 0, 0, 0.1);
      border-radius: var(--radius-sm);
      font: var(--body);
      color: var(--systemPrimary);
      background: var(--defaultBackground);
      transition: border-color var(--transition-fast);
      
      &:focus {
        outline: none;
        border-color: var(--keyColor);
      }
      
      @media (prefers-color-scheme: dark) {
        border-color: rgba(255, 255, 255, 0.1);
      }
    }
    
    textarea {
      resize: vertical;
      font-family: 'SF Mono', Monaco, 'Courier New', monospace;
    }
    
    &.checkbox-group {
      label {
        display: flex;
        align-items: center;
        gap: var(--spacing-sm);
        font-weight: normal;
        cursor: pointer;
        
        input[type="checkbox"] {
          width: auto;
          cursor: pointer;
        }
      }
    }
  }
  
  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--spacing-md);
  }
  
  .modal-footer {
    display: flex;
    gap: var(--spacing-sm);
    justify-content: flex-end;
    padding-top: var(--spacing-lg);
    border-top: 1px solid rgba(0, 0, 0, 0.05);
    
    @media (prefers-color-scheme: dark) {
      border-top-color: rgba(255, 255, 255, 0.05);
    }
  }
  
  .btn-secondary {
    padding: var(--spacing-sm) var(--spacing-lg);
    background: transparent;
    color: var(--systemSecondary);
    border: 1px solid rgba(0, 0, 0, 0.1);
    border-radius: var(--radius-sm);
    font: var(--body);
    font-weight: 600;
    cursor: pointer;
    transition: all var(--transition-fast);
    
    &:hover {
      background: rgba(0, 0, 0, 0.03);
      color: var(--systemPrimary);
    }
  }
</style>
