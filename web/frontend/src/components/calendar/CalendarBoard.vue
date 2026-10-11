<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowLeft,
  ArrowRight,
  Calendar,
  Check,
  CircleCheck,
  Clock,
  List,
  Refresh,
  Timer,
  WarningFilled,
} from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import {
  completePlan,
  completeSchedule,
  completeTask,
  getCalendarEvents,
} from '../../api'

const router = useRouter()

const weekdayLabels = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
const sourceOptions = [
  { value: 'all', label: '全部安排' },
  { value: 'task', label: '任务' },
  { value: 'doubao_schedule', label: '豆包日程' },
  { value: 'personal_plan', label: '个人计划' },
]

const viewDate = ref(startOfMonth(new Date()))
const selectedDate = ref(localDateValue(new Date()))
const sourceFilter = ref('all')
const showCompleted = ref(false)
const response = ref(createEmptyResponse())
const loading = ref(true)
const loadError = ref('')
const completingKey = ref('')

const today = computed(() => response.value.today || localDateValue(new Date()))
const monthLabel = computed(() => `${viewDate.value.getFullYear()}年${viewDate.value.getMonth() + 1}月`)
const gridStart = computed(() => startOfCalendarGrid(viewDate.value))
const gridEnd = computed(() => addDays(gridStart.value, 41))
const gridRangeLabel = computed(() => `${formatShortDate(localDateValue(gridStart.value))} — ${formatShortDate(localDateValue(gridEnd.value))}`)
const summary = computed(() => response.value.summary || createEmptyResponse().summary)
const eventMap = computed(() => {
  const map = new Map()
  for (const event of response.value.events || []) {
    const current = map.get(event.date) || []
    current.push(event)
    map.set(event.date, current)
  }
  return map
})
const calendarDays = computed(() => Array.from({ length: 42 }, (_, index) => {
  const date = addDays(gridStart.value, index)
  const value = localDateValue(date)
  return {
    date,
    value,
    day: date.getDate(),
    isCurrentMonth: date.getMonth() === viewDate.value.getMonth(),
    isToday: value === today.value,
    isSelected: value === selectedDate.value,
  }
}))
const selectedEvents = computed(() => eventMap.value.get(selectedDate.value) || [])
const selectedDateLabel = computed(() => formatLongDate(selectedDate.value))
const sourceValues = computed(() => (sourceFilter.value === 'all' ? [] : [sourceFilter.value]))

function createEmptyResponse() {
  return {
    today: '',
    start: '',
    end: '',
    events: [],
    summary: {
      total: 0,
      today: 0,
      overdue: 0,
      tasks: 0,
      doubao_schedules: 0,
      personal_plans: 0,
    },
  }
}

