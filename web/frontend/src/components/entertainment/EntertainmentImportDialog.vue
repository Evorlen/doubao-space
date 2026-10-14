<script setup>
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { CircleCheck, Collection, Upload, WarningFilled } from '@element-plus/icons-vue'
import { importEntertainmentItems, previewEntertainmentImport } from '../../api'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['update:modelValue', 'imported'])

const fileInput = ref(null)
const file = ref(null)
const preview = ref(null)
const previewing = ref(false)
const importing = ref(false)

const canConfirm = computed(() => Boolean(
  preview.value
  && preview.value.error_count === 0
  && preview.value.new_count > 0
  && !importing.value,
))

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
  if (!previewing.value && !importing.value) fileInput.value?.click()
}

function downloadTemplate() {
  const link = document.createElement('a')
  link.href = '/api/entertainment/import/template'
  link.download = '娱乐导入模板.xlsx'
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
  if (!file.value) return
  previewing.value = true
  preview.value = null
  try {
    preview.value = await previewEntertainmentImport(file.value)
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    previewing.value = false
  }
}

async function confirmImport() {
  if (!canConfirm.value || !file.value) return
  importing.value = true
  try {
    const report = await importEntertainmentItems(file.value)
    if (report.error_count > 0) {
      preview.value = report
      ElMessage.error('导入文件发生变化，请修正错误后重新预览。')
      return
    }
    emit('imported', report)
    emit('update:modelValue', false)
    if (report.created_count > 0) {
      const duplicateText = report.duplicate_count > 0 ? `，跳过 ${report.duplicate_count} 个重复作品` : ''
      ElMessage.success(`已导入 ${report.created_count} 个作品${duplicateText}。`)
    } else {
      ElMessage.info('没有新增作品，文件中的作品均已存在。')
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
    title="导入娱乐作品"
    width="820px"
    destroy-on-close
    :close-on-click-modal="false"
    @close="close"
  >
    <div class="entertainment-import-dialog">
      <div class="entertainment-import-context">
        <Collection />
        <div>
          <strong>批量添加小说、电视剧和电影</strong>
          <p>每一行会创建一个作品，导入前会先检查字段格式并自动跳过重复记录。</p>
        </div>
      </div>

      <div class="entertainment-import-picker">
        <input ref="fileInput" class="hidden-file-input" type="file" accept=".xlsx" @change="handleFileChange" />
        <el-button text type="primary" :disabled="previewing || importing" @click="downloadTemplate">
          下载空白模板
        </el-button>
        <el-button type="primary" :disabled="previewing || importing" @click="chooseFile">
          <Upload class="button-icon" />选择 Excel 文件
        </el-button>
        <span v-if="file" class="entertainment-import-file-name">{{ file.name }}</span>
      </div>

      <div class="entertainment-import-fields">
        <strong>字段填写说明</strong>
        <div class="entertainment-import-field-grid">
          <span>作品类型</span><span>必填，填写“小说 / 电视剧 / 电影”，也支持 <code>novel / series / movie</code>。</span>
          <span>作品名称</span><span>必填，每行一个作品。</span>
          <span>作者或导演</span><span>可选；会参与重复判断。</span>
          <span>年份</span><span>可选，填写四位年份。</span>
          <span>状态</span><span>填写“计划中 / 进行中 / 已完成 / 搁置 / 放弃”，留空默认为计划中。</span>
          <span>当前进度 / 总进度</span><span>小说填写章节，电视剧填写集数；电影会忽略进度。</span>
          <span>评分</span><span>可选，填写 0 到 5 的整数。</span>
          <span>收藏</span><span>填写“是 / 否”，留空默认为否。</span>
          <span>标签</span><span>多个标签用“；”或英文逗号分隔。</span>
          <span>日期</span><span>填写 <code>2026-10-31</code>；也支持 Excel 日期单元格。</span>
        </div>
      </div>

      <div v-if="previewing" class="entertainment-import-state">正在解析 Excel 并检查作品字段…</div>

      <template v-else-if="preview">
        <div class="entertainment-import-summary">
          <span><strong>{{ preview.total }}</strong> 行</span>
          <span class="is-success"><CircleCheck />{{ preview.new_count }} 个待导入</span>
          <span class="is-warning"><WarningFilled />{{ preview.duplicate_count }} 个重复</span>
          <span v-if="preview.error_count" class="is-danger"><WarningFilled />{{ preview.error_count }} 个错误</span>
        </div>

        <el-table v-if="preview.rows?.length" :data="preview.rows" max-height="280" size="small" class="entertainment-import-table">
          <el-table-column prop="row" label="行" width="60" />
          <el-table-column prop="title" label="作品名称" min-width="180" show-overflow-tooltip />
          <el-table-column label="结果" width="100">
            <template #default="scope">
              <span class="entertainment-import-status" :class="`is-${scope.row.status}`">{{ statusLabel(scope.row.status) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="message" label="说明" min-width="280" show-overflow-tooltip />
        </el-table>

        <div v-if="preview.error_count" class="entertainment-import-error-note">
          <WarningFilled />存在错误行，修正后重新选择文件；当前批次不会写入数据库。
        </div>
      </template>

      <div v-else class="entertainment-import-empty">选择文件后会先进行预览和查重。</div>
    </div>

    <template #footer>
      <div class="entertainment-import-footer">
        <span>支持批量导入作品信息和标签</span>
        <div>
          <el-button @click="close">取消</el-button>
          <el-button type="primary" :loading="importing" :disabled="!canConfirm" @click="confirmImport">确认导入</el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.entertainment-import-dialog {
  display: grid;
  gap: 16px;
}

.entertainment-import-context {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 14px 16px;
  color: var(--ds-color-text-secondary);
  background: var(--ds-color-primary-light);
  border-radius: 10px;
}

.entertainment-import-context > svg {
  flex: 0 0 auto;
  width: 20px;
  height: 20px;
  margin-top: 1px;
  color: var(--ds-color-primary);
}

.entertainment-import-context strong {
  color: var(--ds-color-text-primary);
}

.entertainment-import-context p {
  margin: 4px 0 0;
  color: var(--ds-color-text-tertiary);
  font-size: 13px;
}

.entertainment-import-picker {
  display: flex;
  min-width: 0;
  gap: 12px;
  align-items: center;
}

.hidden-file-input {
  display: none;
}

.entertainment-import-file-name {
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
  color: var(--ds-color-text-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.entertainment-import-fields {
  display: grid;
  gap: 8px;
  padding: 12px 14px;
  color: var(--ds-color-text-secondary);
  background: var(--ds-color-surface-muted, #f7f8fa);
  border-radius: 10px;
  font-size: 12px;
}

.entertainment-import-field-grid {
  display: grid;
  grid-template-columns: 112px minmax(0, 1fr);
  gap: 6px 12px;
  line-height: 1.5;
}

.entertainment-import-field-grid span:nth-child(odd) {
  color: var(--ds-color-text-primary);
  font-weight: 600;
}

.entertainment-import-fields code {
  padding: 1px 4px;
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
  border-radius: 4px;
  font-family: inherit;
}

.entertainment-import-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  color: var(--ds-color-text-secondary);
  font-size: 13px;
}

.entertainment-import-summary span {
  display: inline-flex;
  gap: 4px;
  align-items: center;
}

.entertainment-import-summary svg,
.entertainment-import-error-note svg {
  width: 15px;
  height: 15px;
}

.entertainment-import-summary .is-success,
.entertainment-import-status.is-new {
  color: var(--ds-color-success, #2f8f67);
}

.entertainment-import-summary .is-warning,
.entertainment-import-status.is-duplicate {
  color: #b7791f;
}

.entertainment-import-summary .is-danger,
.entertainment-import-status.is-error {
  color: var(--ds-color-danger, #d75b5b);
}

.entertainment-import-status {
  font-size: 12px;
  font-weight: 600;
}

.entertainment-import-status.is-created {
  color: var(--ds-color-primary);
}

.entertainment-import-error-note {
  display: flex;
  gap: 6px;
  align-items: center;
  padding: 10px 12px;
  color: #a05c16;
  font-size: 13px;
  background: #fff8e8;
  border-radius: 8px;
}

.entertainment-import-state,
.entertainment-import-empty {
  padding: 28px 12px;
  color: var(--ds-color-text-tertiary);
  text-align: center;
}

.entertainment-import-table :deep(.el-table__cell) {
  padding: 8px 0;
}

.entertainment-import-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.entertainment-import-footer > span {
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

@media (max-width: 720px) {
  .entertainment-import-picker {
    align-items: flex-start;
    flex-direction: column;
  }

  .entertainment-import-footer {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
