<template>
  <section class="card overflow-hidden" aria-labelledby="manual-quality-title">
    <header class="flex flex-wrap items-start justify-between gap-3 border-b p-5 dark:border-dark-600">
      <div>
        <h2 id="manual-quality-title" class="text-lg font-semibold">{{ t('qualityOps.manual.title') }}</h2>
        <p class="mt-1 max-w-3xl text-sm text-gray-500">{{ t('qualityOps.manual.description') }}</p>
      </div>
      <button type="button" class="btn btn-secondary" data-testid="manual-close" @click="close">{{ t('qualityOps.manual.close') }}</button>
    </header>
    <form class="space-y-4 p-5" @submit.prevent="start">
      <fieldset :disabled="running" class="space-y-4 disabled:opacity-70">
        <div>
          <label for="manual-account-search" class="mb-2 block text-sm font-medium">{{ t('qualityOps.accounts') }}</label>
          <div class="flex gap-2">
            <input id="manual-account-search" v-model="search" class="input" :placeholder="t('qualityOps.search')" @keydown.enter.prevent="loadAccounts(1)" />
            <button type="button" class="btn btn-secondary shrink-0" :disabled="loadingAccounts" data-testid="manual-search" @click="loadAccounts(1)">{{ t('qualityOps.search') }}</button>
          </div>
          <div class="my-2 flex flex-wrap items-center gap-3 text-sm">
            <button type="button" class="text-primary-600" :disabled="loadingAccounts || !accounts.length" @click="selectPage">{{ t('qualityOps.manual.selectPage') }}</button>
            <button type="button" class="text-gray-500" :disabled="!selected.size" @click="selected.clear()">{{ t('qualityOps.manual.clear') }}</button>
            <span>{{ t('qualityOps.selected', { count: selected.size }) }}</span>
          </div>
          <div class="grid max-h-48 gap-2 overflow-auto rounded-lg border p-3 sm:grid-cols-2 xl:grid-cols-3 dark:border-dark-600" :aria-busy="loadingAccounts">
            <label v-for="account in accounts" :key="account.id" class="flex items-center gap-2 text-sm">
              <input type="checkbox" :checked="selected.has(account.id)" :value="account.id" data-testid="manual-account" @change="toggleAccount(account)" />
              <span class="min-w-0 truncate" :title="account.name">{{ account.name }}</span><span class="shrink-0 text-xs text-gray-500">#{{ account.id }} · {{ account.type }}</span>
            </label>
            <p v-if="!accounts.length" class="text-sm text-gray-500">{{ t(loadingAccounts ? 'qualityOps.manual.loading' : 'qualityOps.manual.noAccounts') }}</p>
          </div>
          <div class="mt-2 flex items-center gap-3 text-sm">
            <button type="button" class="btn btn-secondary" :disabled="loadingAccounts || page <= 1" :aria-label="t('qualityOps.manual.previous')" data-testid="manual-previous" @click="loadAccounts(page - 1)">←</button>
            <span>{{ page }} / {{ pages }}</span>
            <button type="button" class="btn btn-secondary" :disabled="loadingAccounts || page >= pages" :aria-label="t('qualityOps.manual.next')" data-testid="manual-next" @click="loadAccounts(page + 1)">→</button>
          </div>
        </div>
        <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <label class="space-y-1"><span class="text-sm">{{ t('qualityOps.model') }}</span><input v-model="model" class="input" required maxlength="100" data-testid="manual-model" placeholder="gpt-6-astra" /></label>
          <label class="space-y-1"><span class="text-sm">{{ t('qualityOps.effort') }}</span><select v-model="effort" class="input" data-testid="manual-effort"><option v-for="value in ['minimal', 'low', 'medium', 'high', 'xhigh']" :key="value" :value="value">{{ value }}</option></select></label>
          <label class="space-y-1"><span class="text-sm">{{ t('qualityOps.manual.concurrency') }}</span><input v-model.number="concurrency" type="number" class="input" min="1" max="8" required data-testid="manual-concurrency" /></label>
          <label class="space-y-1"><span class="text-sm">{{ t('qualityOps.manual.preset') }}</span><select :value="question" class="input" data-testid="manual-preset" @change="changeQuestion"><option value="pelican">{{ t('qualityOps.manual.pelican') }}</option><option value="candy">{{ t('qualityOps.manual.candy') }}</option><option value="custom">{{ t('qualityOps.manual.custom') }}</option></select></label>
        </div>
        <label class="block space-y-1"><span class="text-sm">{{ t('qualityOps.prompt') }}</span><textarea v-model="prompt" required :maxlength="32000 - deliveryContract.length - (deliveryContract ? 2 : 0)" rows="4" class="input font-mono text-sm" data-testid="manual-prompt" /></label>
        <p v-if="isCandyReference" class="text-sm text-gray-500">{{ t('qualityOps.manual.candyReference') }}</p>
      </fieldset>
      <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
      <div class="flex flex-wrap items-center gap-3">
        <button type="submit" class="btn btn-primary" :disabled="!canStart" data-testid="manual-start">{{ t('qualityOps.manual.start') }}</button>
        <button type="button" class="btn btn-secondary" :disabled="!running" data-testid="manual-stop" @click="stop">{{ t('qualityOps.manual.stop') }}</button>
        <span v-if="runs.length" class="text-sm text-gray-500" role="status">{{ t('qualityOps.manual.progress', { completed, total: runs.length }) }}</span>
      </div>
    </form>
    <div v-if="runs.length" class="border-t bg-gray-50/50 p-5 dark:border-dark-600 dark:bg-dark-900/30">
      <p class="mb-4 text-sm text-gray-500">{{ t('qualityOps.manual.compareHelp') }}</p>
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
        <article v-for="run in runs" :key="run.accountId" class="min-w-0 overflow-hidden rounded-xl border bg-white dark:border-dark-600 dark:bg-dark-800" data-testid="manual-result" :data-status="run.status">
          <div class="border-b px-4 py-3 dark:border-dark-600">
            <h3 class="truncate font-semibold" :title="run.accountName">{{ run.accountName }}</h3>
            <p class="mt-1 text-xs text-gray-500">#{{ run.accountId }} · {{ run.model }} / {{ run.effort }}</p>
          </div>
          <div class="p-3">
            <div class="mb-3 flex flex-wrap items-center justify-between gap-2 text-sm">
              <span :class="run.status === 'error' ? 'text-red-600' : run.status === 'success' ? 'text-emerald-600' : 'text-gray-500'">{{ t(`qualityOps.manual.states.${run.status}`) }}</span>
              <span class="text-gray-500">{{ duration(run) }} s</span>
            </div>
            <div v-if="run.html" class="aspect-[4/3] overflow-hidden rounded-lg border bg-white dark:border-dark-600">
              <iframe :srcdoc="run.html" sandbox="allow-scripts" referrerpolicy="no-referrer" class="h-full w-full border-0" :title="t('qualityOps.manual.preview', { account: run.accountName })" />
            </div>
            <p v-if="run.error" class="mb-2 whitespace-pre-wrap break-words text-sm text-red-600" role="alert">{{ run.error }}</p>
            <p v-if="run.status === 'success' && run.question === 'pelican' && !run.html" class="mb-2 text-sm text-amber-600">{{ t('qualityOps.manual.noPreview') }}</p>
            <p v-if="run.candyReference" class="mt-2 text-xs text-gray-500">{{ t('qualityOps.manual.candyReference') }}</p>
            <details v-if="run.output" class="mt-3" :open="!run.html">
              <summary class="cursor-pointer text-sm text-primary-600">{{ t('qualityOps.manual.raw') }}</summary>
              <pre class="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-gray-950 p-3 text-xs text-gray-100">{{ run.output }}</pre>
            </details>
          </div>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { list } from '@/api/admin/accounts'