function localDateValue(value) {
  const date = new Date(value)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function parseLocalDate(value) {
  const [year, month, day] = value.split('-').map(Number)
  return new Date(year, month - 1, day)
}

function startOfMonth(value) {
  const date = new Date(value)
  return new Date(date.getFullYear(), date.getMonth(), 1)
}

function startOfCalendarGrid(value) {
  const first = startOfMonth(value)
  const offset = first.getDay() === 0 ? 6 : first.getDay() - 1
  return addDays(first, -offset)
}

function addDays(value, days) {
  const date = new Date(value)
  date.setDate(date.getDate() + days)
  return date
}

function formatShortDate(value) {
  const [month, day] = value.split('-').slice(1).map(Number)
  return `${month}月${day}日`
}

function formatLongDate(value) {
  const date = parseLocalDate(value)
  return `${date.getFullYear()}年${date.getMonth() + 1}月${date.getDate()}日 ${weekdayLabels[date.getDay()]}`
}

function formatEventTime(value) {
  if (!value) return '全天'
  const date = new Date(value)
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

function formatInterval(event) {
  if (!event.interval_days) return ''
  return `每 ${event.interval_days} 天`
}

function formatEventMeta(event) {
  if (event.source === 'task') {
    return event.priority_label || '任务'
  }
  if (event.status === 'overdue') {
    return `原提醒日 ${formatShortDate(event.original_date || event.next_due_date)}`
  }
  return [formatInterval(event), event.next_due_date ? `提醒 ${formatShortDate(event.next_due_date)}` : ''].filter(Boolean).join(' · ')
}

function eventsForDate(value) {
  return eventMap.value.get(value) || []
}

function visibleEventsForDate(value) {
  return eventsForDate(value).slice(0, 3)
}

function eventOverflow(value) {
  return Math.max(eventsForDate(value).length - 3, 0)
}

function sourceClass(source) {
  return `is-${source.replace('_', '-')}`
}

function sourceIcon(source) {
  if (source === 'task') return List
  if (source === 'personal_plan') return CircleCheck
  return Timer
}

function statusClass(status) {
  return `is-${status}`
}

function statusLabel(event) {
  if (event.status === 'overdue' && event.overdue_days > 0) return `逾期 ${event.overdue_days} 天`
  return event.status_label || '待处理'
}

function sourceCountLabel(source) {
  if (source === 'task') return summary.value.tasks
  if (source === 'doubao_schedule') return summary.value.doubao_schedules
  if (source === 'personal_plan') return summary.value.personal_plans
  return summary.value.total
}

async function loadEvents() {
  loading.value = true
  loadError.value = ''
  try {
    response.value = await getCalendarEvents({
      start: localDateValue(gridStart.value),
      end: localDateValue(gridEnd.value),
      sources: sourceValues.value,
      includeCompleted: showCompleted.value,
    })
  } catch (error) {
    response.value = createEmptyResponse()
    loadError.value = error.message
  } finally {
    loading.value = false
  }
}

function changeMonth(delta) {
  const next = new Date(viewDate.value.getFullYear(), viewDate.value.getMonth() + delta, 1)
  viewDate.value = next
  const current = new Date()
  selectedDate.value = next.getFullYear() === current.getFullYear() && next.getMonth() === current.getMonth()
    ? localDateValue(current)
    : localDateValue(next)
}

function goToday() {
  const current = new Date()
  viewDate.value = startOfMonth(current)
  selectedDate.value = localDateValue(current)
}

function selectDate(day) {
  selectedDate.value = day.value
  if (!day.isCurrentMonth) viewDate.value = startOfMonth(day.date)
}

function openEvent(event) {
  if (!event.route) return
  router.push(event.route)
  if (event.source !== 'task') {
    window.setTimeout(() => {
      const targetID = event.source === 'personal_plan' ? 'personal-plans' : 'doubao-schedule'
      document.getElementById(targetID)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }, 80)
  }
}

async function completeEvent(event) {
  if (!event.can_complete || completingKey.value) return
  if (event.source === 'task') {
    try {
      await ElMessageBox.confirm('完成后将不能恢复或修改这个任务，确认继续吗？', '标记为已完成', {
        type: 'info',
        confirmButtonText: '确认完成',
        cancelButtonText: '再想想',
      })
    } catch {
      return
    }
  }

  completingKey.value = event.id
  try {
    if (event.source === 'task') await completeTask(event.source_id)
    else if (event.source === 'doubao_schedule') await completeSchedule(event.source_id)
    else await completePlan(event.source_id)
    await loadEvents()
    ElMessage.success(`“${event.title}”已完成。`)
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    completingKey.value = ''
  }
}

watch([viewDate, sourceFilter, showCompleted], loadEvents)
onMounted(() => {
  loadEvents()
})
</script>

<template>
  <section class="calendar-board">
    <div class="calendar-board-heading">
      <div>
        <p class="calendar-eyebrow">CALENDAR OVERVIEW</p>
        <h2>日历总览</h2>
        <p>任务、豆包日程和个人计划，都在同一张日历里按日期展开。</p>
      </div>
      <div class="calendar-heading-actions">
        <el-button class="soft-button" @click="goToday">今天</el-button>
        <el-button class="soft-button" :loading="loading" @click="loadEvents">
          <Refresh class="button-icon" />刷新
        </el-button>
      </div>
    </div>

    <div class="calendar-filter-row">
      <div class="calendar-source-tabs" role="tablist" aria-label="日历来源筛选">
        <button
          v-for="option in sourceOptions"
          :key="option.value"
          type="button"
          class="calendar-source-tab"
          :class="{ 'is-active': sourceFilter === option.value }"
          :aria-selected="sourceFilter === option.value"
          role="tab"
          @click="sourceFilter = option.value"
        >
          {{ option.label }}
          <span>{{ sourceCountLabel(option.value) }}</span>
        </button>
      </div>
      <label class="calendar-completed-toggle">
        <input v-model="showCompleted" type="checkbox" />
        <span>显示已完成</span>
      </label>
    </div>

    <div v-if="loadError" class="calendar-error" role="alert">
      <WarningFilled />
      <span>日历暂时无法读取：{{ loadError }}</span>
      <el-button text type="primary" @click="loadEvents">重新加载</el-button>
    </div>

    <div class="calendar-summary-grid">
      <div class="surface-card calendar-summary-card is-today">
        <div class="calendar-summary-icon"><Calendar /></div>
        <div><span>今天待处理</span><strong>{{ summary.today }}</strong><small>{{ summary.today ? '按来源逐项处理' : '今天暂时没有安排' }}</small></div>
      </div>
      <div class="surface-card calendar-summary-card is-overdue">
        <div class="calendar-summary-icon"><WarningFilled /></div>
        <div><span>逾期提醒</span><strong>{{ summary.overdue }}</strong><small>{{ summary.overdue ? '完成后重新计算周期' : '所有提醒都在节奏中' }}</small></div>
      </div>
      <div class="surface-card calendar-summary-card is-total">
        <div class="calendar-summary-icon"><List /></div>
        <div><span>当前范围</span><strong>{{ summary.total }}</strong><small>{{ gridRangeLabel }} · {{ showCompleted ? '含已完成' : '未完成为主' }}</small></div>
      </div>
    </div>

    <div class="calendar-layout">
      <div class="surface-card calendar-month-card">
        <div class="calendar-month-heading">
          <div>
            <p class="calendar-card-overline">MONTH VIEW</p>
            <h3>{{ monthLabel }}</h3>
          </div>
          <div class="calendar-month-actions">
            <button type="button" title="上个月" aria-label="上个月" @click="changeMonth(-1)"><ArrowLeft /></button>
            <button type="button" title="下个月" aria-label="下个月" @click="changeMonth(1)"><ArrowRight /></button>
          </div>
        </div>

        <div class="calendar-weekdays" aria-hidden="true">
          <span v-for="label in ['周一', '周二', '周三', '周四', '周五', '周六', '周日']" :key="label">{{ label }}</span>
        </div>
        <div class="calendar-grid" :class="{ 'is-loading': loading }">
          <article
            v-for="day in calendarDays"
            :key="day.value"
            class="calendar-day"
            :class="{
              'is-outside': !day.isCurrentMonth,
              'is-today': day.isToday,
              'is-selected': day.isSelected,
            }"
            role="button"
            tabindex="0"
            :aria-label="`${formatLongDate(day.value)}，${eventsForDate(day.value).length} 项安排`"
            @click="selectDate(day)"
            @keydown.enter="selectDate(day)"
          >
            <div class="calendar-day-heading">
              <span class="calendar-day-number">{{ day.day }}</span>
              <span v-if="day.isToday" class="calendar-today-label">今天</span>
            </div>
            <div class="calendar-day-events">
              <button
                v-for="event in visibleEventsForDate(day.value)"
                :key="event.id"
                type="button"
                class="calendar-event"
                :class="[sourceClass(event.source), statusClass(event.status)]"
                :title="`${event.source_label} · ${event.title}`"
                @click.stop="openEvent(event)"
              >
                <span class="calendar-event-time">{{ event.all_day ? '全天' : formatEventTime(event.start_at) }}</span>
                <span class="calendar-event-title">{{ event.title }}</span>
              </button>
              <span v-if="eventOverflow(day.value)" class="calendar-event-more">还有 {{ eventOverflow(day.value) }} 项</span>
            </div>
          </article>
        </div>
        <div class="calendar-legend">
          <span><i class="legend-dot is-task"></i>任务</span>
          <span><i class="legend-dot is-doubao"></i>豆包日程</span>
          <span><i class="legend-dot is-personal"></i>个人计划</span>
          <span><i class="legend-dot is-overdue"></i>逾期</span>
        </div>
      </div>

      <aside class="surface-card calendar-agenda-card">
        <div class="calendar-agenda-heading">
          <div>
            <p class="calendar-card-overline">SELECTED DAY</p>
            <h3>{{ selectedDateLabel }}</h3>
            <p>{{ selectedEvents.length }} 项安排 · 点击事件可进入对应模块</p>
          </div>
          <span class="calendar-agenda-badge" :class="{ 'is-today': selectedDate === today }">{{ selectedDate === today ? '今天' : '已选择' }}</span>
        </div>

        <div v-if="loading" class="calendar-agenda-loading">
          <span v-for="line in 3" :key="line"></span>
        </div>
        <div v-else-if="selectedEvents.length" class="calendar-agenda-list">
          <article
            v-for="event in selectedEvents"
            :key="event.id"
            class="calendar-agenda-item"
            :class="[sourceClass(event.source), statusClass(event.status)]"
          >
            <div class="calendar-agenda-marker"><component :is="sourceIcon(event.source)" /></div>
            <div class="calendar-agenda-main">
              <div class="calendar-agenda-title-line">
                <strong>{{ event.title }}</strong>
                <span class="calendar-status-label">{{ statusLabel(event) }}</span>
              </div>
              <p>{{ event.source_label }}<template v-if="formatEventMeta(event)"> · {{ formatEventMeta(event) }}</template></p>
              <small v-if="event.note || event.description">{{ event.note || event.description }}</small>
              <div class="calendar-agenda-actions">
                <el-button text size="small" @click="openEvent(event)">{{ event.source === 'task' ? '查看任务' : '管理安排' }}</el-button>
                <el-button
                  v-if="event.can_complete"
                  type="primary"
                  size="small"
                  :loading="completingKey === event.id"
                  @click="completeEvent(event)"
                >
                  <Check class="button-icon" />完成
                </el-button>
              </div>
            </div>
          </article>
        </div>
        <div v-else class="calendar-agenda-empty">
          <div class="calendar-empty-icon"><Check /></div>
          <strong>这一天还没有安排</strong>
          <p>可以去任务、豆包日程或个人计划中添加一件需要记住的事。</p>
        </div>

        <div class="calendar-agenda-note">
          <Clock />
          <span>逾期的周期事项会留在今天，完成后从当天重新计算下一次提醒。</span>
        </div>
      </aside>
    </div>
  </section>
</template>

<style scoped>
.calendar-board {
  margin-top: 4px;
}

.calendar-board-heading,
.calendar-month-heading,
.calendar-agenda-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
}

