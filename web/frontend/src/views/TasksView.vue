<script setup>
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Expand,
  Filter,
  Fold,
  Plus,
  Search,
  Sort,
} from '@element-plus/icons-vue'
import TaskRow from '../components/tasks/TaskRow.vue'

const router = useRouter()

const filters = [
  { key: 'all', label: '全部任务' },
  { key: 'today', label: '今天到期' },
  { key: 'high', label: '高优先级' },
  { key: 'completed', label: '已完成' },
]

const taskTree = ref([
  {
    id: 1,
    title: '整理个人年度规划',
    completed: false,
    due: '今天 18:00',
    dueTone: 'danger',
    priority: 'high',
    progress: '2 / 4',
    tags: ['生活', '重点'],
    children: [
      {
        id: 11,
        title: '回顾上半年目标完成情况',
        completed: true,
        due: '今天 12:00',
        priority: 'medium',
        tags: ['复盘'],
        children: [],
      },
      {
        id: 12,
        title: '确定下半年的三个关键词',
        completed: false,
        due: '今天 16:00',
        dueTone: 'danger',
        priority: 'high',
        tags: ['规划'],
        children: [],
      },
      {
        id: 13,
        title: '整理到可执行的月度计划',
        completed: false,
        due: '明天 18:00',
        priority: 'medium',
        tags: ['规划'],
        children: [],
      },
      {
        id: 14,
        title: '和自己确认最终版本',
        completed: true,
        due: '周五 18:00',
        priority: 'low',
        tags: [],
        children: [],
      },
    ],
  },
  {
    id: 2,
    title: '准备周末的朋友聚会',
    completed: false,
    due: '周六 14:00',
    priority: 'medium',
    progress: '1 / 3',
    tags: ['生活', '社交'],
    children: [
      {
        id: 21,
        title: '确认聚会地点和人数',
        completed: true,
        due: '周四 20:00',
        priority: 'medium',
        tags: [],
        children: [],
      },
      {
        id: 22,
        title: '准备轻食和饮料清单',
        completed: false,
        due: '周五 12:00',
        priority: 'low',
        tags: ['采买'],
        children: [],
      },
      {
        id: 23,
        title: '收拾客厅和阳台',
        completed: false,
        due: '周六 10:00',
        priority: 'medium',
        tags: [],
        children: [],
      },
    ],
  },
  {
    id: 3,
    title: '阅读《设计中的设计》',
    completed: false,
    due: '下周一 21:00',
    priority: 'low',
    progress: '3 / 8',
    tags: ['阅读', '长期'],
    children: [],
  },
  {
    id: 4,
    title: '整理旧硬盘照片',
    completed: true,
    due: '9 月 20 日',
    priority: 'low',
    tags: ['归档'],
    children: [],
  },
])

const searchQuery = ref('')
const activeFilter = ref('all')
const sortKey = ref('due')
const showCompleted = ref(false)
const expandedIds = ref(new Set([1, 2]))
const dialogVisible = ref(false)
const newTaskTitle = ref('')
const newTaskDue = ref('')
const newTaskPriority = ref('medium')

const summary = computed(() => {
  const allTasks = flattenTasks(taskTree.value)
  const incomplete = allTasks.filter((task) => !task.completed)
  return {
    total: incomplete.length,
    today: incomplete.filter((task) => task.due?.includes('今天')).length,
    high: incomplete.filter((task) => ['urgent', 'high'].includes(task.priority)).length,
  }
})

const visibleTaskTree = computed(() => filterNodes(taskTree.value))

function flattenTasks(nodes) {
  return nodes.flatMap((task) => [task, ...flattenTasks(task.children || [])])
}

function taskMatches(task) {
  const query = searchQuery.value.trim().toLowerCase()
  const keywordMatches = !query || task.title.toLowerCase().includes(query)

  if (activeFilter.value === 'completed') {
    return task.completed && keywordMatches
  }
  if (activeFilter.value === 'today') {
    return !task.completed && task.due?.includes('今天') && keywordMatches
  }
  if (activeFilter.value === 'high') {
    return !task.completed && ['urgent', 'high'].includes(task.priority) && keywordMatches
  }
  return keywordMatches && (showCompleted.value || !task.completed)
}

