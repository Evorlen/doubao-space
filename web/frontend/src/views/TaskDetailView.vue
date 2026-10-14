<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowLeft,
  Calendar,
  CircleCheck,
  Delete,
  Edit,
  Flag,
  Link,
  MoreFilled,
  Paperclip,
  Upload,
} from '@element-plus/icons-vue'
import {
  completeTask as completeTaskRequest,
  createTask as createTaskRequest,
  deleteAttachment as deleteAttachmentRequest,
  deleteTask as deleteTaskRequest,
  getTask,
  scheduleTask as scheduleTaskRequest,
  unscheduleTask as unscheduleTaskRequest,
  uploadAttachment as uploadAttachmentRequest,
  updateTask as updateTaskRequest,
} from '../api'
import TaskImportDialog from '../components/tasks/TaskImportDialog.vue'

const route = useRoute()
const router = useRouter()
const task = ref(null)
const loading = ref(true)
const errorMessage = ref('')
const saving = ref(false)
const scheduleSaving = ref(false)
const deleting = ref(false)
const childDialogVisible = ref(false)
const childTitle = ref('')
const childDue = ref('')
const childPriority = ref('medium')
const childSaving = ref(false)
const importDialogVisible = ref(false)
const uploading = ref(false)
const fileInput = ref(null)
const form = ref(createEmptyForm())
const initialForm = ref('')

const isDirty = computed(() => JSON.stringify(form.value) !== initialForm.value)
const isReadOnly = computed(() => Boolean(task.value?.completed))
const dueLabel = computed(() => task.value?.due_at ? formatDue(new Date(task.value.due_at)) : '无截止时间')
const dueTone = computed(() => task.value?.due_at && !task.value.completed && new Date(task.value.due_at).getTime() < Date.now() ? 'is-danger' : '')
const progressStats = computed(() => {
  const total = Math.max(0, Number(task.value?.progress?.total) || 0)
  const completed = Math.min(total, Math.max(0, Number(task.value?.progress?.completed) || 0))
  return {
    completed,
    total,
    remaining: total - completed,
    percent: total ? Math.round((completed / total) * 100) : 0,
  }
})
const hasSubtaskProgress = computed(() => progressStats.value.total > 0)
const progressSummary = computed(() => {
  const { completed, total, remaining } = progressStats.value
  if (!total) return '暂无子任务'
  if (!remaining) return `已完成全部 ${total} 个子任务`
  return `已完成 ${completed} 个，共 ${total} 个子任务 · 剩余 ${remaining} 个`
})
const progressPropertyLabel = computed(() => {
  if (!hasSubtaskProgress.value) return '暂无子任务'
  const { completed, total, percent } = progressStats.value
  return `${completed}/${total}（${percent}%）`
})

function createEmptyForm() {
  return {
    title: '',
    description: '',
    due_at: '',
    priority: 'medium',
    tags: [],
    links: [],
  }
}

async function loadTask() {
  loading.value = true
  errorMessage.value = ''
  try {
    const data = await getTask(route.params.id)
    task.value = data
    form.value = {
      title: data.title || '',
      description: data.description || '',
      due_at: data.due_at ? formatPickerDate(new Date(data.due_at)) : '',
      priority: data.priority || 'medium',
      tags: [...(data.tags || [])],
      links: (data.links || []).map((link) => ({ title: link.title || '', url: link.url })),
    }
    initialForm.value = JSON.stringify(form.value)
  } catch (error) {
    task.value = null
    errorMessage.value = error.message
  } finally {
    loading.value = false
  }
}

function formatPickerDate(value) {
  const pad = (number) => String(number).padStart(2, '0')
  return `${value.getFullYear()}-${pad(value.getMonth() + 1)}-${pad(value.getDate())} ${pad(value.getHours())}:${pad(value.getMinutes())}:00`
}

function formatDue(value) {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(value)
}

