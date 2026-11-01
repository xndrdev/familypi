<template>
  <main class="start">
    <h1>👪 Familien-Display</h1>
    <nav class="start__links">
      <NuxtLink to="/display" class="start__link">Display<span>Ansicht für den Monitor</span></NuxtLink>
      <NuxtLink to="/admin" class="start__link">Verwaltung<span>Todos, Termine, Ziele, Familie</span></NuxtLink>
    </nav>
  </main>
</template>

<style>
.start {
  max-width: 40rem;
  margin: 10vh auto;
  padding: 0 1rem;
  text-align: center;
}

.start__links {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
  gap: 1rem;
}

.start__link {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  padding: 1.5rem;
  font-size: 1.3rem;
  font-weight: 700;
  color: var(--text);
  text-decoration: none;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.start__link:hover {
  border-color: var(--accent);
}

.start__link span {
  font-size: 0.9rem;
  font-weight: 400;
  color: var(--muted);
}
</style>