function filterNodes(nodes) {
  return nodes.reduce((result, task) => {
    if (activeFilter.value !== 'completed' && !showCompleted.value && task.completed) {
      return result
    }

    const children = filterNodes(task.children || [])
    if (taskMatches(task) || children.length > 0) {
      result.push({ ...task, children })
    }
    return result
  }, [])
}

function findTask(id, nodes = taskTree.value) {
  for (const task of nodes) {
    if (task.id === id) return task
    const found = findTask(id, task.children || [])
    if (found) return found
  }
  return null
}

function collectDescendants(task) {
  return flattenTasks(task.children || [])
}

function toggleExpanded(id) {
  const next = new Set(expandedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedIds.value = next
}

function expandAll() {
  expandedIds.value = new Set(flattenTasks(taskTree.value).filter((task) => task.children?.length).map((task) => task.id))
}

function collapseAll() {
  expandedIds.value = new Set()
}

function setFilter(key) {
  activeFilter.value = key
  if (key === 'completed') showCompleted.value = true
}

function openTask(task) {
  router.push({ name: 'task-detail', params: { id: task.id } })
}

async function completeTask(task) {
  if (task.completed) {
    ElMessage.info('已完成任务不可恢复。')
    return
  }

  const pending = collectDescendants(task).filter((item) => !item.completed)
  if (pending.length > 0) {
    await ElMessageBox.alert(
      `请先完成以下任务：${pending.slice(0, 4).map((item) => item.title).join('、')}${pending.length > 4 ? '等' : ''}`,
      '暂时无法完成',
      { type: 'warning', confirmButtonText: '知道了' },
    )
    return
  }

  try {
    await ElMessageBox.confirm('完成后将不能恢复或修改这个任务，确认继续吗？', '标记为已完成', {
      type: 'info',
      confirmButtonText: '确认完成',
      cancelButtonText: '再想想',
    })
    const source = findTask(task.id)
    if (source) source.completed = true
    ElMessage.success('任务已完成。')
  } catch {
    // 用户取消确认时保持当前状态。
  }
}

async function handleCommand({ command, task }) {
  if (command === 'open') {
    openTask(task)
    return
  }
  if (command === 'add-child') {
    ElMessage.info('子任务表单将在任务 API 接入后开放。')
    return
  }
  if (command === 'move') {
    ElMessage.info('移动任务会在后端任务树规则接入后开放。')
    return
  }
  if (command === 'delete') {
    try {
      await ElMessageBox.confirm(
        `删除后将永久删除“${task.title}”及其全部子任务，且无法恢复。`,
        '删除任务',
        {
          type: 'warning',
          confirmButtonText: '永久删除',
          cancelButtonText: '取消',
          confirmButtonClass: 'el-button--danger',
        },
      )
      removeTask(task.id)
      ElMessage.success('任务已删除。')
    } catch {
      // 用户取消删除时保持当前状态。
    }
  }
}

function removeTask(id, nodes = taskTree.value) {
  const index = nodes.findIndex((task) => task.id === id)
  if (index >= 0) {
    nodes.splice(index, 1)
    return true
  }
  return nodes.some((task) => removeTask(id, task.children || []))
}

function createTask() {
  const title = newTaskTitle.value.trim()
  if (!title) {
    ElMessage.warning('请先填写任务标题。')
    return
  }

  taskTree.value.unshift({
    id: Date.now(),
    title,
    completed: false,
    due: newTaskDue.value.trim() || '无截止时间',
    priority: newTaskPriority.value,
    tags: [],
    children: [],
  })
  dialogVisible.value = false
  newTaskTitle.value = ''
  newTaskDue.value = ''
  newTaskPriority.value = 'medium'
  ElMessage.success('任务已添加到当前页面。')
}
</script>

<template>
  <div class="page-view page-view-wide tasks-page">
    <section class="page-heading">
      <div class="page-heading-copy">
        <p class="eyebrow">TASK SPACE</p>
        <h1>任务</h1>
        <p>把复杂的事情拆成清晰的下一步，今天先完成最重要的一件。</p>
      </div>
      <div class="page-heading-actions">
        <el-button class="soft-button" @click="expandAll"><Expand class="button-icon" />展开全部</el-button>
        <el-button type="primary" @click="dialogVisible = true"><Plus class="button-icon" />新建任务</el-button>
      </div>
    </section>

    <section class="task-summary-grid">
      <div class="task-summary-card">
        <div class="summary-kicker">未完成任务</div>
        <div class="summary-value">{{ summary.total }}</div>
        <div class="summary-note">保持节奏，一步一步来</div>
      </div>
      <div class="task-summary-card is-warm">
        <div class="summary-kicker">今天到期</div>
        <div class="summary-value">{{ summary.today }}</div>
        <div class="summary-note">优先处理有明确时间的事</div>
      </div>
      <div class="task-summary-card is-blue">
        <div class="summary-kicker">高优先级</div>
        <div class="summary-value">{{ summary.high }}</div>
        <div class="summary-note">减少同时进行的事情</div>
      </div>
    </section>

    <section class="task-panel surface-card">
      <div class="task-panel-header">
        <div>
          <div class="card-heading">
            <div>
              <h2>我的任务树</h2>
              <p>父任务和子任务保持清晰的上下文关系。</p>
            </div>
          </div>
        </div>
        <div class="task-panel-actions">
          <el-button text class="panel-action" @click="collapseAll"><Fold class="button-icon" />折叠</el-button>
          <el-button text class="panel-action" @click="expandAll"><Expand class="button-icon" />展开</el-button>
        </div>
      </div>

      <div class="task-toolbar">
        <div class="task-filter-tabs" role="tablist" aria-label="任务筛选">
          <button
            v-for="filter in filters"
            :key="filter.key"
            class="filter-tab"
            :class="{ 'is-active': activeFilter === filter.key }"
            type="button"
            @click="setFilter(filter.key)"
          >
            {{ filter.label }}
          </button>
        </div>

        <div class="task-toolbar-controls">
          <el-input v-model="searchQuery" class="task-search" placeholder="搜索任务标题" clearable>
            <template #prefix><Search /></template>
          </el-input>
          <el-select v-model="sortKey" class="task-sort" aria-label="任务排序">
            <template #prefix><Sort /></template>
            <el-option label="按截止时间" value="due" />
            <el-option label="按优先级" value="priority" />
            <el-option label="按创建时间" value="created" />
          </el-select>
          <el-switch v-model="showCompleted" inline-prompt active-text="已完成" inactive-text="隐藏" />
          <el-button class="filter-button" circle aria-label="筛选"><Filter /></el-button>
        </div>
      </div>

      <div class="task-list-head">
        <span>任务内容</span>
        <span>优先级</span>
        <span>操作</span>
      </div>

      <div v-if="visibleTaskTree.length > 0" class="task-tree-list">
        <TaskRow
          v-for="task in visibleTaskTree"
          :key="task.id"
          :task="task"
          :expanded-ids="expandedIds"
          :show-completed="showCompleted"
          @toggle="toggleExpanded"
          @open="openTask"
          @complete="completeTask"
          @command="handleCommand"
        />
      </div>

      <div v-else class="task-empty-state">
        <div class="task-empty-mark"><Search /></div>
        <h3>没有找到匹配的任务</h3>
        <p>换一个关键词，或调整筛选条件试试。</p>
        <el-button text type="primary" @click="searchQuery = ''; activeFilter = 'all'">清除筛选</el-button>
      </div>

      <div class="task-panel-footer">
        <span><Filter class="footer-icon" />当前显示 {{ visibleTaskTree.length }} 个顶层任务</span>
        <span class="demo-disclaimer">界面预览数据 · 尚未连接任务 API</span>
      </div>
    </section>

    <el-dialog v-model="dialogVisible" title="新建任务" width="620px" destroy-on-close>
      <div class="dialog-lead">先记下来，不需要一次就想得很完整。</div>
      <el-form label-position="top" class="task-create-form" @submit.prevent="createTask">
        <el-form-item label="任务标题" required>
          <el-input v-model="newTaskTitle" autofocus placeholder="例如：准备周末的朋友聚会" @keyup.enter="createTask" />
        </el-form-item>
        <div class="form-grid-two">
          <el-form-item label="截止时间">
            <el-input v-model="newTaskDue" placeholder="例如：周五 18:00" />
          </el-form-item>
          <el-form-item label="优先级">
            <el-select v-model="newTaskPriority" class="full-width">
              <el-option label="紧急" value="urgent" />
              <el-option label="高优先级" value="high" />
              <el-option label="中优先级" value="medium" />
              <el-option label="低优先级" value="low" />
            </el-select>
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <div class="dialog-footer-actions">
          <span>当前仅保留在页面内</span>
          <div>
            <el-button @click="dialogVisible = false">取消</el-button>
            <el-button type="primary" @click="createTask">添加任务</el-button>
          </div>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.tasks-page {
  max-width: 1440px;
}

.task-summary-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 22px;
}

