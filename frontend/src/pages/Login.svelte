<script lang="ts">
  import { api } from '../lib/api'
  import Button from '../components/Button.svelte'
  
  let username = ''
  let password = ''
  let error = ''
  let loading = false
  
  async function handleLogin() {
    if (!username || !password) {
      error = '请输入用户名和密码'
      return
    }
    
    loading = true
    error = ''
    
    try {
      const response = await api.post('/auth/login', {
        username,
        password
      })
      
      // 保存 token
      localStorage.setItem('access_token', response.data.access_token)
      
      // 跳转到主页
      window.location.href = '/'
    } catch (err: any) {
      if (err.response?.status === 401) {
        error = '用户名或密码错误'
      } else if (err.response?.status === 403) {
        error = '账户已被禁用'
      } else {
        error = '登录失败，请稍后重试'
      }
    } finally {
      loading = false
    }
  }
  
  function handleKeyPress(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleLogin()
    }
  }
</script>

<div class="login-container">
  <div class="login-card">
    <div class="login-header">
      <h1 class="brand-title">OS Baka</h1>
      <p class="brand-subtitle">资产管理与 PXE 服务平台</p>
    </div>
    
    <form class="login-form" on:submit|preventDefault={handleLogin}>
      <div class="form-group">
        <label for="username" class="form-label">用户名</label>
        <input
          id="username"
          type="text"
          class="form-input"
          bind:value={username}
          on:keypress={handleKeyPress}
          placeholder="请输入用户名"
          disabled={loading}
          autocomplete="username"
        />
      </div>
      
      <div class="form-group">
        <label for="password" class="form-label">密码</label>
        <input
          id="password"
          type="password"
          class="form-input"
          bind:value={password}
          on:keypress={handleKeyPress}
          placeholder="请输入密码"
          disabled={loading}
          autocomplete="current-password"
        />
      </div>
      
      {#if error}
        <div class="error-message">
          <svg class="error-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="12" y1="8" x2="12" y2="12"></line>
            <line x1="12" y1="16" x2="12.01" y2="16"></line>
          </svg>
          <span>{error}</span>
        </div>
      {/if}
      
      <Button
        variant="primary"
        fullWidth
        disabled={loading}
        on:click={handleLogin}
      >
        {loading ? '登录中...' : '登录'}
      </Button>
    </form>
    
    <div class="login-footer">
      <p class="hint-text">默认账户: admin / admin123</p>
    </div>
  </div>
</div>

<style lang="scss">
  .login-container {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    padding: var(--spacing-lg);
    
    @media (prefers-color-scheme: dark) {
      background: linear-gradient(135deg, #1a202c 0%, #2d3748 100%);
    }
  }
  
  .login-card {
    background: rgba(255, 255, 255, 0.95);
    border-radius: var(--radius-lg);
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
    padding: var(--spacing-2xl);
    width: 100%;
    max-width: 420px;
    backdrop-filter: blur(10px);
    
    @media (prefers-color-scheme: dark) {
      background: rgba(26, 32, 44, 0.95);
    }
  }
  
  .login-header {
    text-align: center;
    margin-bottom: var(--spacing-2xl);
  }
  
  .brand-title {
    font: var(--title-1);
    color: var(--systemPrimary);
    margin: 0 0 var(--spacing-xs) 0;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  }
  
  .brand-subtitle {
    font: var(--body);
    color: var(--systemSecondary);
    margin: 0;
  }
  
  .login-form {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-lg);
  }
  
  .form-group {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-xs);
  }
  
  .form-label {
    font: var(--callout);
    font-weight: 500;
    color: var(--systemPrimary);
  }
  
  .form-input {
    font: var(--body);
    padding: var(--spacing-md) var(--spacing-lg);
    border: 1px solid rgba(0, 0, 0, 0.1);
    border-radius: var(--radius-md);
    background: var(--defaultBackground);
    color: var(--systemPrimary);
    transition: all var(--transition-fast);
    
    &:focus {
      outline: none;
      border-color: var(--keyColor);
      box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
    }
    
    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
    
    &::placeholder {
      color: var(--systemTertiary);
    }
    
    @media (prefers-color-scheme: dark) {
      border-color: rgba(255, 255, 255, 0.1);
      background: rgba(255, 255, 255, 0.05);
    }
  }
  
  .error-message {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
    padding: var(--spacing-md);
    background: rgba(255, 59, 48, 0.1);
    border: 1px solid rgba(255, 59, 48, 0.3);
    border-radius: var(--radius-sm);
    color: #ff3b30;
    font: var(--callout);
  }
  
  .error-icon {
    width: 18px;
    height: 18px;
    flex-shrink: 0;
  }
  
  .login-footer {
    margin-top: var(--spacing-xl);
    text-align: center;
  }
  
  .hint-text {
    font: var(--footnote);
    color: var(--systemTertiary);
    margin: 0;
  }
</style>
