<script setup>
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowRight,
  Bowl,
  Calendar,
  Check,
  CircleCheck,
  Film,
  List,
  Notebook,
  Picture,
  Refresh,
  Timer,
  WarningFilled,
} from '@element-plus/icons-vue'
import {
  completePlan,
  completeSchedule,
  completeTask,
  getCalendarEvents,
  getDoubaoAlbum,
  getEntertainment,
  getHealth,
  getMemos,
  getRecipe,
  getTasks,
} from '../api'

const router = useRouter()

const todayValue = localDateValue(new Date())
const health = ref(null)
const healthLoading = ref(true)
const healthError = ref('')
const dashboardLoading = ref(true)
const loadErrors = ref([])
const tasks = ref([])
const calendar = ref(createEmptyCalendar())
const recipe = ref(createEmptyRecipe())
const albumStats = ref({ total: 0, photos: 0, videos: 0 })
const memos = ref([])
const entertainment = ref([])
const completingKey = ref('')

const todayLabel = computed(() => new Intl.DateTimeFormat('zh-CN', {
  month: 'long',
  day: 'numeric',
  weekday: 'long',
}).format(new Date()))

const allTasks = computed(() => flattenTasks(tasks.value))
const incompleteTasks = computed(() => allTasks.value.filter((task) => !task.completed))
const todayTasks = computed(() => incompleteTasks.value.filter((task) => dateValueFromInput(task.due_at) === todayValue))
const overdueTasks = computed(() => incompleteTasks.value.filter((task) => isTaskOverdue(task)))
const recipeFoods = computed(() => {
  const weekday = recipe.value.today || weekdayNumber(new Date())
  return (recipe.value.foods || []).filter((food) => (food.days || []).includes(weekday))
})

const calendarEvents = computed(() => (calendar.value.events || []).filter((event) => !event.completed))
const todayEvents = computed(() => {
  const events = [...calendarEvents.value]
  const calendarTaskIDs = new Set(
    events.filter((event) => event.source === 'task').map((event) => String(event.source_id)),
  )

  for (const task of overdueTasks.value) {
    if (calendarTaskIDs.has(String(task.id))) continue
    events.push(createOverdueTaskEvent(task))
  }

  return events.sort(compareEvents)
})
const visibleTodayEvents = computed(() => todayEvents.value.slice(0, 8))
const remainingTodayEvents = computed(() => Math.max(todayEvents.value.length - visibleTodayEvents.value.length, 0))
const todayDoubaoEvents = computed(() => todayEvents.value.filter((event) => event.source === 'doubao_schedule'))
const overdueCount = computed(() => todayEvents.value.filter((event) => event.status === 'overdue').length)
const summary = computed(() => calendar.value.summary || createEmptyCalendar().summary)

const stats = computed(() => [
  {
    label: '今日待处理',
    value: String(todayEvents.value.length),
    note: overdueCount.value ? `${overdueCount.value} 项已逾期` : '任务与周期提醒已汇总',
    icon: Calendar,
    tone: 'blue',
  },
  {
    label: '未完成任务',
    value: String(incompleteTasks.value.length),
    note: todayTasks.value.length ? `${todayTasks.value.length} 项今天到期` : '今天没有任务到期',
    icon: Check,
    tone: 'green',
  },
  {
    label: '豆包今日食谱',
    value: String(recipeFoods.value.length),
    note: recipeFoods.value.length ? `${recipe.value.today_label || '今天'}已安排` : '今天还没有安排食物',
    icon: Bowl,
    tone: 'warm',
  },
  {
    label: '循环提醒',
    value: String((summary.value.doubao_schedules || 0) + (summary.value.personal_plans || 0)),
    note: `豆包 ${summary.value.doubao_schedules || 0} · 个人 ${summary.value.personal_plans || 0}`,
    icon: Timer,
    tone: 'purple',
  },
])

