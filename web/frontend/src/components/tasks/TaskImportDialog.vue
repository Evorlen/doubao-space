<script setup>
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { CircleCheck, Document, Upload, WarningFilled } from '@element-plus/icons-vue'
import { importTasks as importTasksRequest, previewTaskImport } from '../../api'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  task: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['update:modelValue', 'imported'])

const fileInput = ref(null)
const file = ref(null)
const preview = ref(null)
const previewing = ref(false)
const importing = ref(false)

const canImport = computed(() => Boolean(props.task && !props.task.completed && props.task.depth < 5))
const canConfirm = computed(() => Boolean(preview.value && preview.value.error_count === 0 && preview.value.new_count > 0 && !importing.value))
const dueHint = computed(() => props.task?.due_at
  ? '当前父任务有截止时间，表格中的每一行都必须填写不晚于父任务的截止时间。'
  : '当前父任务没有截止时间，表格中的截止时间可以按子任务需要填写，也可以留空。')

watch(() => props.modelValue, (visible) => {
  if (visible) reset()
})

function reset() {
  file.value = null
  preview.value = null
  previewing.value = false
  importing.value = false
  if (fileInput.value) fileInput.value.value = ''
}

function close() {
  if (!previewing.value && !importing.value) emit('update:modelValue', false)
}

function chooseFile() {
  if (canImport.value) fileInput.value?.click()
}

function downloadTemplate() {
  const link = document.createElement('a')
  link.href = '/api/tasks/import/template'
  link.download = '任务导入模板.xlsx'
  document.body.appendChild(link)
  link.click()
  link.remove()
}

async function handleFileChange(event) {
  const selected = event.target.files?.[0]
  event.target.value = ''
  if (!selected) return
  if (!selected.name.toLowerCase().endsWith('.xlsx')) {
    ElMessage.warning('请选择 .xlsx 格式的 Excel 文件。')
    return
  }
  file.value = selected
  await loadPreview()
}

async function loadPreview() {
  if (!file.value || !props.task) return
  previewing.value = true
  preview.value = null
  try {
    preview.value = await previewTaskImport(props.task.id, file.value)
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    previewing.value = false
  }
}

async function confirmImport() {
  if (!canConfirm.value || !props.task || !file.value) return
  importing.value = true
  try {
    const report = await importTasksRequest(props.task.id, file.value)
    if (report.error_count > 0) {
      preview.value = report
      ElMessage.error('导入文件发生变化，请修正错误后重新预览。')
      return
    }
    emit('imported', report)
    emit('update:modelValue', false)
    if (report.created_count > 0) {
      const duplicateText = report.duplicate_count > 0 ? `，跳过 ${report.duplicate_count} 个重复任务` : ''
      ElMessage.success(`已导入 ${report.created_count} 个子任务${duplicateText}。`)
    } else {
      ElMessage.info('没有新增任务，文件中的任务均已存在。')
    }
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    importing.value = false
  }
}

