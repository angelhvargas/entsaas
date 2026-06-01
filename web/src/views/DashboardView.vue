<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { FolderKanban, Users, Activity, ArrowUpRight } from 'lucide-vue-next'
import client from '@/api/client'
import { UiCard, UiEmptyState, UiLoadingState, UiBadge, UiButton } from '@/components/ui'

const auth = useAuthStore()
const projects = ref([])
const loading = ref(true)

onMounted(async () => {
  try { const { data } = await client.get('/projects'); projects.value = data.projects || [] }
  catch {} finally { loading.value = false }
})

const stats = [
  { label: 'Projects', icon: FolderKanban, color: 'oklch(0.92 0.06 265)', iconColor: 'var(--color-primary)',   value: () => projects.value.length },
  { label: 'Team Members',  icon: Users,         color: 'oklch(0.92 0.06 160)', iconColor: 'var(--color-accent)',   value: () => '—' },
  { label: 'API Calls (24h)', icon: Activity,    color: 'oklch(0.92 0.06 70)',  iconColor: 'var(--color-warning)', value: () => '—' },
]

function statusVariant(s) {
  return s === 'active' ? 'success' : s === 'paused' ? 'warning' : 'default'
}
</script>

<template>
  <div class="dashboard animate-fadeIn">
    <div class="page-header">
      <div>
        <h1 class="page-title">Dashboard</h1>
        <p class="page-subtitle">Welcome back, {{ auth.user?.email }}</p>
      </div>
    </div>

    <!-- Stats -->
    <div class="stats-grid">
      <UiCard v-for="s in stats" :key="s.label" padding="md" :hover="true">
        <div class="stat-inner">
          <div class="stat-icon" :style="{ background: s.color }">
            <component :is="s.icon" :size="20" :style="{ color: s.iconColor }" />
          </div>
          <div>
            <div class="stat-value">{{ s.value() }}</div>
            <div class="stat-label">{{ s.label }}</div>
          </div>
        </div>
      </UiCard>
    </div>

    <!-- Recent Projects -->
    <section class="section">
      <div class="section-header">
        <h2 class="section-title">Recent Projects</h2>
        <router-link to="/projects" class="section-link">
          View all <ArrowUpRight :size="14" />
        </router-link>
      </div>

      <UiLoadingState v-if="loading" mode="skeleton" :lines="3" />

      <UiCard v-else-if="projects.length === 0" padding="none">
        <UiEmptyState
          :icon="FolderKanban"
          title="No projects yet"
          description="Create your first project to get started."
          action-label="Create Project"
          @action="$router.push('/projects')"
        />
      </UiCard>

      <div v-else class="projects-grid">
        <UiCard
          v-for="p in projects.slice(0, 6)"
          :key="p.id"
          padding="md"
          :hover="true"
          class="project-card"
          @click="$router.push('/projects')"
        >
          <div class="project-name">{{ p.name }}</div>
          <div class="project-meta">
            <UiBadge :variant="statusVariant(p.status)" dot size="sm">{{ p.status }}</UiBadge>
            <span class="project-date">{{ new Date(p.created_at).toLocaleDateString() }}</span>
          </div>
        </UiCard>
      </div>
    </section>
  </div>
</template>

<style scoped>
.dashboard { max-width: 100%; }
.page-header { margin-bottom: 2rem; }
.page-title { font-size: 1.5rem; font-weight: 700; letter-spacing: -0.02em; }
.page-subtitle { color: var(--color-text-secondary); font-size: 0.9375rem; margin-top: 0.25rem; }

.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 16px; margin-bottom: 2rem; }
.stat-inner { display: flex; align-items: center; gap: 16px; }
.stat-icon { width: 44px; height: 44px; border-radius: var(--radius-md); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.stat-value { font-size: 1.5rem; font-weight: 700; letter-spacing: -0.02em; }
.stat-label { font-size: 0.8125rem; color: var(--color-text-secondary); }

.section { margin-bottom: 2rem; }
.section-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 1rem; }
.section-title { font-size: 1.125rem; font-weight: 600; }
.section-link { font-size: 0.8125rem; color: var(--color-primary); font-weight: 500; display: flex; align-items: center; gap: 4px; }

.projects-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px; }
.project-card { cursor: pointer; }
.project-name { font-weight: 600; margin-bottom: 0.5rem; }
.project-meta { display: flex; align-items: center; justify-content: space-between; }
.project-date { font-size: 0.75rem; color: var(--color-text-tertiary); }
</style>
