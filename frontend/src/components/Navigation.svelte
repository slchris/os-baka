<script lang="ts">
  import { authStore } from '../lib/auth'
  
  let user = $authStore.user
  let currentRoute = 'assets'
  
  authStore.subscribe(state => {
    user = state.user
  })
  
  function updateRoute() {
    const hash = window.location.hash.slice(2) || 'assets' // Remove #/
    currentRoute = hash.split('/')[0]
  }
  
  // Update on mount and hash change
  $: if (typeof window !== 'undefined') {
    updateRoute()
    window.addEventListener('hashchange', updateRoute)
  }
  
  function handleLogout() {
    if (confirm('确定要退出登录吗？')) {
      authStore.logout()
      window.location.href = '/login'
    }
  }
</script>

<nav class="navigation">
  <div class="navigation__container">
    <div class="navigation__brand">
      <h1 class="brand-title">OS Baka</h1>
    </div>
    
    <div class="navigation__menu">
      <a href="#/assets" class="nav-link" class:active={currentRoute === 'assets'}>
        <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path>
          <polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline>
          <line x1="12" y1="22.08" x2="12" y2="12"></line>
        </svg>
        <span class="nav-label">资产管理</span>
      </a>
      <a href="#/pxe" class="nav-link" class:active={currentRoute === 'pxe'}>
        <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
          <line x1="8" y1="21" x2="16" y2="21"></line>
          <line x1="12" y1="17" x2="12" y2="21"></line>
        </svg>
        <span class="nav-label">PXE 服务</span>
      </a>
      <a href="#/deploy" class="nav-link" class:active={currentRoute === 'deploy'}>
        <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <polygon points="10 8 16 12 10 16 10 8"></polygon>
        </svg>
        <span class="nav-label">部署管理</span>
      </a>
      <a href="#/usbkeys" class="nav-link" class:active={currentRoute === 'usbkeys'}>
        <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="6" y="2" width="12" height="20" rx="2"></rect>
          <line x1="12" y1="18" x2="12.01" y2="18"></line>
        </svg>
        <span class="nav-label">USB Key</span>
      </a>
      <a href="#/settings" class="nav-link" class:active={currentRoute === 'settings'}>
        <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3"></circle>
          <path d="M12 1v6m0 6v6m-9-9h6m6 0h6m-4.22-4.22l-4.24 4.24m0 6l4.24 4.24m-8.48 0l4.24-4.24m0-6L7.76 2.76"></path>
        </svg>
        <span class="nav-label">设置</span>
      </a>
    </div>
    
    <div class="navigation__user">
      <span class="user-name">{user?.username || '用户'}</span>
      <button class="logout-button" on:click={handleLogout}>
        <svg class="logout-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
          <polyline points="16 17 21 12 16 7"></polyline>
          <line x1="21" y1="12" x2="9" y2="12"></line>
        </svg>
        <span>退出</span>
      </button>
    </div>
  </div>
</nav>

<style lang="scss">
  .navigation {
    background: var(--defaultBackground);
    border-bottom: 1px solid rgba(0, 0, 0, 0.1);
    position: sticky;
    top: 0;
    z-index: 100;
    backdrop-filter: blur(20px);
    background: rgba(255, 255, 255, 0.8);
    
    @media (prefers-color-scheme: dark) {
      background: rgba(0, 0, 0, 0.8);
      border-bottom-color: rgba(255, 255, 255, 0.1);
    }
  }
  
  .navigation__container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 0 var(--spacing-lg);
    height: 52px;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  
  .navigation__brand {
    .brand-title {
      font: var(--title-3);
      color: var(--systemPrimary);
      margin: 0;
    }
  }
  
  .navigation__menu {
    display: flex;
    gap: var(--spacing-md);
  }
  
  .nav-link {
    display: flex;
    align-items: center;
    gap: var(--spacing-xs);
    padding: var(--spacing-sm) var(--spacing-md);
    border-radius: var(--radius-sm);
    text-decoration: none;
    color: var(--systemSecondary);
    font: var(--body);
    transition: all var(--transition-fast);
    
    &:hover {
      background: rgba(0, 0, 0, 0.05);
      color: var(--systemPrimary);
      
      @media (prefers-color-scheme: dark) {
        background: rgba(255, 255, 255, 0.1);
      }
    }
    
    &.active {
      background: var(--keyColor);
      color: white;
      
      &:hover {
        background: var(--keyColorHover);
      }
    }
  }
  
  .nav-icon {
    width: 18px;
    height: 18px;
  }
  
  .nav-label {
    font: var(--body);
  }
  
  .navigation__user {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
  }
  
  .user-name {
    font: var(--callout);
    color: var(--systemPrimary);
  }
  
  .logout-button {
    display: flex;
    align-items: center;
    gap: var(--spacing-xs);
    padding: var(--spacing-sm) var(--spacing-md);
    border: 1px solid rgba(0, 0, 0, 0.1);
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--systemSecondary);
    font: var(--body);
    cursor: pointer;
    transition: all var(--transition-fast);
    
    &:hover {
      background: rgba(255, 59, 48, 0.1);
      border-color: rgba(255, 59, 48, 0.3);
      color: #ff3b30;
      
      @media (prefers-color-scheme: dark) {
        background: rgba(255, 69, 58, 0.1);
      }
    }
    
    @media (prefers-color-scheme: dark) {
      border-color: rgba(255, 255, 255, 0.1);
    }
  }
  
  .logout-icon {
    width: 16px;
    height: 16px;
  }
</style>
