<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Alert as AAlert } from 'ant-design-vue'
import { usePluginHost } from '@rayleabot/plugin-ui'

import defaultConfigSource from '../../internal/assets/fortunes.json'
import ChipEditor from './components/ChipEditor.vue'
import {
  buildSettingsPayload,
  EXPECTED_STARS,
  firstStarsForName,
  FORTUNE_NAMES,
  normalizeSettings,
  stableSettingsJSON,
  TIMEZONE_OPTIONS,
  type FortuneItem,
  type FortuneSettings,
  type SpecialDate,
  validateSettings,
} from './model'

const host = usePluginHost()
const defaultSettings = normalizeSettings(defaultConfigSource as unknown as Record<string, unknown>)
const draft = ref<FortuneSettings>(normalizeSettings({}))
const savedSnapshot = ref(stableSettingsJSON(draft.value))
const pageTitle = ref('运势设置')
const status = ref('正在等待宿主初始化')
const statusIsError = ref(false)
const busy = ref(false)
const loaded = ref(false)
const fortuneFilter = ref('')
const fortuneTypeFilter = ref('')
const currentPage = ref(1)
const pageSize = 10
let rowSequence = 0
const fortuneKeys = new WeakMap<FortuneItem, string>()
const specialDateKeys = new WeakMap<SpecialDate, string>()

const hostErrorMessage = computed(() => host.error.value?.message ?? '')
const validation = computed(() => validateSettings(draft.value))
const isDirty = computed(() => stableSettingsJSON(draft.value) !== savedSnapshot.value)
const fortuneNames = computed(() => [...new Set(draft.value.fortunes.map((item) => item.name).filter(Boolean))])
const filteredFortunes = computed(() => {
  const query = fortuneFilter.value.trim().toLowerCase()
  return draft.value.fortunes
    .map((fortune, originalIndex) => ({ fortune, originalIndex }))
    .filter(({ fortune }) => {
      if (fortuneTypeFilter.value && fortune.name !== fortuneTypeFilter.value) return false
      const searchable = `${fortune.name} ${fortune.sign} ${fortune.explanation}`.toLowerCase()
      return !query || searchable.includes(query)
    })
})
const totalPages = computed(() => Math.max(1, Math.ceil(filteredFortunes.value.length / pageSize)))
const pageStart = computed(() => (currentPage.value - 1) * pageSize)
const pageEnd = computed(() => Math.min(pageStart.value + pageSize, filteredFortunes.value.length))
const pageFortunes = computed(() => filteredFortunes.value.slice(pageStart.value, pageEnd.value))
const validationSummary = computed(() => validation.value.length === 0 ? '可保存' : `${validation.value.length} 个问题`)
const dirtyStateText = computed(() => {
  if (!loaded.value) return '等待载入'
  if (busy.value) return '正在处理'
  if (validation.value.length > 0) return '存在未修正问题'
  return isDirty.value ? '有未保存更改' : '设置已同步'
})

watch([fortuneFilter, fortuneTypeFilter], () => { currentPage.value = 1 })
watch(totalPages, (pages) => {
  currentPage.value = Math.min(Math.max(currentPage.value, 1), pages)
})

void host.ready
  .then((init) => {
    pageTitle.value = init.page.label || '运势设置'
    applySettings(init.config, true)
    setStatus('已载入设置')
  })
  .catch((error: unknown) => {
    setStatus(errorMessage(error, '插件页面连接失败'), true)
  })

function applySettings(values: Record<string, unknown>, markSaved: boolean) {
  draft.value = normalizeSettings(values, defaultSettings)
  if (markSaved) savedSnapshot.value = stableSettingsJSON(draft.value)
  loaded.value = true
  currentPage.value = 1
}

function setStatus(message: string, isError = false) {
  status.value = message
  statusIsError.value = isError
}

async function reload() {
  busy.value = true
  setStatus('正在重新读取设置')
  try {
    const result = await host.client.reloadSettings()
    applySettings(result.config, true)
    setStatus('已重新读取设置')
  } catch (error) {
    setStatus(errorMessage(error, '重新读取设置失败'), true)
  } finally {
    busy.value = false
  }
}