.calendar-board-heading {
  margin-bottom: 18px;
}

.calendar-eyebrow,
.calendar-card-overline {
  margin: 0 0 6px;
  color: var(--ds-color-primary);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.14em;
  line-height: 1.3;
}

.calendar-board-heading h2,
.calendar-month-heading h3,
.calendar-agenda-heading h3 {
  margin: 0;
  color: var(--ds-color-text-primary);
  letter-spacing: -0.04em;
}

.calendar-board-heading h2 {
  font-size: 24px;
}

.calendar-month-heading h3,
.calendar-agenda-heading h3 {
  font-size: 18px;
}

.calendar-board-heading p:last-child,
.calendar-agenda-heading p:last-child {
  margin: 5px 0 0;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  line-height: 1.6;
}

.calendar-heading-actions,
.calendar-month-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.calendar-filter-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.calendar-source-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.calendar-source-tab {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 34px;
  padding: 0 11px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  cursor: pointer;
  background: var(--ds-color-surface);
  border: 1px solid var(--ds-color-border);
  border-radius: var(--ds-radius-pill);
  transition: color 160ms ease, background-color 160ms ease, border-color 160ms ease;
}

.calendar-source-tab span {
  min-width: 17px;
  padding: 1px 5px;
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
  text-align: center;
  background: var(--ds-color-surface-muted);
  border-radius: var(--ds-radius-pill);
}

