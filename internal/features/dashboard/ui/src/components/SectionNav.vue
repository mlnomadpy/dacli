<template>
  <nav
    aria-label="Dashboard sections"
    class="section-nav sticky top-3 z-20 my-5 flex items-center gap-1 overflow-x-auto rounded-md border border-border bg-background/90 p-1 shadow-[0_14px_36px_-28px_#000] backdrop-blur-md"
  >
    <a href="#pulse"><span>01</span>Pulse</a>
    <a href="#delivery"><span>02</span>Delivery</a>
    <a href="#agents"><span>03</span>Agents</a>
    <a href="#team"><span>04</span>Team</a>
  </nav>
</template>

<style scoped>
.section-nav a {
  display: inline-flex;
  min-height: 36px;
  flex: 1 0 auto;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border-radius: 0.2rem;
  padding: 0 0.9rem;
  color: var(--muted-foreground);
  font-size: 0.75rem;
  font-weight: 600;
  text-decoration: none;
  transition:
    background-color 150ms ease,
    color 150ms ease;
}
.section-nav a span {
  color: var(--primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.62rem;
}
.section-nav a:hover {
  background: var(--secondary);
  color: var(--foreground);
}
.section-nav a:focus-visible {
  outline: 2px solid var(--ring);
  outline-offset: 2px;
}
</style>
