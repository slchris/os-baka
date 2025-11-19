<script lang="ts">
  import { onMount } from 'svelte'
  import { authStore } from './lib/auth'
  import Navigation from './components/Navigation.svelte'
  import Footer from './components/Footer.svelte'
  import AssetList from './pages/AssetList.svelte'
  import PXEService from './pages/PXEService.svelte'
  import Deploy from './pages/Deploy.svelte'
  import USBKeys from './pages/USBKeys.svelte'
  import Settings from './pages/Settings.svelte'
  import Login from './pages/Login.svelte'
  
  let isAuthenticated = false
  let isLoading = true
  let currentRoute = 'assets'
  
  // Subscribe to auth store
  authStore.subscribe(state => {
    isAuthenticated = state.isAuthenticated
    isLoading = state.isLoading
  })
  
  // Handle hash-based routing
  function updateRoute() {
    const hash = window.location.hash.slice(2) || 'assets' // Remove #/
    currentRoute = hash.split('/')[0]
  }
  
  // Initialize authentication on mount
  onMount(async () => {
    await authStore.init()
    
    updateRoute()
    window.addEventListener('hashchange', updateRoute)
    
    // Redirect to login if not authenticated
    if (!isAuthenticated && window.location.pathname !== '/login') {
      window.location.href = '/login'
    }
    
    return () => {
      window.removeEventListener('hashchange', updateRoute)
    }
  })
  
  // Simple routing based on path
  $: isLoginPage = window.location.pathname === '/login' || !isAuthenticated
</script>

{#if isLoading}
  <div class="loading-container">
    <div class="loading-spinner"></div>
    <p>加载中...</p>
  </div>
{:else if isLoginPage}
  <Login />
{:else}
  <div class="app-container">
    <Navigation />
    
    <main class="main-content">
      {#if currentRoute === 'assets'}
        <AssetList />
      {:else if currentRoute === 'pxe'}
        <PXEService />
      {:else if currentRoute === 'deploy'}
        <Deploy />
      {:else if currentRoute === 'usbkeys'}
        <USBKeys />
      {:else if currentRoute === 'settings'}
        <Settings />
      {:else}
        <AssetList />
      {/if}
    </main>
    
    <Footer />
  </div>
{/if}

<style lang="scss">
  .loading-container {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--spacing-lg);
    
    p {
      font: var(--body);
      color: var(--systemSecondary);
    }
  }
  
  .loading-spinner {
    width: 48px;
    height: 48px;
    border: 4px solid rgba(0, 122, 255, 0.1);
    border-top-color: var(--keyColor);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }
  
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
  
  .app-container {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }
  
  .main-content {
    flex: 1;
    width: 100%;
  }
  
  .placeholder-page {
    min-height: 80vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--spacing-md);
    
    h2 {
      font: var(--title-1);
      color: var(--systemPrimary);
      margin: 0;
    }
    
    p {
      font: var(--body);
      color: var(--systemSecondary);
      margin: 0;
    }
  }
</style>