.calendar-source-tab:hover,
.calendar-source-tab.is-active {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
  border-color: #d3daea;
}

.calendar-source-tab.is-active span {
  color: var(--ds-color-primary);
  background: #ffffff;
}

.calendar-completed-toggle {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  flex: 0 0 auto;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  cursor: pointer;
}

.calendar-completed-toggle input {
  width: 15px;
  height: 15px;
  accent-color: var(--ds-color-primary);
}

.calendar-error {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-bottom: 16px;
  padding: 11px 14px;
  color: #8b5a26;
  font-size: 13px;
  background: #fff8e7;
  border: 1px solid #f2dfb2;
  border-radius: var(--ds-radius-md);
}

.calendar-error svg {
  width: 16px;
  flex: 0 0 auto;
}

.calendar-error .el-button {
  margin-left: auto;
}

.calendar-summary-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 16px;
}

.calendar-summary-card {
  display: flex;
  align-items: center;
  gap: 13px;
  min-height: 88px;
  padding: 16px 18px;
}

.calendar-summary-icon {
  display: grid;
  width: 36px;
  height: 36px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 11px;
}

.calendar-summary-icon svg {
  width: 18px;
}

.calendar-summary-card.is-today .calendar-summary-icon {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.calendar-summary-card.is-overdue .calendar-summary-icon {
  color: #b56569;
  background: #fbe5e2;
}

.calendar-summary-card.is-total .calendar-summary-icon {
  color: #5d9076;
  background: #e8f4ed;
}

.calendar-summary-card span,
.calendar-summary-card strong,
.calendar-summary-card small {
  display: block;
}

.calendar-summary-card span {
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.calendar-summary-card strong {
  margin: 1px 0;
  color: var(--ds-color-text-primary);
  font-size: 22px;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.03em;
}

.calendar-summary-card small {
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
}

.calendar-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(350px, 0.6fr);
  gap: 16px;
}

.calendar-month-card,
.calendar-agenda-card {
  min-width: 0;
  padding: 22px;
}

.calendar-month-actions button {
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  color: var(--ds-color-text-tertiary);
  cursor: pointer;
  background: var(--ds-color-surface-muted);
  border: 1px solid var(--ds-color-border);
  border-radius: 8px;
}

.calendar-month-actions button:hover {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.calendar-month-actions svg {
  width: 14px;
}

.calendar-weekdays,
.calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
}

.calendar-weekdays {
  margin-top: 20px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  font-weight: 700;
  text-align: right;
}

.calendar-weekdays span {
  padding: 0 9px 8px;
}

.calendar-grid {
  border-top: 1px solid var(--ds-color-border);
  border-left: 1px solid var(--ds-color-border);
}

.calendar-day {
  position: relative;
  min-width: 0;
  min-height: 122px;
  padding: 8px;
  text-align: left;
  cursor: pointer;
  background: var(--ds-color-surface);
  border-right: 1px solid var(--ds-color-border);
  border-bottom: 1px solid var(--ds-color-border);
  transition: background-color 160ms ease, box-shadow 160ms ease;
}

.calendar-day:hover {
  background: #fbfcfe;
}

.calendar-day.is-outside {
  color: var(--ds-color-text-disabled);
  background: var(--ds-color-surface-muted);
}

.calendar-day.is-selected {
  z-index: 1;
  box-shadow: inset 0 0 0 2px var(--ds-color-primary);
}

.calendar-day.is-today {
  background: #fffdf7;
}

.calendar-day.is-selected.is-today {
  background: #fffaf0;
}

.calendar-day-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 20px;
}