const memoItems = computed(() => memos.value.slice(0, 3))
const entertainmentItems = computed(() => entertainment.value.slice(0, 3))
const quickLinks = [
  { to: '/dashboard/tasks', label: '任务', note: '拆解今天要做的事', icon: List, tone: 'blue' },
  { to: '/dashboard/doubao', label: '豆包', note: '食谱、日程和相册', icon: Bowl, tone: 'warm' },
  { to: '/dashboard/calendar', label: '日历', note: '查看所有日期安排', icon: Calendar, tone: 'green' },
  { to: '/dashboard/plans#personal-plans', label: '个人计划', note: '管理周期性打卡', icon: CircleCheck, tone: 'purple' },
  { to: '/dashboard/memos', label: '备忘录', note: '记录需要保留的内容', icon: Notebook, tone: 'blue' },
  { to: '/dashboard/entertainment', label: '娱乐', note: '小说、剧集和电影', icon: Film, tone: 'pink' },
]

function createEmptyCalendar() {
  return {
    today: todayValue,
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

function createEmptyRecipe() {
  return {
    today: weekdayNumber(new Date()),
    today_label: '',
    foods: [],
  }
}

function flattenTasks(nodes) {
  return (nodes || []).flatMap((task) => [task, ...flattenTasks(task.children || [])])
}

function weekdayNumber(value) {
  const day = new Date(value).getDay()
  return day === 0 ? 7 : day
}

function localDateValue(value) {
  const date = new Date(value)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function dateValueFromInput(value) {
  if (!value) return ''
  if (typeof value === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(value)) return value
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : localDateValue(date)
}

function isTaskOverdue(task) {
  const dueDate = dateValueFromInput(task.due_at)
  return !task.completed && Boolean(dueDate) && dueDate < todayValue
}

function createOverdueTaskEvent(task) {
  return {
    id: `task:${task.id}`,
    source: 'task',
    source_id: task.id,
    source_label: '任务',
    title: task.title,
    date: todayValue,
    start_at: task.due_at,
    all_day: !task.due_at.includes('T'),
    status: 'overdue',
    status_label: '已逾期',
    completed: false,
    can_complete: true,
    priority: task.effective_priority || task.priority,
    priority_label: priorityLabel(task.effective_priority || task.priority),
    description: task.description,
    route: `/dashboard/tasks/${task.id}`,
  }
}

function compareEvents(left, right) {
  const leftOverdue = left.status === 'overdue' ? 0 : 1
  const rightOverdue = right.status === 'overdue' ? 0 : 1
  if (leftOverdue !== rightOverdue) return leftOverdue - rightOverdue
  if (left.all_day !== right.all_day) return left.all_day ? 1 : -1
  if (left.start_at && right.start_at) {
    const difference = new Date(left.start_at).getTime() - new Date(right.start_at).getTime()
    if (difference !== 0) return difference
  }
  return String(left.title || '').localeCompare(String(right.title || ''), 'zh-CN')
}

function priorityLabel(priority) {
  return { urgent: '紧急', high: '高优先级', medium: '中优先级', low: '低优先级' }[priority] || '任务'
}

function sourceClass(source) {
  return `is-${String(source || '').replace('_', '-')}`
}

function sourceIcon(source) {
  if (source === 'task') return List
  if (source === 'personal_plan') return CircleCheck
  return Timer
}

function statusClass(status) {
  return `is-${status || 'upcoming'}`
}

function statusLabel(event) {
  if (event.status === 'overdue' && event.overdue_days > 0) return `逾期 ${event.overdue_days} 天`
  return event.status_label || '待处理'
}

function formatEventTime(value) {
  if (!value) return '全天'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '全天'
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

function formatShortDate(value) {
  const dateValue = dateValueFromInput(value)
  if (!dateValue) return ''
  const [, month, day] = dateValue.split('-').map(Number)
  return `${month}月${day}日`
}

function formatEventMeta(event) {
  if (event.source === 'task') return event.priority_label || '任务'
  const details = []
  if (event.status === 'overdue' && event.original_date) {
    details.push(`原提醒日 ${formatShortDate(event.original_date)}`)
  } else if (event.interval_days) {
    details.push(`每 ${event.interval_days} 天`)
  }
  if (event.next_due_date && event.status !== 'overdue') {
    details.push(`下次 ${formatShortDate(event.next_due_date)}`)
  }
  return details.join(' · ')
}

function formatDateTime(value) {
  if (!value) return ''
  const normalized = typeof value === 'string' ? value.replace(' ', 'T') : value
  const date = new Date(normalized)
  if (Number.isNaN(date.getTime())) return value
  if (localDateValue(date) === todayValue) return `今天 ${formatEventTime(date)}`
  return new Intl.DateTimeFormat('zh-CN', { month: 'numeric', day: 'numeric' }).format(date)
}

function errorMessage(error) {
  return error?.message || '读取失败'
}

async function refreshDashboard() {
  dashboardLoading.value = true
  healthLoading.value = true
  loadErrors.value = []

  const results = await Promise.allSettled([
    getHealth(),
    getTasks({ includeCompleted: true, sort: 'due' }),
    getCalendarEvents({ start: todayValue, end: todayValue }),
    getRecipe(),
    getDoubaoAlbum(),
    getMemos({ archived: false }),
    getEntertainment({ status: 'in_progress', sort: 'updated' }),
  ])

  const failed = []
  const [healthResult, tasksResult, calendarResult, recipeResult, albumResult, memosResult, entertainmentResult] = results

  if (healthResult.status === 'fulfilled') {
    health.value = healthResult.value
    healthError.value = ''
  } else {
    health.value = null
    healthError.value = errorMessage(healthResult.reason)
    failed.push('服务状态')
  }

  if (tasksResult.status === 'fulfilled') tasks.value = tasksResult.value || []
  else {
    tasks.value = []
    failed.push('任务')
  }

  if (calendarResult.status === 'fulfilled') calendar.value = calendarResult.value || createEmptyCalendar()
  else {
    calendar.value = createEmptyCalendar()
    failed.push('日历')
  }

  if (recipeResult.status === 'fulfilled') recipe.value = recipeResult.value || createEmptyRecipe()
  else {
    recipe.value = createEmptyRecipe()
    failed.push('豆包食谱')
  }

  if (albumResult.status === 'fulfilled') albumStats.value = albumResult.value?.stats || { total: 0, photos: 0, videos: 0 }
  else {
    albumStats.value = { total: 0, photos: 0, videos: 0 }
    failed.push('豆包相册')
  }

  if (memosResult.status === 'fulfilled') memos.value = memosResult.value?.items || []
  else {
    memos.value = []
    failed.push('备忘录')
  }

  if (entertainmentResult.status === 'fulfilled') entertainment.value = entertainmentResult.value?.items || []
  else {
    entertainment.value = []
    failed.push('娱乐')
  }

  loadErrors.value = failed
  healthLoading.value = false
  dashboardLoading.value = false
}

async function refreshHealth() {
  healthLoading.value = true
  healthError.value = ''
  try {
    health.value = await getHealth()
    loadErrors.value = loadErrors.value.filter((item) => item !== '服务状态')
  } catch (error) {
    health.value = null
    healthError.value = errorMessage(error)
    if (!loadErrors.value.includes('服务状态')) loadErrors.value = [...loadErrors.value, '服务状态']
  } finally {
    healthLoading.value = false
  }
}

function goToTasks() {
  router.push('/dashboard/tasks')
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
    await refreshDashboard()
    ElMessage.success(`“${event.title}”已完成。`)
  } catch (error) {
    ElMessage.error(errorMessage(error))
  } finally {
    completingKey.value = ''
  }
}

onMounted(() => {
  refreshDashboard()
})
</script>

<template>
  <div class="page-view dashboard-page">
    <section class="dashboard-heading page-heading">
      <div class="page-heading-copy">
        <p class="eyebrow">{{ todayLabel }}</p>
        <h1>今天的工作台</h1>
        <p>把任务、豆包安排、个人计划和最近内容集中到一个清晰的入口。</p>
      </div>
      <div class="page-heading-actions dashboard-heading-actions">
        <span class="dashboard-service-pill" :class="{ 'is-error': healthError, 'is-loading': healthLoading }">
          <span class="dashboard-service-dot"></span>
          {{ healthLoading ? '检查中' : (healthError ? '服务异常' : '本地服务正常') }}
        </span>
        <el-button class="soft-button" :loading="dashboardLoading" @click="refreshDashboard">
          <Refresh class="button-icon" />刷新
        </el-button>
        <el-button type="primary" @click="goToTasks">管理任务</el-button>
      </div>
    </section>

    <div v-if="loadErrors.length" class="dashboard-partial-warning" role="alert">
      <WarningFilled />
      <span>部分模块暂时无法读取：{{ loadErrors.join('、') }}。</span>
      <el-button text type="primary" @click="refreshDashboard">重新加载</el-button>
    </div>

    <section class="dashboard-stat-grid" aria-label="工作台摘要">
      <article v-for="stat in stats" :key="stat.label" class="dashboard-stat-card">
        <div class="stat-icon" :class="`is-${stat.tone}`"><component :is="stat.icon" /></div>
        <div class="stat-content">
          <div class="stat-label">{{ stat.label }}</div>
          <div class="stat-number">{{ stat.value }}</div>
          <div class="stat-note">{{ stat.note }}</div>
        </div>
      </article>
    </section>

    <section class="dashboard-main-grid">
      <article class="surface-card dashboard-panel today-panel">
        <div class="card-heading">
          <div>
            <h2>今天要处理</h2>
            <p>任务、豆包日程和个人计划的今日安排。</p>
          </div>
          <RouterLink to="/dashboard/calendar" class="muted-link">完整日历 <ArrowRight /></RouterLink>
        </div>

        <div v-if="dashboardLoading" class="dashboard-loading">正在整理今天的安排……</div>
        <div v-else-if="visibleTodayEvents.length" class="dashboard-event-list">
          <article
            v-for="event in visibleTodayEvents"
            :key="event.id"
            class="dashboard-event"
            :class="statusClass(event.status)"
          >
            <div class="dashboard-event-time">{{ event.all_day ? '全天' : formatEventTime(event.start_at) }}</div>
            <div class="dashboard-event-line" :class="sourceClass(event.source)"></div>
            <div class="dashboard-event-main">
              <div class="dashboard-event-title-line">
                <strong>{{ event.title }}</strong>
                <span class="dashboard-event-status" :class="statusClass(event.status)">{{ statusLabel(event) }}</span>
              </div>
              <p class="dashboard-event-meta">
                {{ event.source_label }}<template v-if="formatEventMeta(event)"> · {{ formatEventMeta(event) }}</template>
              </p>
              <p v-if="event.note || event.description" class="dashboard-event-note">{{ event.note || event.description }}</p>
              <div class="dashboard-event-actions">
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
        <div v-else class="dashboard-empty-state">
          <div class="dashboard-empty-icon"><CircleCheck /></div>
          <strong>今天暂时没有待处理安排</strong>
          <p>可以去任务、豆包日程或个人计划中添加需要记住的事。</p>
        </div>
        <p v-if="remainingTodayEvents" class="dashboard-more-hint">
          还有 {{ remainingTodayEvents }} 项安排，<RouterLink to="/dashboard/calendar">去日历查看全部</RouterLink>
        </p>
      </article>

      <aside class="surface-card dashboard-panel doubao-today-panel">
        <div class="card-heading">
          <div>
            <h2>豆包今天</h2>
            <p>{{ recipe.today_label || '今日食谱与提醒' }}</p>
          </div>
          <RouterLink to="/dashboard/doubao" class="muted-link">打开豆包 <ArrowRight /></RouterLink>
        </div>

        <div class="dashboard-subsection">
          <div class="dashboard-subheading"><span class="subheading-icon is-warm"><Bowl /></span><strong>今日食谱</strong><span class="subheading-count">{{ recipeFoods.length }}</span></div>
          <div v-if="recipeFoods.length" class="recipe-mini-list">
            <div v-for="food in recipeFoods" :key="food.id" class="recipe-mini-item">
              <span class="recipe-food-dot"></span>
              <div><strong>{{ food.name }}</strong><small>{{ food.category || '日常安排' }}<template v-if="food.note"> · {{ food.note }}</template></small></div>
            </div>
          </div>
          <div v-else class="dashboard-subsection-empty">今天还没有安排食物。</div>
          <RouterLink to="/dashboard/doubao#doubao-recipe" class="dashboard-inline-link">管理豆包食谱 <ArrowRight /></RouterLink>
        </div>

        <div class="dashboard-subsection">
          <div class="dashboard-subheading"><span class="subheading-icon is-blue"><Timer /></span><strong>豆包日程</strong><span class="subheading-count">{{ todayDoubaoEvents.length }}</span></div>
          <div v-if="todayDoubaoEvents.length" class="schedule-mini-list">
            <div v-for="event in todayDoubaoEvents.slice(0, 3)" :key="event.id" class="schedule-mini-item">
              <span class="schedule-mini-dot" :class="statusClass(event.status)"></span>
              <div><strong>{{ event.title }}</strong><small>{{ statusLabel(event) }}<template v-if="formatEventMeta(event)"> · {{ formatEventMeta(event) }}</template></small></div>
            </div>
          </div>
          <div v-else class="dashboard-subsection-empty">今天没有豆包日程提醒。</div>
          <RouterLink to="/dashboard/doubao#doubao-schedule" class="dashboard-inline-link">管理豆包日程 <ArrowRight /></RouterLink>
        </div>

        <div v-if="summary.personal_plans" class="doubao-personal-note">
          <CircleCheck /> 今天另有 {{ summary.personal_plans }} 项个人计划提醒，可在日历中统一处理。
        </div>

        <div class="album-mini-summary">
          <span class="album-mini-title"><Picture />豆包相册</span>
          <span>{{ albumStats.total }} 条记录 · {{ albumStats.photos }} 张照片 · {{ albumStats.videos }} 个视频</span>
          <RouterLink to="/dashboard/doubao#doubao-album" class="dashboard-inline-link">打开相册 <ArrowRight /></RouterLink>
        </div>
      </aside>
    </section>

    <section class="dashboard-secondary-grid">
      <article class="surface-card dashboard-panel mini-module-panel">
        <div class="card-heading">
          <div>
            <h2>最近备忘录</h2>
            <p>把临时想法留在容易找回的地方。</p>
          </div>
          <RouterLink to="/dashboard/memos" class="muted-link">查看全部 <ArrowRight /></RouterLink>
        </div>
        <div v-if="dashboardLoading" class="dashboard-loading">正在读取备忘录……</div>
        <div v-else-if="memoItems.length" class="mini-item-list">
          <div v-for="memo in memoItems" :key="memo.id" class="mini-item">
            <span class="mini-item-icon is-blue"><Notebook /></span>
            <div class="mini-item-main">
              <strong>{{ memo.title || '无标题备忘录' }}</strong>
              <p>{{ memo.excerpt || '暂无摘要内容' }}</p>
              <small>{{ formatDateTime(memo.updated_at) }}</small>
            </div>
          </div>
        </div>
        <div v-else class="dashboard-subsection-empty">还没有活动备忘录。</div>
      </article>

      <article class="surface-card dashboard-panel mini-module-panel">
        <div class="card-heading">
          <div>
            <h2>正在进行的娱乐</h2>
            <p>记录最近在读、在看和正在追的内容。</p>
          </div>
          <RouterLink to="/dashboard/entertainment" class="muted-link">打开娱乐 <ArrowRight /></RouterLink>
        </div>
        <div v-if="dashboardLoading" class="dashboard-loading">正在读取娱乐内容……</div>
        <div v-else-if="entertainmentItems.length" class="mini-item-list">
          <div v-for="item in entertainmentItems" :key="item.id" class="mini-item">
            <span class="mini-item-icon is-pink"><Film /></span>
            <div class="mini-item-main">
              <div class="mini-item-title-line"><strong>{{ item.title }}</strong><small>{{ item.media_type_label }}</small></div>
              <p>{{ item.creator || '未填写创作者' }}<template v-if="item.progress_label"> · {{ item.progress_label }}</template></p>
              <div class="mini-progress"><span :style="{ width: `${Math.min(Math.max(item.progress_percent || 0, 0), 100)}%` }"></span></div>
            </div>
          </div>
        </div>
        <div v-else class="dashboard-subsection-empty">还没有正在进行的娱乐内容。</div>
      </article>
    </section>

    <section class="surface-card dashboard-panel quick-entry-panel">
      <div class="card-heading">
        <div>
          <h2>快速进入</h2>
          <p>从这里直接进入各个模块，继续处理你的事项。</p>
        </div>
      </div>
      <div class="quick-link-grid">
        <RouterLink v-for="link in quickLinks" :key="link.to" :to="link.to" class="quick-link">
          <span class="quick-link-icon" :class="`is-${link.tone}`"><component :is="link.icon" /></span>
          <span class="quick-link-copy"><strong>{{ link.label }}</strong><small>{{ link.note }}</small></span>
          <ArrowRight class="quick-link-arrow" />
        </RouterLink>
      </div>
    </section>

    <section class="dashboard-health-bar surface-card">
      <div class="dashboard-health-copy">
        <span class="dashboard-health-icon" :class="{ 'is-error': healthError }"><CircleCheck v-if="!healthError" /><WarningFilled v-else /></span>
        <div>
          <strong>{{ healthError ? '本地服务暂时不可用' : '本地服务与数据库' }}</strong>
          <span>{{ healthError || (health ? `已连接 · 版本 ${health.version}` : '正在检查连接状态') }}</span>
        </div>
      </div>
      <el-button text type="primary" :loading="healthLoading" @click="refreshHealth">重新检查</el-button>
    </section>
  </div>
</template>

<style scoped>
.dashboard-heading {
  margin-bottom: 22px;
}

.dashboard-heading-actions {
  align-items: center;
}

.dashboard-service-pill {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: var(--ds-color-success);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.dashboard-service-dot {
  width: 7px;
  height: 7px;
  background: currentColor;
  border-radius: 50%;
}

.dashboard-service-pill.is-error {
  color: var(--ds-color-danger);
}

.dashboard-service-pill.is-loading {
  color: var(--ds-color-text-tertiary);
}

.dashboard-page .el-button--primary {
  margin-bottom: 0;
}

.dashboard-partial-warning {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-bottom: 16px;
  padding: 11px 14px;
  color: #8a651e;
  font-size: 12px;
  background: #fff8e8;
  border: 1px solid #f5e4b9;
  border-radius: 10px;
}

.dashboard-partial-warning > svg {
  width: 16px;
  flex: 0 0 auto;
}

.dashboard-partial-warning span {
  flex: 1;
}

.dashboard-stat-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 18px;
}

.dashboard-stat-card {
  display: flex;
  align-items: center;
  min-height: 112px;
  gap: 14px;
  padding: 20px;
  background: var(--ds-color-surface);
  border: 1px solid var(--ds-color-border);
  border-radius: var(--ds-radius-lg);
}

.stat-icon,
.subheading-icon,
.mini-item-icon,
.quick-link-icon {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 11px;
}

.stat-icon {
  width: 40px;
  height: 40px;
}

.stat-icon svg {
  width: 19px;
}

.stat-icon.is-blue,
.subheading-icon.is-blue,
.mini-item-icon.is-blue,
.quick-link-icon.is-blue {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.stat-icon.is-green,
.quick-link-icon.is-green {
  color: var(--ds-color-success);
  background: #edf8e9;
}

.stat-icon.is-warm,
.subheading-icon.is-warm,
.quick-link-icon.is-warm {
  color: #ae7d2e;
  background: #fff7e6;
}

.stat-icon.is-purple,
.quick-link-icon.is-purple {
  color: #8171c6;
  background: #f1effc;
}

.quick-link-icon.is-pink,
.mini-item-icon.is-pink {
  color: #be6681;
  background: #fff0f4;
}

.stat-content {
  min-width: 0;
}

.stat-label {
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.stat-number {
  margin-top: 2px;
  color: var(--ds-color-text-primary);
  font-size: 27px;
  font-weight: 700;
  letter-spacing: -0.05em;
  line-height: 1.1;
}

.stat-note {
  margin-top: 5px;
  overflow: hidden;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-main-grid,
.dashboard-secondary-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.18fr) minmax(350px, 0.82fr);
  gap: 18px;
  margin-bottom: 18px;
}

.dashboard-panel {
  min-width: 0;
  padding: 24px;
}

.today-panel,
.doubao-today-panel {
  min-height: 390px;
}

.dashboard-event-list {
  margin-top: 18px;
}

.dashboard-event {
  display: grid;
  grid-template-columns: 42px 4px minmax(0, 1fr);
  align-items: stretch;
  gap: 12px;
  min-height: 74px;
  padding: 9px 0;
  border-bottom: 1px solid #eff1f6;
}

.dashboard-event:last-child {
  border-bottom: 0;
}

.dashboard-event-time {
  padding-top: 4px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  text-align: right;
}

.dashboard-event-line {
  width: 4px;
  margin: 3px 0;
  border-radius: 4px;
}

.dashboard-event-line.is-task {
  background: var(--ds-color-primary);
}

.dashboard-event-line.is-doubao-schedule {
  background: var(--ds-color-accent);
}

.dashboard-event-line.is-personal-plan {
  background: var(--ds-color-success);
}

.dashboard-event.is-overdue .dashboard-event-line {
  background: var(--ds-color-danger);
}

.dashboard-event-main {
  min-width: 0;
}

.dashboard-event-title-line {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.dashboard-event-title-line strong {
  min-width: 0;
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-event-status {
  flex: 0 0 auto;
  color: var(--ds-color-primary);
  font-size: 11px;
}

.dashboard-event-status.is-overdue {
  color: var(--ds-color-danger);
}

.dashboard-event-status.is-due {
  color: #ae7d2e;
}

.dashboard-event-meta,
.dashboard-event-note {
  margin: 4px 0 0;
  overflow: hidden;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-event-note {
  margin-top: 3px;
  color: #8a93a6;
}

.dashboard-event-actions {
  display: flex;
  gap: 4px;
  margin-top: 3px;
}

.dashboard-event-actions .el-button {
  min-height: 22px;
  padding: 1px 5px;
  font-size: 11px;
}

.dashboard-event-actions .button-icon {
  width: 13px;
  height: 13px;
  margin-right: 4px;
}

.dashboard-loading {
  padding: 44px 0 20px;
  color: var(--ds-color-text-tertiary);
  font-size: 13px;
  text-align: center;
}

.dashboard-empty-state {
  padding: 38px 18px 24px;
  color: var(--ds-color-text-tertiary);
  text-align: center;
}

.dashboard-empty-icon {
  display: grid;
  width: 38px;
  height: 38px;
  margin: 0 auto 12px;
  place-items: center;
  color: var(--ds-color-success);
  background: #edf8e9;
  border-radius: 50%;
}

.dashboard-empty-icon svg {
  width: 19px;
}

.dashboard-empty-state strong {
  color: var(--ds-color-text-primary);
  font-size: 13px;
}

.dashboard-empty-state p {
  margin: 7px 0 0;
  font-size: 12px;
}

.dashboard-more-hint {
  margin: 14px 0 0;
  padding-top: 13px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  border-top: 1px solid var(--ds-color-border);
}

.dashboard-more-hint a,
.dashboard-inline-link {
  color: var(--ds-color-primary);
  font-weight: 600;
}

.dashboard-subsection {
  margin-top: 22px;
}

.dashboard-subsection + .dashboard-subsection {
  padding-top: 20px;
  border-top: 1px solid var(--ds-color-border);
}

.dashboard-subheading {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--ds-color-text-primary);
  font-size: 13px;
}

.subheading-icon {
  width: 27px;
  height: 27px;
  border-radius: 8px;
}

.subheading-icon svg {
  width: 14px;
}

.subheading-count {
  margin-left: auto;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.recipe-mini-list,
.schedule-mini-list {
  margin: 12px 0 0 35px;
}

.recipe-mini-item,
.schedule-mini-item {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  min-height: 34px;
}

.recipe-food-dot,
.schedule-mini-dot {
  width: 7px;
  height: 7px;
  flex: 0 0 auto;
  margin-top: 5px;
  background: var(--ds-color-accent);
  border-radius: 50%;
}

.schedule-mini-dot {
  background: var(--ds-color-primary);
}

.schedule-mini-dot.is-overdue {
  background: var(--ds-color-danger);
}

.recipe-mini-item strong,
.schedule-mini-item strong {
  display: block;
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.recipe-mini-item small,
.schedule-mini-item small {
  display: block;
  margin-top: 2px;
  overflow: hidden;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-inline-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: 10px 0 0 35px;
  font-size: 11px;
}

.dashboard-inline-link svg {
  width: 13px;
}

.dashboard-subsection-empty {
  margin: 12px 0 0 35px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.doubao-personal-note {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  margin-top: 20px;
  padding: 10px 11px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  line-height: 1.55;
  background: #f7f8fb;
  border-radius: 9px;
}

.doubao-personal-note svg {
  width: 14px;
  flex: 0 0 auto;
  margin-top: 1px;
  color: var(--ds-color-success);
}

.album-mini-summary {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 5px 8px;
  margin-top: 16px;
  padding-top: 14px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  border-top: 1px solid var(--ds-color-border);
}

.album-mini-title {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--ds-color-text-primary);
  font-weight: 600;
}

.album-mini-title svg {
  width: 14px;
  color: var(--ds-color-primary);
}

.album-mini-summary .dashboard-inline-link {
  margin: 0 0 0 auto;
}

.dashboard-secondary-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.mini-module-panel {
  min-height: 248px;
}

.mini-item-list {
  margin-top: 18px;
}

.mini-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 11px 0;
  border-bottom: 1px solid #eff1f6;
}

.mini-item:last-child {
  border-bottom: 0;
}

.mini-item-icon {
  width: 30px;
  height: 30px;
  border-radius: 9px;
}

.mini-item-icon svg {
  width: 15px;
}

.mini-item-main {
  min-width: 0;
  flex: 1;
}

.mini-item-main strong {
  display: block;
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-item-main p {
  display: -webkit-box;
  margin: 4px 0 0;
  overflow: hidden;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.mini-item-main small {
  display: block;
  margin-top: 4px;
  color: #9ca5b5;
  font-size: 10px;
}

.mini-item-title-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.mini-item-title-line small {
  flex: 0 0 auto;
  margin-top: 0;
  color: var(--ds-color-text-tertiary);
}

.mini-progress {
  height: 4px;
  margin-top: 7px;
  overflow: hidden;
  background: #edf0f5;
  border-radius: 4px;
}

.mini-progress span {
  display: block;
  height: 100%;
  background: #be6681;
  border-radius: inherit;
}

.quick-entry-panel {
  margin-bottom: 18px;
}

.quick-link-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-top: 18px;
}

.quick-link {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--ds-color-border);
  border-radius: 11px;
  transition: border-color 160ms ease, background 160ms ease;
}

.quick-link:hover {
  background: #fbfcfe;
  border-color: #cbd5e7;
}

.quick-link-icon {
  width: 30px;
  height: 30px;
  border-radius: 9px;
}

.quick-link-icon svg {
  width: 15px;
}

.quick-link-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 3px;
}

.quick-link-copy strong {
  color: var(--ds-color-text-primary);
  font-size: 12px;
}

.quick-link-copy small {
  overflow: hidden;
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.quick-link-arrow {
  width: 14px;
  flex: 0 0 auto;
  color: #c5ccda;
}

.dashboard-health-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 14px 18px;
}

.dashboard-health-copy {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 10px;
}

.dashboard-health-icon {
  display: grid;
  width: 30px;
  height: 30px;
  flex: 0 0 auto;
  place-items: center;
  color: var(--ds-color-success);
  background: #edf8e9;
  border-radius: 9px;
}

.dashboard-health-icon svg {
  width: 16px;
}

.dashboard-health-icon.is-error {
  color: var(--ds-color-danger);
  background: #fff0f0;
}

.dashboard-health-copy strong,
.dashboard-health-copy span {
  display: block;
}

.dashboard-health-copy strong {
  color: var(--ds-color-text-primary);
  font-size: 12px;
}

.dashboard-health-copy span {
  margin-top: 2px;
  overflow: hidden;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1120px) {
  .dashboard-stat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dashboard-main-grid {
    grid-template-columns: minmax(0, 1fr) minmax(310px, 0.8fr);
  }
}

@media (max-width: 840px) {
  .dashboard-heading,
  .dashboard-main-grid,
  .dashboard-secondary-grid {
    grid-template-columns: 1fr;
  }

  .dashboard-heading {
    display: block;
  }

  .dashboard-heading-actions {
    margin-top: 16px;
  }

  .dashboard-main-grid,
  .dashboard-secondary-grid {
    display: grid;
  }

  .quick-link-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 560px) {
  .dashboard-stat-grid,
  .quick-link-grid {
    grid-template-columns: 1fr;
  }

  .dashboard-heading-actions {
    flex-wrap: wrap;
  }

  .dashboard-service-pill {
    width: 100%;
  }

  .dashboard-panel {
    padding: 18px;
  }

  .dashboard-event {
    grid-template-columns: 36px 3px minmax(0, 1fr);
    gap: 9px;
  }

  .dashboard-event-title-line {
    display: block;
  }

  .dashboard-event-status {
    display: block;
    margin-top: 3px;
  }
}
</style>
