<script setup>
import { ref, onMounted } from 'vue'
import { Plus, Trash2, FolderKanban } from 'lucide-vue-next'
import client from '@/api/client'
import {
  UiButton, UiCard, UiInput, UiField, UiBadge,
  UiTable, UiEmptyState, UiLoadingState, UiConfirmDialog, UiIconButton
} from '@/components/ui'

const projects  = ref([])
const loading   = ref(true)
const showCreate = ref(false)
const newName   = ref('')
const creating  = ref(false)
const deleteTarget = ref(null) // project to delete
const deleting  = ref(false)

const columns = [
  { key: 'name',       label: 'Name',    sortable: true },
  { key: 'status',     label: 'Status',  sortable: true },
  { key: 'created_at', label: 'Created', sortable: true },
  { key: '_actions',   label: '' },
]

async function fetchProjects() {
  loading.value = true
  try { const { data } = await client.get('/projects'); projects.value = data.projects || [] }
  catch {} finally { loading.value = false }
}

async function createProject() {
  if (!newName.value.trim()) return
  creating.value = true
  try {
    await client.post('/projects', { name: newName.value })
    newName.value = ''; showCreate.value = false
    await fetchProjects()
  } catch {} finally { creating.value = false }
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  try { await client.delete(`/projects/${deleteTarget.value.id}`); await fetchProjects() }
  catch {} finally { deleting.value = false; deleteTarget.value = null }
}

function statusVariant(s) {
  return s === 'active' ? 'success' : s === 'paused' ? 'warning' : 'default'
}

onMounted(fetchProjects)
</script>

<template>
  <div class="animate-fadeIn">
    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">Projects</h1>
        <p class="page-subtitle">Manage your workspaces</p>
      </div>
      <UiButton variant="primary" @click="showCreate = true">
        <Plus :size="16" /> New Project
      </UiButton>
    </div>

    <!-- Create form -->
    <UiCard v-if="showCreate" padding="md" style="margin-bottom:1.5rem">
      <form @submit.prevent="createProject" class="create-form">
        <UiField label="Project Name" html-for="new-proj-name" :required="true" style="flex:1;margin-bottom:0">
          <UiInput id="new-proj-name" v-model="newName" placeholder="My Project" required autofocus />
        </UiField>
        <UiButton type="submit" variant="primary" :loading="creating">{{ creating ? 'Creating…' : 'Create' }}</UiButton>
        <UiButton type="button" variant="ghost" @click="showCreate = false">Cancel</UiButton>
      </form>
    </UiCard>

    <!-- Loading -->
    <UiLoadingState v-if="loading" mode="skeleton" :lines="4" />

    <!-- Empty -->
    <UiCard v-else-if="projects.length === 0" padding="none">
      <UiEmptyState
        :icon="FolderKanban"
        title="No projects yet"
        description="Create your first project to get started building."
        action-label="New Project"
        @action="showCreate = true"
      />
    </UiCard>

    <!-- Table -->
    <UiTable
      v-else
      :columns="columns"
      :rows="projects"
      :page-size="20"
      empty-text="No projects found."
    >
      <template #cell-status="{ value }">
        <UiBadge :variant="statusVariant(value)" dot>{{ value }}</UiBadge>
      </template>
      <template #cell-created_at="{ value }">
        {{ value ? new Date(value).toLocaleDateString() : '—' }}
      </template>
      <template #cell-_actions="{ row }">
        <UiIconButton :icon="Trash2" variant="danger" label="Delete project" @click.stop="deleteTarget = row" />
      </template>
    </UiTable>

    <!-- Confirm delete dialog -->
    <UiConfirmDialog
      :open="!!deleteTarget"
      title="Delete project?"
      :message="`'${deleteTarget?.name}' will be permanently deleted. This cannot be undone.`"
      confirm-label="Delete"
      variant="danger"
      @confirm="confirmDelete"
      @cancel="deleteTarget = null"
      @update:open="v => { if (!v) deleteTarget = null }"
    />
  </div>
</template>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 1.5rem; }
.page-title { font-size: 1.5rem; font-weight: 700; letter-spacing: -0.02em; }
.page-subtitle { color: var(--color-text-secondary); font-size: 0.9375rem; margin-top: 0.25rem; }
.create-form { display: flex; gap: 12px; align-items: flex-end; }
</style>
