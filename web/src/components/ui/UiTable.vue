<script setup>
import { ref, computed } from 'vue'
import { ChevronUp, ChevronDown } from 'lucide-vue-next'

const props = defineProps({
  columns: { type: Array, default: () => [] }, // [{ key, label, sortable, align }]
  rows: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  pageSize: { type: Number, default: 0 }, // 0 = no pagination
  emptyText: { type: String, default: 'No data found.' },
  rowKey: { type: String, default: 'id' },
})
const emit = defineEmits(['row-click'])

const sortKey = ref('')
const sortDir = ref('asc') // asc | desc
const page = ref(1)

const sorted = computed(() => {
  if (!sortKey.value) return props.rows
  return [...props.rows].sort((a, b) => {
    const va = a[sortKey.value] ?? ''
    const vb = b[sortKey.value] ?? ''
    const cmp = String(va).localeCompare(String(vb), undefined, { numeric: true })
    return sortDir.value === 'asc' ? cmp : -cmp
  })
})

const paginated = computed(() => {
  if (!props.pageSize) return sorted.value
  const start = (page.value - 1) * props.pageSize
  return sorted.value.slice(start, start + props.pageSize)
})

const totalPages = computed(() => props.pageSize ? Math.max(1, Math.ceil(props.rows.length / props.pageSize)) : 1)

function toggleSort(key) {
  if (sortKey.value === key) sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  else { sortKey.value = key; sortDir.value = 'asc' }
}

function align(col) { return col.align || 'left' }
</script>

<template>
  <div class="ui-table__container">
    <table class="ui-table">
      <thead>
        <tr>
          <th
            v-for="col in columns" :key="col.key"
            :style="{ textAlign: align(col) }"
            :class="{ 'ui-table__th--sortable': col.sortable }"
            @click="col.sortable && toggleSort(col.key)"
          >
            {{ col.label }}
            <span v-if="col.sortable" class="ui-table__sort-icon">
              <ChevronUp :size="12" :class="{ active: sortKey === col.key && sortDir === 'asc' }" />
              <ChevronDown :size="12" :class="{ active: sortKey === col.key && sortDir === 'desc' }" />
            </span>
          </th>
        </tr>
      </thead>
      <tbody>
        <template v-if="loading">
          <tr v-for="i in (pageSize || 5)" :key="i">
            <td v-for="col in columns" :key="col.key">
              <div class="ui-table__skeleton" />
            </td>
          </tr>
        </template>
        <template v-else-if="paginated.length === 0">
          <tr>
            <td :colspan="columns.length" class="ui-table__empty">{{ emptyText }}</td>
          </tr>
        </template>
        <template v-else>
          <tr
            v-for="row in paginated"
            :key="row[rowKey]"
            class="ui-table__row"
            :class="{ 'ui-table__row--clickable': $attrs.onRowClick }"
            @click="emit('row-click', row)"
          >
            <td v-for="col in columns" :key="col.key" :style="{ textAlign: align(col) }">
              <slot :name="`cell-${col.key}`" :row="row" :value="row[col.key]">
                {{ row[col.key] ?? '—' }}
              </slot>
            </td>
          </tr>
        </template>
      </tbody>
    </table>

    <!-- Pagination -->
    <div v-if="pageSize && totalPages > 1" class="ui-table__pagination">
      <span class="ui-table__page-info">Page {{ page }} of {{ totalPages }}</span>
      <div class="ui-table__page-controls">
        <button class="ui-table__page-btn" :disabled="page <= 1" @click="page--">←</button>
        <button class="ui-table__page-btn" :disabled="page >= totalPages" @click="page++">→</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ui-table__container {
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--color-bg-elevated);
}
.ui-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}
thead { background: var(--color-bg-muted); }
th {
  padding: 10px 16px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  white-space: nowrap;
  user-select: none;
}
.ui-table__th--sortable { cursor: pointer; }
.ui-table__th--sortable:hover { color: var(--color-text); }

.ui-table__sort-icon {
  display: inline-flex;
  flex-direction: column;
  margin-left: 4px;
  vertical-align: middle;
  color: var(--color-text-tertiary);
}
.ui-table__sort-icon .active { color: var(--color-primary); }

td {
  padding: 12px 16px;
  color: var(--color-text);
  border-top: 1px solid var(--color-border-subtle);
}
.ui-table__row { transition: background var(--transition-fast); }
.ui-table__row:hover { background: var(--color-bg-muted); }
.ui-table__row--clickable { cursor: pointer; }

.ui-table__empty {
  text-align: center;
  padding: 3rem;
  color: var(--color-text-tertiary);
}

.ui-table__skeleton {
  height: 14px;
  border-radius: var(--radius-sm);
  background: linear-gradient(90deg, var(--color-bg-muted) 25%, var(--color-bg) 50%, var(--color-bg-muted) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}
@keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }

.ui-table__pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-top: 1px solid var(--color-border-subtle);
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}
.ui-table__page-controls { display: flex; gap: 4px; }
.ui-table__page-btn {
  padding: 4px 10px;
  background: var(--color-bg-muted);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 0.8125rem;
  cursor: pointer;
  transition: all var(--transition-fast);
}
.ui-table__page-btn:hover:not(:disabled) { border-color: var(--color-primary); color: var(--color-primary); }
.ui-table__page-btn:disabled { opacity: 0.4; cursor: not-allowed; }
</style>