.task-summary-card {
  position: relative;
  min-height: 132px;
  padding: 22px 24px;
  overflow: hidden;
  background: #ffffff;
  border: 1px solid var(--ds-color-border);
  border-radius: var(--ds-radius-lg);
}

.task-summary-card::after {
  position: absolute;
  right: -20px;
  bottom: -35px;
  width: 120px;
  height: 120px;
  content: '';
  background: rgba(82, 107, 155, 0.08);
  border-radius: 50%;
}

.task-summary-card.is-warm::after {
  background: rgba(247, 200, 115, 0.18);
}

.task-summary-card.is-blue::after {
  background: rgba(94, 175, 76, 0.1);
}

.summary-kicker {
  color: var(--ds-color-text-tertiary);
  font-size: 13px;
}

.summary-value {
  position: relative;
  z-index: 1;
  margin-top: 7px;
  color: var(--ds-color-text-primary);
  font-size: 31px;
  font-weight: 700;
  letter-spacing: -0.05em;
  line-height: 1;
}

.summary-note {
  position: relative;
  z-index: 1;
  margin-top: 12px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.task-panel {
  overflow: hidden;
}

.task-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 22px 24px 18px;
}

.task-panel-actions {
  display: flex;
  gap: 4px;
}

.panel-action.el-button {
  color: var(--ds-color-text-tertiary);
}