import { ManualQualityError, runManualQualityProbe, type ManualQualityRequest } from '@/api/admin/manualQuality'
import { CANDY_PROMPT, questionContract, questionPrompt, type IntelligenceQuestion } from '@/utils/intelligenceTest'
import { extractPelicanHtml } from '@/utils/pelicanHtml'
import type { AccountListItem } from '@/types'

type Question = IntelligenceQuestion | 'custom'
type RunStatus = 'pending' | 'running' | 'success' | 'error' | 'cancelled'
interface AccountSelection { id: number; name: string }
interface Run {
  accountId: number; accountName: string; model: string; effort: string; question: Question
  candyReference: boolean; status: RunStatus; output: string; html: string; error: string
  startedAt: number | null; durationMs: number
}

const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const search = ref(''), accounts = ref<AccountListItem[]>([]), selected = ref(new Map<number, AccountSelection>())
const page = ref(1), pages = ref(1), loadingAccounts = ref(false), error = ref('')
const model = ref('gpt-6-astra'), effort = ref('high'), concurrency = ref(3)
const question = ref<Question>('pelican'), prompt = ref(questionPrompt('pelican'))
const runs = ref<Run[]>([]), running = ref(false), now = ref(Date.now())
const controllers = new Map<number, AbortController>()
let batch = 0, searchRequest = 0, accountController: AbortController | undefined
let timer: ReturnType<typeof setInterval> | undefined
const isCandyReference = computed(() => question.value === 'candy' && prompt.value.trim() === CANDY_PROMPT)
const deliveryContract = computed(() => question.value === 'custom' ? '' : questionContract(question.value))
const canStart = computed(() => !running.value && selected.value.size > 0 && !!model.value.trim() && !!prompt.value.trim() && prompt.value.length + deliveryContract.value.length + (deliveryContract.value ? 2 : 0) <= 32000 && Number.isInteger(concurrency.value) && concurrency.value >= 1 && concurrency.value <= 8)
const completed = computed(() => runs.value.filter(run => !['pending', 'running'].includes(run.status)).length)

