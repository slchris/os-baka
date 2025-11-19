<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  
  export let variant: 'primary' | 'secondary' = 'primary'
  export let disabled = false
  export let type: 'button' | 'submit' = 'button'
  export let fullWidth = false
  
  const dispatch = createEventDispatcher()
  
  function handleClick(event: MouseEvent) {
    if (!disabled) {
      dispatch('click', event)
    }
  }
</script>

<button
  class="button button--{variant}"
  class:button--disabled={disabled}
  class:button--full-width={fullWidth}
  {type}
  {disabled}
  on:click={handleClick}
>
  <slot />
</button>

<style lang="scss">
  .button {
    padding: 12px 24px;
    border-radius: var(--radius-sm);
    font: var(--body-emphasized);
    cursor: pointer;
    transition: all var(--transition-fast);
    border: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--spacing-xs);
    
    &--primary {
      background: var(--keyColor);
      color: white;
      
      &:hover:not(:disabled) {
        background: var(--keyColorHover);
      }
      
      &:active:not(:disabled) {
        opacity: 0.8;
      }
    }
    
    &--secondary {
      background: transparent;
      color: var(--keyColor);
      border: 1px solid rgba(0, 113, 227, 0.3);
      
      &:hover:not(:disabled) {
        background: rgba(0, 113, 227, 0.1);
      }
    }
    
    &--disabled,
    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
    
    &--full-width {
      width: 100%;
    }
  }
</style>