function formatSize(size) {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / (1024 * 1024)).toFixed(1)} MB`
}

function priorityLabel(priority) {
  return {
    urgent: '紧急',
    high: '高优先级',
    medium: '中优先级',
    low: '低优先级',
  }[priority] || '中优先级'
}

function addLink() {
  form.value.links.push({ title: '', url: '' })
}

function removeLink(index) {
  form.value.links.splice(index, 1)
}

function chooseAttachment() {
  fileInput.value?.click()
}

async function handleAttachmentChange(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file || !task.value || isReadOnly.value) return
  uploading.value = true
  try {
    await uploadAttachmentRequest(task.value.id, file)
    await loadTask()
    ElMessage.success('附件已上传。')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    uploading.value = false
  }
}

async function removeAttachment(attachment) {
  try {
    await ElMessageBox.confirm(`确定删除附件“${attachment.name}”吗？删除后无法恢复。`, '删除附件', {
      type: 'warning',
      confirmButtonText: '删除附件',
      cancelButtonText: '取消',
    })
    await deleteAttachmentRequest(task.value.id, attachment.id)
    await loadTask()
    ElMessage.success('附件已删除。')
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') ElMessage.error(error.message)
  }
}

function goToTask(id) {
  router.push({ name: 'task-detail', params: { id } })
}

async function goBack() {
  if (isDirty.value && !isReadOnly.value) {
    try {
      await ElMessageBox.confirm('当前任务还有未保存的修改，确定离开吗？', '放弃修改', {
        type: 'warning',
        confirmButtonText: '离开',
        cancelButtonText: '继续编辑',
      })
    } catch {
      return
    }
  }
  router.back()
}

async function saveTask() {
  if (!task.value || isReadOnly.value) return
  if (!form.value.title.trim()) {
    ElMessage.warning('任务标题不能为空。')
    return
  }
  saving.value = true
  try {
    const data = await updateTaskRequest(task.value.id, {
      title: form.value.title.trim(),
      description: form.value.description.trim(),
      due_at: form.value.due_at,
      priority: form.value.priority,
      tags: form.value.tags,
      links: form.value.links.filter((link) => link.url.trim()),
    })
    task.value = data
    form.value.links = (data.links || []).map((link) => ({ title: link.title || '', url: link.url }))
    initialForm.value = JSON.stringify(form.value)
    ElMessage.success('任务已保存。')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    saving.value = false
  }
}

function cancelChanges() {
  if (!task.value) return
  form.value = {
    title: task.value.title || '',
    description: task.value.description || '',
    due_at: task.value.due_at ? formatPickerDate(new Date(task.value.due_at)) : '',
    priority: task.value.priority || 'medium',
    tags: [...(task.value.tags || [])],
    links: (task.value.links || []).map((link) => ({ title: link.title || '', url: link.url })),
  }
}

async function completeTask() {
  if (!task.value || task.value.completed) return
  try {
    await ElMessageBox.confirm('完成后将不能恢复或修改这个任务，确认继续吗？', '标记为已完成', {
      type: 'info',
      confirmButtonText: '确认完成',
      cancelButtonText: '再想想',
    })
    task.value = await completeTaskRequest(task.value.id)
    form.value = { ...form.value }
    initialForm.value = JSON.stringify(form.value)
    ElMessage.success('任务已完成。')
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') ElMessage.error(error.message)
  }
}

async function toggleSchedule() {
  if (!task.value || isReadOnly.value || scheduleSaving.value) return
  scheduleSaving.value = true
  try {
    task.value = task.value.scheduled
      ? await unscheduleTaskRequest(task.value.id)
      : await scheduleTaskRequest(task.value.id)
    ElMessage.success(task.value.scheduled ? '任务已提上日程。' : '任务已移出日程。')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    scheduleSaving.value = false
  }
}

async function deleteTask() {
  if (!task.value) return
  try {
    await ElMessageBox.confirm(
      `删除后将永久删除“${task.value.title}”及其全部子任务，且无法恢复。`,
      '删除任务',
      { type: 'warning', confirmButtonText: '永久删除', cancelButtonText: '取消', confirmButtonClass: 'el-button--danger' },
    )
    deleting.value = true
    await deleteTaskRequest(task.value.id)
    ElMessage.success('任务已删除。')
    router.replace({ name: 'tasks' })
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') ElMessage.error(error.message)
  } finally {
    deleting.value = false
  }
}

function openChildDialog() {
  childTitle.value = ''
  childDue.value = task.value?.due_at ? formatPickerDate(new Date(task.value.due_at)) : ''
  childPriority.value = 'medium'
  childDialogVisible.value = true
}

function openImportDialog() {
  if (!task.value || isReadOnly.value || task.value.depth >= 5) return
  importDialogVisible.value = true
}

async function handleImported() {
  await loadTask()
}

async function createChild() {
  if (!task.value || !childTitle.value.trim()) {
    ElMessage.warning('请先填写子任务标题。')
    return
  }
  childSaving.value = true
  try {
    await createTaskRequest({
      parent_id: task.value.id,
      title: childTitle.value.trim(),
      description: '',
      due_at: childDue.value,
      priority: childPriority.value,
      tags: [],
      links: [],
    })
    childDialogVisible.value = false
    await loadTask()
    ElMessage.success('子任务已添加。')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    childSaving.value = false
  }
}

watch(() => route.params.id, loadTask)
onMounted(loadTask)
</script>

<template>
  <div class="page-view task-detail-page">
    <div v-if="loading" class="detail-loading-state">正在加载任务…</div>
    <div v-else-if="errorMessage" class="detail-error-state">
      <h2>任务暂时无法打开</h2>
      <p>{{ errorMessage }}</p>
      <el-button type="primary" @click="loadTask">重新加载</el-button>
    </div>

    <template v-else-if="task">
      <button class="detail-back-button" type="button" @click="goBack"><ArrowLeft />返回任务</button>

      <section class="detail-heading">
        <div class="detail-heading-copy">
          <div class="detail-path">{{ task.path?.slice(0, -1).join(' / ') || '顶层任务' }}</div>
          <h1>{{ task.title }}</h1>
          <div class="detail-heading-meta">
            <span class="detail-status" :class="{ 'is-complete': task.completed }"><span></span>{{ task.completed ? '已完成' : '未完成' }}</span>
            <span>更新于 {{ formatDue(new Date(task.updated_at)) }}</span>
          </div>
        </div>
        <div class="page-heading-actions">
          <el-button v-if="!isReadOnly" type="primary" :loading="saving" @click="saveTask"><Edit class="button-icon" />保存修改</el-button>
          <el-button v-if="!isReadOnly && isDirty" class="soft-button" @click="cancelChanges">取消修改</el-button>
          <el-button v-if="!isReadOnly" class="soft-button" @click="completeTask"><CircleCheck class="button-icon" />标记完成</el-button>
          <el-button v-if="!isReadOnly" class="soft-button" :loading="scheduleSaving" @click="toggleSchedule"><Calendar class="button-icon" />{{ task.scheduled ? '移出日程' : '提上日程' }}</el-button>
          <el-button circle :loading="deleting" aria-label="删除任务" @click="deleteTask"><Delete /></el-button>
        </div>
      </section>

      <section class="detail-layout">
        <div class="detail-main-column">
          <div class="surface-card detail-description-card">
            <div class="detail-section-heading">
              <h2>任务内容</h2>
              <span class="demo-badge">{{ isReadOnly ? '只读' : '可编辑' }}</span>
            </div>
            <el-form label-position="top" class="detail-form" :disabled="isReadOnly">
              <el-form-item label="任务标题" required>
                <el-input v-model="form.title" />
              </el-form-item>
              <el-form-item label="任务说明">
                <el-input v-model="form.description" type="textarea" :rows="5" placeholder="补充任务背景和完成标准" />
              </el-form-item>
              <div class="form-grid-two">
                <el-form-item label="截止时间">
                  <el-date-picker
                    v-model="form.due_at"
                    type="datetime"
                    value-format="YYYY-MM-DD HH:mm:ss"
                    format="YYYY-MM-DD HH:mm"
                    class="full-width"
                    clearable
                    placeholder="无截止时间"
                  />
                </el-form-item>
                <el-form-item label="优先级">
                  <el-select v-model="form.priority" class="full-width">
                    <el-option label="紧急" value="urgent" />
                    <el-option label="高优先级" value="high" />
                    <el-option label="中优先级" value="medium" />
                    <el-option label="低优先级" value="low" />
                  </el-select>
                </el-form-item>
              </div>
              <el-form-item label="标签">
                <el-select v-model="form.tags" class="full-width" multiple filterable allow-create default-first-option placeholder="添加标签">
                  <el-option v-for="tag in form.tags" :key="tag" :label="tag" :value="tag" />
                </el-select>
              </el-form-item>
            </el-form>
          </div>

          <div class="surface-card detail-subtasks-card">
            <div class="detail-section-heading">
              <div>
                <h2>子任务</h2>
                <p>{{ progressSummary }}</p>
              </div>
              <div class="detail-subtask-actions">
                <el-button text type="primary" :disabled="isReadOnly || task.depth >= 5" @click="openImportDialog">导入子任务</el-button>
                <el-button text type="primary" :disabled="isReadOnly || task.depth >= 5" @click="openChildDialog">添加子任务</el-button>
              </div>
            </div>
            <div v-if="hasSubtaskProgress" class="detail-progress-wrap">
              <div class="detail-progress-meta">
                <span>完成度</span>
                <strong>{{ progressStats.percent }}%</strong>
              </div>
              <div
                class="detail-progress-bar"
                role="progressbar"
                :aria-valuenow="progressStats.percent"
                aria-valuemin="0"
                aria-valuemax="100"
                :aria-label="progressSummary"
              >
                <span :style="{ width: `${progressStats.percent}%` }"></span>
              </div>
            </div>
            <div v-if="task.children?.length" class="subtask-list">
              <button
                v-for="subtask in task.children"
                :key="subtask.id"
                class="subtask-item"
                :class="{ 'is-complete': subtask.completed }"
                type="button"
                @click="goToTask(subtask.id)"
              >
                <span class="subtask-check"><CircleCheck v-if="subtask.completed" /></span>
                <span class="subtask-title">{{ subtask.title }}</span>
                <span class="subtask-due"><Calendar />{{ subtask.due_at ? formatDue(new Date(subtask.due_at)) : '无截止时间' }}</span>
                <MoreFilled class="subtask-more" />
              </button>
            </div>
            <div v-else class="detail-inline-empty">还没有子任务，可以把下一步拆出来。</div>
          </div>

          <div class="surface-card detail-description-card">
            <div class="detail-section-heading">
              <h2>附件与链接</h2>
              <div class="detail-asset-actions">
                <input ref="fileInput" class="hidden-file-input" type="file" @change="handleAttachmentChange" />
                <el-button text type="primary" :loading="uploading" :disabled="isReadOnly" @click="chooseAttachment"><Upload class="button-icon" />上传附件</el-button>
                <el-button text type="primary" :disabled="isReadOnly" @click="addLink"><Link class="button-icon" />添加链接</el-button>
              </div>
            </div>
            <div v-if="task.attachments?.length" class="detail-attachment-list">
              <div v-for="attachment in task.attachments" :key="attachment.id" class="detail-attachment-item">
                <Paperclip class="detail-attachment-icon" />
                <a :href="`/api/tasks/${task.id}/attachments/${attachment.id}`" target="_blank" rel="noreferrer" class="detail-attachment-name">{{ attachment.name }}</a>
                <span class="detail-attachment-size">{{ formatSize(attachment.size) }}</span>
                <el-button v-if="!isReadOnly" text type="danger" @click="removeAttachment(attachment)">移除</el-button>
              </div>
            </div>
            <div v-if="form.links.length" class="detail-link-list">
              <div v-for="(link, index) in form.links" :key="index" class="detail-link-item">
                <Link class="detail-link-icon" />
                <div class="detail-link-fields">
                  <el-input v-model="link.title" :disabled="isReadOnly" placeholder="链接名称" />
                  <el-input v-model="link.url" :disabled="isReadOnly" placeholder="https://example.com" />
                </div>
                <el-button v-if="!isReadOnly" text type="danger" @click="removeLink(index)">移除</el-button>
              </div>
            </div>
            <div v-if="!task.attachments?.length && !form.links.length" class="detail-assets-empty">
              <span class="asset-empty-icon"><Paperclip /></span>
              <div>
                <strong>还没有外部链接</strong>
                <p>可以把参考资料放在这里，之后更容易找到。</p>
              </div>
              <Link class="asset-empty-link" />
            </div>
          </div>
        </div>

        <aside class="detail-side-column">
          <div class="surface-card detail-property-card">
            <div class="detail-section-heading"><h2>任务属性</h2></div>
            <div class="property-list">
              <div class="property-row">
                <span class="property-label"><Calendar />日程状态</span>
                <strong class="property-value" :class="{ 'is-scheduled': task.scheduled }">{{ task.scheduled ? '已提上日程' : '未提上日程' }}</strong>
              </div>
              <div class="property-row">
                <span class="property-label"><Calendar />截止时间</span>
                <strong class="property-value" :class="dueTone">{{ dueLabel }}</strong>
              </div>
              <div class="property-row">
                <span class="property-label"><Flag />最终优先级</span>
                <strong class="property-value" :class="`is-${task.effective_priority || task.priority}`">{{ priorityLabel(task.effective_priority || task.priority) }}</strong>
              </div>
              <div class="property-row">
                <span class="property-label"><CircleCheck />完成进度</span>
                <strong class="property-value">{{ progressPropertyLabel }}</strong>
              </div>
            </div>
          </div>

          <div v-if="!isReadOnly" class="surface-card detail-note-card">
            <div class="detail-note-icon"><CircleCheck /></div>
            <h3>完成前先确认</h3>
            <p>所有子任务完成后，才可以将这个父任务标记为完成。</p>
          </div>

          <div class="detail-meta-card">
            <span>创建于 {{ formatDue(new Date(task.created_at)) }}</span>
            <span>任务 ID · #{{ String(task.id).padStart(4, '0') }}</span>
          </div>
        </aside>
      </section>

      <el-dialog v-model="childDialogVisible" title="添加子任务" width="560px" destroy-on-close>
        <div class="dialog-lead">子任务会继承当前任务树的期限规则。</div>
        <el-form label-position="top">
          <el-form-item label="任务标题" required>
            <el-input v-model="childTitle" autofocus placeholder="例如：整理参考资料" />
          </el-form-item>
          <div class="form-grid-two">
            <el-form-item label="截止时间">
              <el-date-picker
                v-model="childDue"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                format="YYYY-MM-DD HH:mm"
                class="full-width"
                clearable
                placeholder="无截止时间"
              />
            </el-form-item>
            <el-form-item label="优先级">
              <el-select v-model="childPriority" class="full-width">
                <el-option label="紧急" value="urgent" />
                <el-option label="高优先级" value="high" />
                <el-option label="中优先级" value="medium" />
                <el-option label="低优先级" value="low" />
              </el-select>
            </el-form-item>
          </div>
        </el-form>
        <template #footer>
          <el-button @click="childDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="childSaving" @click="createChild">保存子任务</el-button>
        </template>
      </el-dialog>

      <TaskImportDialog v-model="importDialogVisible" :task="task" @imported="handleImported" />
    </template>
  </div>
</template>

<style scoped>
.task-detail-page {
  max-width: 1120px;
}

.detail-back-button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  margin-bottom: 22px;
  padding: 5px 0;
  color: var(--ds-color-text-tertiary);
  font-size: 13px;
  cursor: pointer;
  background: transparent;
}

.detail-back-button:hover {
  color: var(--ds-color-primary);
}

.detail-back-button svg {
  width: 16px;
}

.detail-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 28px;
}

.detail-path {
  margin-bottom: 9px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.detail-heading h1 {
  margin: 0;
  color: var(--ds-color-text-primary);
  font-size: 30px;
  letter-spacing: -0.045em;
  line-height: 1.2;
}

.detail-heading-meta {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 12px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.detail-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--ds-color-primary);
  font-weight: 600;
}

.detail-status span {
  width: 7px;
  height: 7px;
  background: var(--ds-color-primary);
  border-radius: 50%;
}

.detail-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 300px;
  align-items: start;
  gap: 20px;
}

.detail-main-column,
.detail-side-column {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.detail-description-card,
.detail-subtasks-card,
.detail-property-card {
  padding: 24px;
}

.detail-section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.detail-section-heading h2 {
  margin: 0;
  color: var(--ds-color-text-primary);
  font-size: 16px;
}

.detail-asset-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.detail-subtask-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.hidden-file-input {
  display: none;
}

.detail-section-heading p {
  margin: 4px 0 0;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.demo-badge {
  padding: 4px 7px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  background: var(--ds-color-surface-muted);
  border-radius: 5px;
}

.detail-description {
  margin: 18px 0 0;
  color: var(--ds-color-text-secondary);
  line-height: 1.8;
}

.detail-form {
  margin-top: 20px;
}

.detail-form :deep(.el-form-item) {
  margin-bottom: 18px;
}

.detail-loading-state,
.detail-error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 360px;
  color: var(--ds-color-text-tertiary);
  text-align: center;
}

.detail-error-state h2 {
  margin: 0 0 6px;
  color: var(--ds-color-text-primary);
  font-size: 18px;
}

.detail-error-state p {
  margin: 0 0 16px;
}

.detail-tags {
  display: flex;
  gap: 7px;
  margin-top: 20px;
}

.task-tag {
  padding: 4px 8px;
  color: var(--ds-color-primary);
  font-size: 11px;
  background: var(--ds-color-primary-light);
  border-radius: 5px;
}

.detail-progress-bar {
  height: 8px;
  overflow: hidden;
  background: #eef1f6;
  border-radius: var(--ds-radius-pill);
}

.detail-progress-bar span {
  display: block;
  height: 100%;
  background: var(--ds-color-primary);
  border-radius: inherit;
  transition: width 180ms ease;
}

.detail-progress-wrap {
  margin-top: 19px;
}

.detail-progress-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 7px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.detail-progress-meta strong {
  color: var(--ds-color-primary);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.subtask-list {
  margin-top: 12px;
}

.subtask-item {
  display: grid;
  grid-template-columns: 22px minmax(0, 1fr) auto 18px;
  align-items: center;
  min-height: 52px;
  gap: 10px;
  width: 100%;
  padding: 0;
  color: inherit;
  text-align: left;
  cursor: pointer;
  background: transparent;
  border-bottom: 1px solid #eff1f6;
}

.subtask-item:hover {
  background: var(--ds-color-surface-muted);
}

.subtask-item:last-child {
  border-bottom: 0;
}

.subtask-check {
  display: grid;
  width: 19px;
  height: 19px;
  place-items: center;
  color: var(--ds-color-success);
  border: 1px solid #c6cfdd;
  border-radius: 50%;
}

.subtask-check svg {
  width: 19px;
  margin: -1px;
  background: #ffffff;
  border-radius: 50%;
}

.subtask-title {
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.subtask-item.is-complete .subtask-title {
  color: var(--ds-color-text-disabled);
  text-decoration: line-through;
}

.subtask-due {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.subtask-due svg {
  width: 13px;
}

.subtask-more {
  width: 15px;
  color: #c2cbd9;
}

.detail-inline-empty {
  margin-top: 18px;
  padding: 18px;
  color: var(--ds-color-text-tertiary);
  text-align: center;
  background: var(--ds-color-surface-muted);
  border-radius: 10px;
}

.detail-link-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 18px;
}

.detail-attachment-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 18px;
}

.detail-attachment-item {
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 42px;
  padding: 0 9px;
  background: var(--ds-color-surface-muted);
  border-radius: 8px;
}

.detail-attachment-icon {
  width: 16px;
  color: var(--ds-color-primary);
}

.detail-attachment-name {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-attachment-name:hover {
  color: var(--ds-color-primary);
}

.detail-attachment-size {
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.detail-link-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.detail-link-icon {
  width: 18px;
  margin-top: 9px;
  color: var(--ds-color-primary);
}

.detail-link-fields {
  display: grid;
  min-width: 0;
  flex: 1;
  gap: 8px;
}

.detail-assets-empty {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 18px;
  padding: 15px;
  background: var(--ds-color-surface-muted);
  border: 1px dashed var(--ds-color-border-strong);
  border-radius: 10px;
}

.asset-empty-icon,
.detail-note-icon {
  display: grid;
  width: 36px;
  height: 36px;
  flex: 0 0 auto;
  place-items: center;
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
  border-radius: 10px;
}

.asset-empty-icon svg,
.detail-note-icon svg {
  width: 18px;
}

.detail-assets-empty strong {
  display: block;
  font-size: 13px;
}

.detail-assets-empty p {
  margin: 4px 0 0;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.asset-empty-link {
  width: 18px;
  margin-left: auto;
  color: #c3ccda;
}

.property-list {
  margin-top: 17px;
}

.property-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 48px;
  border-bottom: 1px solid #eff1f6;
}

.property-row:last-child {
  border-bottom: 0;
}

.property-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.property-label svg {
  width: 15px;
}

.property-value {
  color: var(--ds-color-text-primary);
  font-size: 12px;
}

.property-value.is-danger {
  color: var(--ds-color-danger);
}

.property-value.is-high {
  color: #bd5d63;
}

.detail-note-card {
  padding: 20px;
  background: #f8f5ed;
  border: 1px solid #eee4cb;
  border-radius: var(--ds-radius-lg);
}

.detail-note-icon {
  color: #a87a2d;
  background: #fff4d7;
}

.detail-note-card h3 {
  margin: 14px 0 5px;
  font-size: 14px;
}

.detail-note-card p {
  margin: 0;
  color: #8b7956;
  font-size: 12px;
  line-height: 1.7;
}

.detail-meta-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 0 4px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

@media (max-width: 900px) {
  .detail-layout {
    grid-template-columns: 1fr;
  }
}
</style>