.panel-action.el-button:hover {
  color: var(--ds-color-primary);
}

.task-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 12px 24px;
  background: var(--ds-color-surface-muted);
  border-top: 1px solid var(--ds-color-border);
  border-bottom: 1px solid var(--ds-color-border);
}

.task-filter-tabs {
  display: flex;
  align-items: center;
  gap: 3px;
}

.filter-tab {
  padding: 8px 11px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  cursor: pointer;
  background: transparent;
  border-radius: 8px;
  transition: color 160ms ease, background-color 160ms ease;
}

.filter-tab:hover {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.filter-tab.is-active {
  color: var(--ds-color-primary);
  font-weight: 700;
  background: #ffffff;
  box-shadow: 0 2px 7px rgba(37, 48, 71, 0.06);
}

.task-toolbar-controls {
  display: flex;
  align-items: center;
  gap: 9px;
}

.task-search {
  width: 210px;
}

.task-sort {
  width: 142px;
}

.filter-button.el-button {
  color: var(--ds-color-text-tertiary);
  border-color: var(--ds-color-border);
}

.task-list-head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 100px 42px;
  gap: 12px;
  padding: 14px 24px 10px 62px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  letter-spacing: 0.04em;
}

.task-list-head span:nth-child(2) {
  text-align: center;
}

.task-list-head span:last-child {
  text-align: right;
}

.task-tree-list {
  padding: 0 14px 8px;
}

.task-tree-node + .task-tree-node {
  border-top: 1px solid #f0f2f7;
}