function errorMessage(value: unknown): string {
  if (value instanceof ManualQualityError) return value.message || t(`qualityOps.manual.errors.${value.code}`)
  if (value instanceof Error) return value.message
  if (value && typeof value === 'object' && 'message' in value && typeof value.message === 'string') return value.message
  return t('qualityOps.error')
}
async function loadAccounts(nextPage = 1) {
  const request = ++searchRequest
  accountController?.abort()
  accountController = new AbortController()
  loadingAccounts.value = true
  error.value = ''
  try {
    const result = await list(nextPage, 50, { platform: 'openai', search: search.value.trim(), lite: 'true' }, { signal: accountController.signal })
    if (request !== searchRequest) return
    accounts.value = result.items.filter(account => account.platform === 'openai')
    page.value = nextPage
    pages.value = Math.max(1, Math.ceil(result.total / 50))
  } catch (value) {
    if (request === searchRequest && !accountController.signal.aborted) error.value = errorMessage(value)
  } finally {
    if (request === searchRequest) loadingAccounts.value = false
  }
}
function toggleAccount(account: AccountSelection) {
  if (selected.value.has(account.id)) selected.value.delete(account.id)
  else selected.value.set(account.id, { id: account.id, name: account.name })
}
function selectPage() { for (const account of accounts.value) selected.value.set(account.id, { id: account.id, name: account.name }) }
function changeQuestion(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  if (value !== 'pelican' && value !== 'candy' && value !== 'custom') return
  question.value = value
  if (value !== 'custom') prompt.value = questionPrompt(value)
}
function clearTimer() { if (timer) clearInterval(timer); timer = undefined }
function duration(run: Run) { return ((run.status === 'running' && run.startedAt !== null ? now.value - run.startedAt : run.durationMs) / 1000).toFixed(1) }
function stop() {
  ++batch
  for (const controller of controllers.values()) controller.abort()
  controllers.clear()
  for (const run of runs.value) {
    if (run.status !== 'pending' && run.status !== 'running') continue
    if (run.startedAt !== null) run.durationMs = Math.max(0, Date.now() - run.startedAt)
    run.html = extractPelicanHtml(run.output)
    run.status = 'cancelled'
  }
  running.value = false
  clearTimer()
}
function close() { stop(); emit('close') }
async function start() {
  if (!canStart.value) return
  const generation = ++batch
  const kind = question.value
  const contract = deliveryContract.value
  const request: Readonly<ManualQualityRequest> = Object.freeze({ model_id: model.value.trim(), prompt: `${prompt.value.trim()}${contract ? `\n\n${contract}` : ''}`, reasoning_effort: effort.value })
  const batchRuns: Run[] = [...selected.value.values()].map(account => ({
    accountId: account.id, accountName: account.name, model: request.model_id, effort: request.reasoning_effort,
    question: kind, candyReference: isCandyReference.value, status: 'pending', output: '', html: '', error: '', startedAt: null, durationMs: 0
  }))
  runs.value = batchRuns
  // Use Vue's reactive run instances for async progress updates.
  const activeRuns = runs.value
  running.value = true
  error.value = ''
  now.value = Date.now()
  timer = setInterval(() => { now.value = Date.now() }, 250)
  let next = 0
  async function worker() {
    while (generation === batch && next < activeRuns.length) {
      const run = activeRuns[next++]!
      const controller = new AbortController()
      controllers.set(run.accountId, controller)
      run.status = 'running'
      run.startedAt = Date.now()
      try {
        await runManualQualityProbe(run.accountId, request, controller.signal, text => {
          if (generation === batch) run.output += text
        })
        if (generation === batch) run.status = 'success'
      } catch (value) {
        if (generation === batch) {
          run.status = controller.signal.aborted ? 'cancelled' : 'error'
          if (!controller.signal.aborted) run.error = errorMessage(value)
        }
      } finally {
        if (generation === batch) {
          run.durationMs = Math.max(0, Date.now() - run.startedAt!)
          run.html = extractPelicanHtml(run.output)
          controllers.delete(run.accountId)
        }
      }
    }
  }
  await Promise.all(Array.from({ length: Math.min(concurrency.value, activeRuns.length) }, worker))
  if (generation === batch) { running.value = false; clearTimer() }
}
onMounted(() => { void loadAccounts() })
onBeforeUnmount(() => { stop(); ++searchRequest; accountController?.abort() })
</script>