async function save() {
  if (validation.value.length > 0) {
    setStatus(validation.value[0]!.message, true)
    return
  }
  busy.value = true
  setStatus('正在保存设置')
  try {
    const payload = buildSettingsPayload(draft.value)
    const result = await host.client.saveSettings(payload as unknown as Record<string, unknown>)
    applySettings(result.config, true)
    setStatus('设置已保存')
  } catch (error) {
    setStatus(errorMessage(error, '保存设置失败'), true)
  } finally {
    busy.value = false
  }
}

function resetSettings() {
  draft.value = structuredClone(defaultSettings)
  currentPage.value = 1
  setStatus('默认设置已载入，保存后生效')
}

function addFortune() {
  draft.value.fortunes.unshift({
    name: '大吉',
    stars: firstStarsForName('大吉'),
    sign: '',
    explanation: '',
  })
  currentPage.value = 1
}

function duplicateFortune(index: number) {
  draft.value.fortunes.splice(index + 1, 0, { ...draft.value.fortunes[index]! })
}

function removeFortune(index: number) {
  draft.value.fortunes.splice(index, 1)
}

function updateFortuneName(index: number, event: Event) {
  const name = (event.target as HTMLSelectElement).value
  const fortune = draft.value.fortunes[index]
  if (!fortune) return
  fortune.name = name
  fortune.stars = firstStarsForName(name)
}

function availableStars(fortune: FortuneItem): string[] {
  const values = [...(EXPECTED_STARS[fortune.name] ?? [])]
  if (fortune.stars && !values.includes(fortune.stars)) values.push(fortune.stars)
  return values
}

function addSpecialDate() {
  draft.value.special_dates.push({ date: '', fortune_name: draft.value.fortunes[0]?.name ?? '大吉' })
}

function duplicateSpecialDate(index: number) {
  draft.value.special_dates.splice(index + 1, 0, { ...draft.value.special_dates[index]! })
}

function removeSpecialDate(index: number) {
  draft.value.special_dates.splice(index, 1)
}

function issueText(scope: string): string {
  return validation.value.find((issue) => issue.scope === scope)?.message ?? ''
}

function fortuneClass(name: string): string {
  return `fortune-select fortune-select--${name}`
}

function fortuneKey(item: FortuneItem): string {
  let key = fortuneKeys.get(item)
  if (!key) {
    key = `fortune-${++rowSequence}`
    fortuneKeys.set(item, key)
  }
  return key
}

function specialDateKey(item: SpecialDate): string {
  let key = specialDateKeys.get(item)
  if (!key) {
    key = `special-${++rowSequence}`
    specialDateKeys.set(item, key)
  }
  return key
}

function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback
}
</script>

