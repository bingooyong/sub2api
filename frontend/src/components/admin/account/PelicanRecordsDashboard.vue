<template>
  <div class="fixed inset-0 z-[60] overflow-y-auto bg-gray-50 p-4 dark:bg-dark-950" role="dialog" aria-modal="true">
    <div class="mx-auto max-w-7xl">
      <header class="mb-4 flex items-center justify-between gap-3">
        <div>
          <h2 class="text-xl font-semibold">{{ t('admin.accounts.pelicanTest.history') }}</h2>
          <p class="text-sm text-gray-500">{{ t('admin.accounts.pelicanTest.dashboardHint') }}</p>
        </div>
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.close') }}</button>
      </header>
      <p v-if="error" role="alert" class="mb-4 text-sm text-red-600">{{ error }}</p>
      <div v-if="loading" class="py-16 text-center text-sm text-gray-500">{{ t('common.loading') }}...</div>
      <div v-else-if="cards.length" class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <article v-for="card in cards" :key="card.id" class="overflow-hidden rounded-xl border bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="p-4">
            <h3 class="font-semibold">{{ card.accountName }}</h3>
            <p class="mt-1 text-xs text-gray-500">#{{ card.accountId }} · {{ formatDate(card.startedAt) }}</p>
            <span :class="card.status === 'success' ? 'text-emerald-600' : 'text-red-500'">{{ t(card.status === 'success' ? 'admin.accounts.pelicanTest.success' : 'admin.accounts.pelicanTest.failed') }}</span>
            <div class="mt-3 aspect-[4/3] overflow-auto rounded bg-gray-50 dark:bg-dark-900">
              <iframe v-if="card.html" :srcdoc="card.html" class="h-full w-full border-0" sandbox="allow-scripts" referrerpolicy="no-referrer" :title="card.accountName" />
              <pre v-else class="whitespace-pre-wrap break-words p-3 text-xs">{{ card.output || card.error || '—' }}</pre>
            </div>
          </div>
        </article>
      </div>
      <div v-else class="rounded-xl border border-dashed py-16 text-center text-sm text-gray-500">{{ t('admin.accounts.pelicanTest.noHistory') }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { scheduledTestsAPI, type PelicanHistoryResult } from '@/api/admin/scheduledTests'
import { extractPelicanHtml } from '@/utils/pelicanHtml'
import type { Account, AccountListItem, ScheduledTestResult } from '@/types'

const props = defineProps<{ accounts: AccountListItem[]; account: Account | null; manualRecord: { runs?: Array<{ output?: string; error?: string; status?: string; startedAt?: string; questionKind?: string }> } | null }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const loading = ref(true)
const error = ref('')
const cards = ref<Array<{ id: string; accountId: number; accountName: string; status: string; output: string; html: string; error: string; startedAt?: string }>>([])

function formatDate(value?: string) {
  if (!value) return '—'
  const timestamp = Date.parse(value)
  return Number.isFinite(timestamp) ? new Date(timestamp).toLocaleString() : '—'
}
function addServer(result: ScheduledTestResult & Partial<PelicanHistoryResult>) {
  const output = result.response_text || ''
  cards.value.push({ id: `server-${result.id}`, accountId: result.account_id || props.account?.id || 0, accountName: result.account_name || props.account?.name || '—', status: result.status, output, html: result.pelican_config?.question_kind === 'candy' ? '' : extractPelicanHtml(output), error: result.error_message, startedAt: result.started_at })
}
async function load() {
  try {
    const page = await scheduledTestsAPI.listPelicanHistory()
    cards.value = []
    for (const result of page.items) addServer(result)
    for (const [index, run] of (props.manualRecord?.runs || []).entries()) {
      const output = run.output || ''
      cards.value.push({ id: `manual-${index}`, accountId: props.account?.id || 0, accountName: props.account?.name || '—', status: run.status || 'error', output, html: run.questionKind === 'candy' ? '' : extractPelicanHtml(output), error: run.error || '', startedAt: run.startedAt })
    }
  } catch {
    error.value = t('admin.accounts.pelicanTest.historyLoadError')
  } finally {
    loading.value = false
  }
}
onMounted(load)
void emit
</script>
