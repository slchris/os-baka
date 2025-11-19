<script lang="ts">
  const currentYear = new Date().getFullYear()
</script>

<footer class="footer">
  <div class="footer__container">
    <div class="footer__content">
      <p class="footer__text">
        © {currentYear} OS Baka. All rights reserved.
      </p>
      <div class="footer__links">
        <a href="#/docs" class="footer__link">文档</a>
      </div>
    </div>
  </div>
</footer>

<style lang="scss">
  .footer {
    background: var(--footerBg);
    border-top: 1px solid rgba(0, 0, 0, 0.1);
    margin-top: auto;
    
    @media (prefers-color-scheme: dark) {
      border-top-color: rgba(255, 255, 255, 0.1);
    }
  }
  
  .footer__container {
    max-width: 1200px;
    margin: 0 auto;
    padding: var(--spacing-xl) var(--spacing-lg);
  }
  
  .footer__content {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--spacing-md);
  }
  
  .footer__text {
    font: var(--footnote);
    color: var(--systemSecondary);
    margin: 0;
  }
  
  .footer__links {
    display: flex;
    gap: var(--spacing-lg);
  }
  
  .footer__link {
    font: var(--footnote);
    color: var(--systemSecondary);
    text-decoration: none;
    transition: color var(--transition-fast);
    
    &:hover {
      color: var(--systemPrimary);
    }
  }
</style>