.task-row {
  display: grid;
  grid-template-columns: 24px 26px minmax(0, 1fr) 100px 34px;
  align-items: center;
  min-height: 68px;
  gap: 10px;
  padding: 8px 10px 8px calc(10px + var(--task-depth) * 27px);
  border-radius: 10px;
  transition: background-color 160ms ease;
}

.task-row:hover,
.task-row:focus-visible {
  background: #f7f9fd;
}

.task-expand-button,
.task-check-button,
.task-more-button {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  color: var(--ds-color-text-tertiary);
  cursor: pointer;
  background: transparent;
  border-radius: 8px;
}

.task-expand-button:hover,
.task-more-button:hover {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.task-expand-button:disabled {
  cursor: default;
}

.task-expand-button svg,
.task-more-button svg {
  width: 16px;
}

.task-expand-placeholder {
  width: 6px;
  height: 6px;
  background: #cbd2e0;
  border-radius: 50%;
}

.task-check-button {
  color: var(--ds-color-primary);
}

.task-check-button:hover {
  color: var(--ds-color-primary-hover);
}

.task-check-button svg {
  width: 20px;
  height: 20px;
}

.task-check-ring {
  display: block;
  width: 18px;
  height: 18px;
  border: 1.5px solid #bdc7d8;
  border-radius: 50%;
}

.task-row-content {
  min-width: 0;
  cursor: pointer;
}

.task-title-line {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 7px;
}

.task-title {
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-row.is-complete .task-title {
  color: var(--ds-color-text-disabled);
  text-decoration: line-through;
}

.task-tag {
  flex: 0 0 auto;
  padding: 3px 7px;
  color: var(--ds-color-primary);
  font-size: 11px;
  background: var(--ds-color-primary-light);
  border-radius: 5px;
}

.task-tag-more {
  color: var(--ds-color-text-tertiary);
  background: #f0f2f6;
}

.task-meta-line {
  display: flex;
  align-items: center;
  gap: 13px;
  margin-top: 6px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.task-due,
.task-progress {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.task-due svg {
  width: 13px;
}

.task-due.is-overdue {
  color: var(--ds-color-danger);
}

.task-progress {
  padding-left: 13px;
  border-left: 1px solid var(--ds-color-border);
}

.task-priority {
  justify-self: center;
  padding: 5px 8px;
  font-size: 11px;
  white-space: nowrap;
  border-radius: 6px;
}

.task-priority.is-urgent,
.task-priority.is-high {
  color: #c0525b;
  background: #fff0f0;
}

.task-priority.is-medium {
  color: #9b6e24;
  background: #fff8e8;
}

.task-priority.is-low {
  color: var(--ds-color-text-tertiary);
  background: #f1f3f7;
}

.task-more-button {
  justify-self: end;
}

.task-children {
  position: relative;
}

.task-children::before {
  position: absolute;
  top: 0;
  bottom: 22px;
  left: 39px;
  width: 1px;
  content: '';
  background: #e8ecf4;
}

.task-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 72px 20px 84px;
  text-align: center;
}

.task-empty-mark {
  display: grid;
  width: 58px;
  height: 58px;
  place-items: center;
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
  border-radius: 18px;
}

.task-empty-mark svg {
  width: 24px;
}

.task-empty-state h3 {
  margin: 18px 0 4px;
  font-size: 16px;
}

.task-empty-state p {
  margin: 0;
  color: var(--ds-color-text-tertiary);
}

.task-panel-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 14px 24px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  background: var(--ds-color-surface-muted);
  border-top: 1px solid var(--ds-color-border);
}

.task-panel-footer > span:first-child {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.footer-icon {
  width: 14px;
}

.demo-disclaimer {
  color: #a0aabc;
}

.dialog-lead {
  margin-bottom: 20px;
  color: var(--ds-color-text-secondary);
}

.task-create-form :deep(.el-form-item) {
  margin-bottom: 18px;
}

.form-grid-two {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.full-width {
  width: 100%;
}

.dialog-footer-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.dialog-footer-actions > span {
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

@media (max-width: 1120px) {
  .task-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .task-toolbar-controls {
    width: 100%;
  }

  .task-search {
    flex: 1;
  }
}
</style>