function statusLabel(status) {
  return {
    new: '待导入',
    duplicate: '重复跳过',
    error: '错误',
    created: '已创建',
  }[status] || status
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    title="导入子任务"
    width="760px"
    destroy-on-close
    @close="close"
  >
    <div class="task-import-dialog">
      <div class="task-import-context">
        <Document />
        <div>
          <strong>导入到：{{ task?.title }}</strong>
          <p>每一行会创建为当前任务的直系子任务。</p>
        </div>
      </div>

      <div class="task-import-picker">
        <input ref="fileInput" class="hidden-file-input" type="file" accept=".xlsx" @change="handleFileChange" />
        <el-button text type="primary" :disabled="previewing || importing" @click="downloadTemplate">
          下载空白模板
        </el-button>
        <el-button type="primary" :disabled="!canImport || previewing || importing" @click="chooseFile">
          <Upload class="button-icon" />选择 Excel 文件
        </el-button>
        <span v-if="file" class="task-import-file-name">{{ file.name }}</span>
      </div>

      <p class="task-import-hint">{{ dueHint }}</p>

      <div class="task-import-fields">
        <strong>字段填写说明</strong>
        <div class="task-import-field-grid">
          <span>任务标题</span><span>必填，每行一个任务。</span>
          <span>导入编号</span><span>可选；同一父任务下应保持唯一，便于重复导入时查重。</span>
          <span>任务描述</span><span>可选，填写普通文本。</span>
          <span>截止时间</span><span>填写 <code>2026-10-31 00:00</code>；也支持 Excel 日期单元格。父任务无期限时可按子任务需要填写。</span>
          <span>优先级</span><span>填写“紧急 / 高 / 中 / 低”，留空默认为“中”。</span>
          <span>标签</span><span>多个标签用“；”或英文逗号分隔。</span>
          <span>外部链接</span><span>格式为 <code>名称|https://example.com</code>，多个链接用“；”或换行分隔。</span>
        </div>
      </div>

      <div v-if="previewing" class="task-import-state">正在解析 Excel 并检查任务规则…</div>

      <template v-else-if="preview">
        <div class="task-import-summary">
          <span><strong>{{ preview.total }}</strong> 行</span>
          <span class="is-success"><CircleCheck />{{ preview.new_count }} 个待导入</span>
          <span class="is-warning"><WarningFilled />{{ preview.duplicate_count }} 个重复</span>
          <span v-if="preview.error_count" class="is-danger"><WarningFilled />{{ preview.error_count }} 个错误</span>
        </div>

        <el-table v-if="preview.rows?.length" :data="preview.rows" max-height="280" size="small" class="task-import-table">
          <el-table-column prop="row" label="行" width="60" />
          <el-table-column prop="title" label="任务标题" min-width="180" show-overflow-tooltip />
          <el-table-column label="结果" width="100">
            <template #default="scope">
              <span class="task-import-status" :class="`is-${scope.row.status}`">{{ statusLabel(scope.row.status) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="message" label="说明" min-width="260" show-overflow-tooltip />
        </el-table>

        <div v-if="preview.error_count" class="task-import-error-note">
          <WarningFilled />存在错误行，修正后重新选择文件；当前批次不会写入数据库。
        </div>
      </template>

      <div v-else class="task-import-empty">选择文件后会先进行预览和查重。</div>
    </div>

    <template #footer>
      <div class="dialog-footer-actions">
        <span>支持字段：标题、导入编号、描述、截止时间、优先级、标签和链接</span>
        <div>
          <el-button @click="close">取消</el-button>
          <el-button type="primary" :loading="importing" :disabled="!canConfirm" @click="confirmImport">确认导入</el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.task-import-dialog {
  display: grid;
  gap: 16px;
}

.task-import-context {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 14px 16px;
  color: var(--ds-color-text-secondary);
  background: var(--ds-color-primary-light);
  border-radius: 10px;
}

.task-import-context > svg {
  flex: 0 0 auto;
  width: 20px;
  height: 20px;
  margin-top: 1px;
  color: var(--ds-color-primary);
}

.task-import-context strong {
  color: var(--ds-color-text-primary);
}

.task-import-context p,
.task-import-hint {
  margin: 4px 0 0;
  color: var(--ds-color-text-tertiary);
  font-size: 13px;
}

.task-import-picker {
  display: flex;
  min-width: 0;
  gap: 12px;
  align-items: center;
}

.hidden-file-input {
  display: none;
}

.task-import-file-name {
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
  color: var(--ds-color-text-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-import-hint {
  margin-top: -8px;
}

.task-import-fields {
  display: grid;
  gap: 8px;
  padding: 12px 14px;
  color: var(--ds-color-text-secondary);
  background: var(--ds-color-surface-muted, #f7f8fa);
  border-radius: 10px;
  font-size: 12px;
}

.task-import-field-grid {
  display: grid;
  grid-template-columns: 76px minmax(0, 1fr);
  gap: 6px 12px;
  line-height: 1.5;
}

.task-import-field-grid span:nth-child(odd) {
  color: var(--ds-color-text-primary);
  font-weight: 600;
}

.task-import-fields code {
  padding: 1px 4px;
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
  border-radius: 4px;
  font-family: inherit;
}

.task-import-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  color: var(--ds-color-text-secondary);
  font-size: 13px;
}

.task-import-summary span {
  display: inline-flex;
  gap: 4px;
  align-items: center;
}

.task-import-summary svg,
.task-import-error-note svg {
  width: 15px;
  height: 15px;
}

.task-import-summary .is-success,
.task-import-status.is-new {
  color: var(--ds-color-success, #2f8f67);
}

.task-import-summary .is-warning,
.task-import-status.is-duplicate {
  color: #b7791f;
}

.task-import-summary .is-danger,
.task-import-status.is-error {
  color: var(--ds-color-danger, #d75b5b);
}

.task-import-status {
  font-size: 12px;
  font-weight: 600;
}

.task-import-status.is-created {
  color: var(--ds-color-primary);
}

.task-import-error-note {
  display: flex;
  gap: 6px;
  align-items: center;
  padding: 10px 12px;
  color: #a05c16;
  font-size: 13px;
  background: #fff8e8;
  border-radius: 8px;
}

.task-import-state,
.task-import-empty {
  padding: 28px 12px;
  color: var(--ds-color-text-tertiary);
  text-align: center;
}

.task-import-table :deep(.el-table__cell) {
  padding: 8px 0;
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

@media (max-width: 720px) {
  .task-import-picker {
    align-items: flex-start;
    flex-direction: column;
  }

  .dialog-footer-actions {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