<template>
  <main class="page-shell">
    <h1 class="sr-only">{{ pageTitle }}</h1>

    <AAlert v-if="hostErrorMessage" class="host-alert" type="error" :message="hostErrorMessage" show-icon />

    <section class="overview-grid" aria-label="配置概览">
      <article class="metric">
        <div class="metric-icon" aria-hidden="true">🔮</div>
        <div class="metric-content"><span>运势条目</span><strong>{{ draft.fortunes.length }}</strong></div>
      </article>
      <article class="metric">
        <div class="metric-icon" aria-hidden="true">📅</div>
        <div class="metric-content"><span>特殊日期</span><strong>{{ draft.special_dates.length }}</strong></div>
      </article>
      <article class="metric">
        <div class="metric-icon" aria-hidden="true">🔑</div>
        <div class="metric-content"><span>触发词</span><strong>{{ draft.trigger_commands.length + draft.stats_trigger_commands.length }}</strong></div>
      </article>
      <article class="metric">
        <div class="metric-icon" aria-hidden="true">🌐</div>
        <div class="metric-content"><span>当前时区</span><strong>{{ draft.timezone || 'Asia/Shanghai' }}</strong></div>
      </article>
      <article class="metric">
        <div class="metric-icon" aria-hidden="true">🛡️</div>
        <div class="metric-content"><span>校验状态</span><strong :class="{ 'is-error': validation.length > 0 }">{{ validationSummary }}</strong></div>
      </article>
    </section>

    <div class="dashboard-grid">
      <aside class="config-sidebar">
        <section class="panel">
          <div class="section-title">
            <h2>触发词设置</h2>
            <p>配置用户在群聊或私聊中触发运势查询的关键词。</p>
          </div>
          <ChipEditor
            id="fortune-trigger-input"
            v-model="draft.trigger_commands"
            label="运势触发词"
            placeholder="添加触发词后回车..."
          />
          <ChipEditor
            id="stats-trigger-input"
            v-model="draft.stats_trigger_commands"
            label="统计触发词"
            placeholder="添加统计词后回车..."
          />
        </section>

        <section class="panel">
          <div class="section-title">
            <h2>时区设置</h2>
            <p>配置机器人刷新每日运势数据所采用的参考时区。</p>
          </div>
          <label class="field" for="timezone-input">
            <span>时区名称</span>
            <input id="timezone-input" v-model.trim="draft.timezone" list="timezone-options" type="text" placeholder="Asia/Shanghai" autocomplete="off" />
          </label>
          <datalist id="timezone-options">
            <option v-for="item in TIMEZONE_OPTIONS" :key="item.value" :value="item.value">{{ item.offset }} · {{ item.label }}</option>
          </datalist>
          <div class="timezone-list">
            <button
              v-for="item in TIMEZONE_OPTIONS.slice(0, 6)"
              :key="item.value"
              type="button"
              class="timezone-option"
              :class="{ 'is-active': draft.timezone === item.value }"
              @click="draft.timezone = item.value"
            >
              {{ item.value }} · {{ item.offset }} · {{ item.label }}
            </button>
          </div>
          <small v-if="issueText('timezone')" class="field-error">{{ issueText('timezone') }}</small>
        </section>
      </aside>

      <div class="workspace-content">
        <section class="panel">
          <div class="section-title section-title--inline">
            <div>
              <h2>运势库</h2>
              <p>配置运势库的主签与解签。星级由运势名决定，可直接在表格中编辑内容。</p>
            </div>
            <div class="toolbar">
              <input v-model="fortuneFilter" class="toolbar-input" type="search" placeholder="输入关键字筛选..." aria-label="筛选运势" />
              <select v-model="fortuneTypeFilter" class="toolbar-select" aria-label="按运势类型筛选">
                <option value="">全部运势</option>
                <option v-for="name in FORTUNE_NAMES" :key="name" :value="name">{{ name }}</option>
              </select>
              <button type="button" class="button button--primary-accent" @click="addFortune">新增运势</button>
            </div>
          </div>
          <div class="table-wrap">
            <table class="data-table" aria-label="运势库">
              <thead><tr><th>运势名</th><th>星级</th><th>签文</th><th>解签</th><th>操作</th></tr></thead>
              <tbody>
                <tr
                  v-for="entry in pageFortunes"
                  :key="fortuneKey(entry.fortune)"
                  :class="{ 'has-error': issueText(`fortune-${entry.originalIndex}`) }"
                >
                  <td>
                    <select
                      :class="fortuneClass(entry.fortune.name)"
                      :value="entry.fortune.name"
                      aria-label="运势名"
                      @change="updateFortuneName(entry.originalIndex, $event)"
                    >
                      <option v-for="name in FORTUNE_NAMES" :key="name" :value="name">{{ name }}</option>
                    </select>
                  </td>
                  <td>
                    <select v-model="entry.fortune.stars" aria-label="星级">
                      <option v-for="stars in availableStars(entry.fortune)" :key="stars" :value="stars">{{ stars }}</option>
                    </select>
                  </td>
                  <td><textarea v-model="entry.fortune.sign" rows="3" aria-label="签文" /></td>
                  <td><textarea v-model="entry.fortune.explanation" rows="3" aria-label="解签" /></td>
                  <td>
                    <div class="row-actions">
                      <button type="button" class="button button--small" @click="duplicateFortune(entry.originalIndex)">复制</button>
                      <button type="button" class="button button--small button--danger" @click="removeFortune(entry.originalIndex)">删除</button>
                    </div>
                    <small v-if="issueText(`fortune-${entry.originalIndex}`)" class="field-error">{{ issueText(`fortune-${entry.originalIndex}`) }}</small>
                  </td>
                </tr>
                <tr v-if="pageFortunes.length === 0"><td colspan="5" class="empty-cell">没有符合条件的运势</td></tr>
              </tbody>
            </table>
          </div>
          <div class="pagination">
            <span class="pagination-info">
              {{ filteredFortunes.length === 0 ? '共 0 条运势' : `显示第 ${pageStart + 1} - ${pageEnd} 条，共 ${filteredFortunes.length} 条` }}
            </span>
            <div class="pagination-controls">
              <button type="button" class="button button--small" :disabled="currentPage <= 1" @click="currentPage -= 1">上一页</button>
              <span class="pagination-current">第 {{ currentPage }} / {{ totalPages }} 页</span>
              <button type="button" class="button button--small" :disabled="currentPage >= totalPages" @click="currentPage += 1">下一页</button>
            </div>
          </div>
        </section>

        <section class="panel">
          <div class="section-title section-title--inline">
            <div>
              <h2>特殊日期设定</h2>
              <p>在指定日期（如节日或特殊纪念日）固定抽取的运势结果。</p>
            </div>
            <button type="button" class="button button--primary-accent" @click="addSpecialDate">新增日期</button>
          </div>
          <div class="table-wrap">
            <table class="data-table data-table--compact" aria-label="特殊日期">
              <thead><tr><th>日期</th><th>指定运势</th><th>状态</th><th>操作</th></tr></thead>
              <tbody>
                <tr
                  v-for="(item, index) in draft.special_dates"
                  :key="specialDateKey(item)"
                  :class="{ 'has-error': issueText(`special-${index}`) }"
                >
                  <td><input v-model.trim="item.date" type="text" placeholder="05-04 或 2026-05-04" aria-label="特殊日期" /></td>
                  <td>
                    <select v-model="item.fortune_name" :class="fortuneClass(item.fortune_name)" aria-label="指定运势">
                      <option v-for="name in fortuneNames" :key="name" :value="name">{{ name }}</option>
                    </select>
                  </td>
                  <td><span class="row-status" :class="{ 'is-error': issueText(`special-${index}`) }">{{ issueText(`special-${index}`) || '有效' }}</span></td>
                  <td>
                    <div class="row-actions">
                      <button type="button" class="button button--small" @click="duplicateSpecialDate(index)">复制</button>
                      <button type="button" class="button button--small button--danger" @click="removeSpecialDate(index)">删除</button>
                    </div>
                  </td>
                </tr>
                <tr v-if="draft.special_dates.length === 0"><td colspan="4" class="empty-cell">没有特殊日期</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="panel">
          <div class="section-title">
            <h2>每日宜忌设定</h2>
            <p>设定从每日运势中随机推荐的今日适宜与忌讳活动，支持添加多项内容。</p>
          </div>
          <div class="actions-grid">
            <ChipEditor id="good-actions-input" v-model="draft.good_actions" tone="good_actions" label="今日宜" placeholder="添加适宜活动后回车...">
              <template #icon><span class="action-icon action-icon--good" aria-hidden="true">👍</span></template>
            </ChipEditor>
            <ChipEditor id="bad-actions-input" v-model="draft.bad_actions" tone="bad_actions" label="今日忌" placeholder="添加忌讳活动后回车...">
              <template #icon><span class="action-icon action-icon--bad" aria-hidden="true">👎</span></template>
            </ChipEditor>
          </div>
        </section>
      </div>
    </div>

    <footer class="actions">
      <span
        class="dirty-state"
        :class="{
          'is-error': statusIsError || validation.length > 0,
          'is-dirty': !statusIsError && isDirty && validation.length === 0,
          'is-synced': !statusIsError && loaded && !isDirty && validation.length === 0,
        }"
        aria-live="polite"
      >{{ statusIsError ? status : dirtyStateText }}</span>
      <div class="footer-buttons">
        <button type="button" class="button" :disabled="busy" @click="reload">重新载入</button>
        <button type="button" class="button" :disabled="busy" @click="resetSettings">恢复默认</button>
        <button type="button" class="button button--primary" :disabled="busy || validation.length > 0 || !isDirty" @click="save">保存设置</button>
      </div>
    </footer>
  </main>
</template>