.calendar-day-number {
  display: inline-grid;
  width: 24px;
  height: 24px;
  place-items: center;
  color: var(--ds-color-text-secondary);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  border-radius: 50%;
}

.is-outside .calendar-day-number {
  color: var(--ds-color-text-disabled);
}

.is-today .calendar-day-number {
  color: #ffffff;
  font-weight: 700;
  background: var(--ds-color-primary);
}

.calendar-today-label {
  color: #a4772e;
  font-size: 10px;
}

.calendar-day-events {
  display: grid;
  gap: 4px;
  margin-top: 7px;
}

.calendar-event {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 4px;
  padding: 4px 5px;
  color: var(--event-color);
  font-size: 10px;
  line-height: 1.3;
  text-align: left;
  cursor: pointer;
  background: var(--event-background);
  border-left: 2px solid var(--event-color);
  border-radius: 4px;
}

.calendar-event:hover {
  filter: brightness(0.97);
}

.calendar-event.is-task {
  --event-color: var(--ds-color-primary);
  --event-background: #edf1fa;
}

.calendar-event.is-doubao-schedule {
  --event-color: #a4772e;
  --event-background: #fff4d7;
}

.calendar-event.is-personal-plan {
  --event-color: #5d9076;
  --event-background: #e8f4ed;
}

.calendar-event.is-overdue {
  --event-color: var(--ds-color-danger);
  --event-background: #fff0ee;
}

.calendar-event.is-completed {
  opacity: 0.5;
  text-decoration: line-through;
}

.calendar-event-time {
  flex: 0 0 auto;
  font-size: 9px;
  font-variant-numeric: tabular-nums;
  opacity: 0.75;
}

.calendar-event-title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.calendar-event-more {
  padding: 1px 4px;
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
}

.calendar-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 13px;
  margin-top: 15px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.calendar-legend span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.legend-dot.is-task { background: var(--ds-color-primary); }
.legend-dot.is-doubao { background: #d6a247; }
.legend-dot.is-personal { background: #6eaa8a; }
.legend-dot.is-overdue { background: var(--ds-color-danger); }

.calendar-agenda-badge {
  padding: 6px 9px;
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
  white-space: nowrap;
  background: var(--ds-color-surface-muted);
  border: 1px solid var(--ds-color-border);
  border-radius: var(--ds-radius-pill);
}

.calendar-agenda-badge.is-today {
  color: #a4772e;
  background: #fff5d9;
  border-color: #f2dfb2;
}

.calendar-agenda-list {
  display: grid;
  gap: 10px;
  max-height: 528px;
  margin-top: 20px;
  padding-right: 2px;
  overflow-y: auto;
}

.calendar-agenda-item {
  display: grid;
  grid-template-columns: 32px minmax(0, 1fr);
  gap: 10px;
  padding: 12px;
  background: var(--ds-color-surface-muted);
  border: 1px solid var(--ds-color-border);
  border-left: 3px solid var(--event-color);
  border-radius: 11px;
}

.calendar-agenda-item.is-task { --event-color: var(--ds-color-primary); }
.calendar-agenda-item.is-doubao-schedule { --event-color: #d6a247; }
.calendar-agenda-item.is-personal-plan { --event-color: #6eaa8a; }
.calendar-agenda-item.is-overdue { --event-color: var(--ds-color-danger); background: #fff8f6; }
.calendar-agenda-item.is-completed { opacity: 0.58; }

.calendar-agenda-marker {
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  color: var(--event-color);
  background: color-mix(in srgb, var(--event-color) 12%, white);
  border-radius: 9px;
}

.calendar-agenda-marker svg {
  width: 15px;
}

.calendar-agenda-main {
  min-width: 0;
}

.calendar-agenda-title-line {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 7px;
}

.calendar-agenda-title-line strong {
  min-width: 0;
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.calendar-status-label {
  flex: 0 0 auto;
  padding: 2px 5px;
  color: var(--event-color);
  font-size: 9px;
  background: color-mix(in srgb, var(--event-color) 10%, white);
  border-radius: 4px;
}

.calendar-agenda-item.is-completed .calendar-status-label {
  color: var(--ds-color-text-tertiary);
  background: var(--ds-color-border);
}

.calendar-agenda-main p,
.calendar-agenda-main small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.calendar-agenda-main p {
  margin: 4px 0 0;
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
}

.calendar-agenda-main small {
  margin-top: 5px;
  color: var(--ds-color-text-secondary);
  font-size: 10px;
}

.calendar-agenda-actions {
  display: flex;
  justify-content: flex-end;
  gap: 4px;
  margin-top: 8px;
}

.calendar-agenda-actions .el-button {
  min-height: 26px;
  padding: 4px 7px;
  font-size: 11px;
}

.calendar-agenda-actions .button-icon {
  width: 13px;
  height: 13px;
  margin-right: 4px;
}

.calendar-agenda-empty {
  display: flex;
  min-height: 280px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
  text-align: center;
}

.calendar-empty-icon {
  display: grid;
  width: 42px;
  height: 42px;
  margin-bottom: 11px;
  place-items: center;
  color: #5d9275;
  background: #e8f4ed;
  border-radius: 13px;
}

.calendar-empty-icon svg {
  width: 20px;
}

.calendar-agenda-empty strong {
  color: var(--ds-color-text-primary);
  font-size: 13px;
}

.calendar-agenda-empty p {
  max-width: 240px;
  margin: 6px 0 0;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  line-height: 1.6;
}

.calendar-agenda-note {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 16px;
  padding: 11px 12px;
  color: #8b7956;
  font-size: 10px;
  line-height: 1.55;
  background: #f8f5ed;
  border: 1px solid #eee4cb;
  border-radius: 9px;
}

.calendar-agenda-note svg {
  width: 14px;
  flex: 0 0 auto;
  margin-top: 1px;
  color: #a87a2d;
}

.calendar-agenda-loading {
  display: grid;
  gap: 10px;
  margin-top: 20px;
}

.calendar-agenda-loading span {
  height: 72px;
  background: linear-gradient(90deg, #f2f4f8 20%, #fafbfc 45%, #f2f4f8 70%);
  background-size: 200% 100%;
  border-radius: 11px;
  animation: calendar-loading 1.4s ease-in-out infinite;
}

@keyframes calendar-loading {
  from { background-position: 100% 0; }
  to { background-position: -100% 0; }
}

@media (max-width: 1100px) {
  .calendar-layout {
    grid-template-columns: 1fr;
  }

  .calendar-agenda-list {
    max-height: none;
  }
}

@media (max-width: 760px) {
  .calendar-board-heading,
  .calendar-filter-row {
    display: block;
  }

  .calendar-heading-actions {
    margin-top: 14px;
  }

  .calendar-completed-toggle {
    margin-top: 12px;
  }

  .calendar-summary-grid {
    grid-template-columns: 1fr;
  }

  .calendar-month-card,
  .calendar-agenda-card {
    padding: 15px;
  }

  .calendar-day {
    min-height: 105px;
    padding: 5px;
  }

  .calendar-weekdays span {
    padding-right: 5px;
    padding-left: 5px;
  }

  .calendar-event {
    display: block;
  }

  .calendar-event-time {
    display: none;
  }
}
</style>
