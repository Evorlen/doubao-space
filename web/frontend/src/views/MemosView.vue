<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Delete,
  DocumentAdd,
  EditPen,
  Folder,
  FolderOpened,
  Notebook,
  Refresh,
  Search,
  Upload,
} from '@element-plus/icons-vue'
import {
  archiveMemo,
  createMemo,
  deleteMemo,
  getMemos,
  restoreMemo,
  updateMemo,
} from '../api'

const viewOptions = [
  { value: 'active', label: '未归档' },
  { value: 'all', label: '全部' },
  { value: 'archived', label: '已归档' },
]

const memos = ref([])
const stats = ref({ total: 0, active: 0, archived: 0 })
const loading = ref(true)
const loadError = ref('')
const activeView = ref('active')
const searchQuery = ref('')
const selectedId = ref(null)
const editorOpen = ref(false)
const form = ref(createEmptyForm())
const savedSnapshot = ref('')
const saving = ref(false)
const contentEditor = ref(null)

const selectedMemo = computed(() => memos.value.find((memo) => memo.id === selectedId.value) || null)
const isNewMemo = computed(() => editorOpen.value && selectedId.value === null)
const isDirty = computed(() => editorOpen.value && formSnapshot() !== savedSnapshot.value)
const editorLabel = computed(() => (isNewMemo.value ? '新建备忘录' : '编辑备忘录'))

function createEmptyForm() {
  return { title: '', content_html: '' }
}

function formSnapshot() {
  return JSON.stringify({
    title: form.value.title.trim(),
    content_html: cleanEditorHTML(form.value.content_html),
  })
}

function memoSnapshot(memo) {
  return JSON.stringify({
    title: memo.title || '',
    content_html: cleanEditorHTML(memo.content_html || ''),
  })
}

function applyMemo(memo) {
  form.value = {
    title: memo.title || '',
    content_html: memo.content_html || '',
  }
  savedSnapshot.value = memoSnapshot(memo)
}

function formatDate(value) {
  if (!value) return '刚刚'
  const date = new Date(value.replace(' ', 'T'))
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function viewArchivedValue() {
  if (activeView.value === 'active') return false
  if (activeView.value === 'archived') return true
  return undefined
}

async function loadMemos() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await getMemos({
      archived: viewArchivedValue(),
      search: searchQuery.value.trim(),
    })
    memos.value = Array.isArray(response?.items) ? response.items : []
    stats.value = response?.stats || { total: 0, active: 0, archived: 0 }

    if (selectedId.value && !memos.value.some((memo) => memo.id === selectedId.value)) {
      selectedId.value = null
      editorOpen.value = false
    }
    if (!editorOpen.value && memos.value.length) {
      await selectMemo(memos.value[0], true)
    }
  } catch (error) {
    memos.value = []
    loadError.value = error.message
  } finally {
    loading.value = false
  }
}

async function changeView(value) {
  if (value === activeView.value) return
  if (!(await canLeaveEditor())) return
  activeView.value = value
  await loadMemos()
}

async function canLeaveEditor() {
  if (!isDirty.value) return true
  try {
    await ElMessageBox.confirm('当前备忘录还有未保存的修改，确定放弃吗？', '放弃修改', {
      type: 'warning',
      confirmButtonText: '放弃修改',
      cancelButtonText: '继续编辑',
    })
    return true
  } catch {
    return false
  }
}

async function selectMemo(memo, force = false) {
  if (!memo || (!force && memo.id === selectedId.value)) return
  if (!force && !(await canLeaveEditor())) return
  selectedId.value = memo.id
  editorOpen.value = true
  applyMemo(memo)
  await nextTick()
  syncEditorElement()
}

async function openCreate() {
  if (!(await canLeaveEditor())) return
  selectedId.value = null
  editorOpen.value = true
  form.value = createEmptyForm()
  savedSnapshot.value = formSnapshot()
  await nextTick()
  syncEditorElement()
}

function syncEditorElement() {
  if (contentEditor.value && contentEditor.value.innerHTML !== form.value.content_html) {
    contentEditor.value.innerHTML = form.value.content_html || ''
  }
}

function cleanEditorHTML(value) {
  const html = (value || '').trim()
  if (!html) return ''
  const container = document.createElement('div')
  container.innerHTML = html
  return container.textContent?.trim() ? html : ''
}

function syncEditor() {
  if (!contentEditor.value) return
  form.value.content_html = cleanEditorHTML(contentEditor.value.innerHTML)
}

function executeFormat(command, value = null) {
  contentEditor.value?.focus()
  document.execCommand(command, false, value)
  syncEditor()
}

function createLink() {
  contentEditor.value?.focus()
  const value = window.prompt('输入链接地址（建议使用 https:// 开头）')
  if (!value?.trim()) return
  document.execCommand('createLink', false, value.trim())
  syncEditor()
}

function handleEditorKeydown(event) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 's') {
    event.preventDefault()
    saveMemo()
  }
}

async function saveMemo() {
  syncEditor()
  if (!form.value.title.trim()) {
    ElMessage.warning('请先填写备忘录标题。')
    return
  }

  saving.value = true
  const creating = isNewMemo.value
  try {
    const payload = {
      title: form.value.title.trim(),
      content_html: cleanEditorHTML(form.value.content_html),
    }
    const saved = isNewMemo.value
      ? await createMemo(payload)
      : await updateMemo(selectedId.value, payload)
    selectedId.value = saved.id
    editorOpen.value = true
    applyMemo(saved)
    await loadMemos()
    const listed = memos.value.find((memo) => memo.id === saved.id)
    if (listed) applyMemo(listed)
    await nextTick()
    syncEditorElement()
    ElMessage.success(creating ? '备忘录已创建。' : '备忘录已保存。')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    saving.value = false
  }
}

async function archiveSelected() {
  if (!selectedMemo.value) return
  await changeArchiveState(selectedMemo.value)
}

async function restoreSelected() {
  if (!selectedMemo.value) return
  await changeArchiveState(selectedMemo.value)
}

async function changeArchiveState(memo) {
  try {
    const updated = memo.archived ? await restoreMemo(memo.id) : await archiveMemo(memo.id)
    ElMessage.success(updated.archived ? '备忘录已归档。' : '备忘录已恢复。')
    if (selectedId.value === memo.id) {
      if (activeView.value === 'active' && updated.archived) {
        selectedId.value = null
        editorOpen.value = false
      } else if (activeView.value === 'archived' && !updated.archived) {
        selectedId.value = null
        editorOpen.value = false
      } else {
        applyMemo(updated)
      }
    }
    await loadMemos()
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function removeMemo(memo) {
  try {
    await ElMessageBox.confirm(`删除“${memo.title}”后将无法恢复，确定继续吗？`, '删除备忘录', {
      type: 'warning',
      confirmButtonText: '永久删除',
      cancelButtonText: '取消',
      confirmButtonClass: 'el-button--danger',
    })
  } catch {
    return
  }

  try {
    await deleteMemo(memo.id)
    if (selectedId.value === memo.id) {
      selectedId.value = null
      editorOpen.value = false
    }
    await loadMemos()
    ElMessage.success('备忘录已删除。')
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function cancelChanges() {
  if (!(await canLeaveEditor())) return
  if (isNewMemo.value) {
    selectedId.value = null
    editorOpen.value = false
    return
  }
  if (selectedMemo.value) {
    applyMemo(selectedMemo.value)
    await nextTick()
    syncEditorElement()
  }
}

onMounted(async () => {
  await loadMemos()
  if (!memos.value.length) await openCreate()
})
</script>

<template>
  <div class="page-view page-view-wide memos-page">
    <section class="page-heading">
      <div class="page-heading-copy">
        <p class="eyebrow">MEMO SPACE</p>
        <h1>备忘录</h1>
        <p>把突然想到的事、值得记住的话和还没整理的灵感，先放在一个安静的地方。</p>
      </div>
      <div class="page-heading-actions">
        <el-button class="soft-button" :loading="loading" @click="loadMemos">
          <Refresh class="button-icon" />刷新
        </el-button>
        <el-button type="primary" @click="openCreate">
          <DocumentAdd class="button-icon" />新建备忘录
        </el-button>
      </div>
    </section>

    <div v-if="loadError" class="memos-error" role="alert">
      <Notebook />
      <span>备忘录暂时无法读取：{{ loadError }}</span>
      <el-button text type="primary" @click="loadMemos">重新加载</el-button>
    </div>

    <section class="memos-summary-grid">
      <div class="surface-card memo-summary-card is-total">
        <div class="memo-summary-icon"><Notebook /></div>
        <div><span>全部备忘录</span><strong>{{ stats.total }}</strong><small>所有记录都在这里</small></div>
      </div>
      <div class="surface-card memo-summary-card is-active">
        <div class="memo-summary-icon"><EditPen /></div>
        <div><span>正在整理</span><strong>{{ stats.active }}</strong><small>随时可以继续编辑</small></div>
      </div>
      <div class="surface-card memo-summary-card is-archived">
        <div class="memo-summary-icon"><FolderOpened /></div>
        <div><span>已归档</span><strong>{{ stats.archived }}</strong><small>收好，但不会忘记</small></div>
      </div>
    </section>

    <section class="surface-card memos-workspace">
      <aside class="memo-list-pane">
        <div class="memo-list-header">
          <div class="memo-view-tabs" role="tablist" aria-label="备忘录筛选">
            <button
              v-for="option in viewOptions"
              :key="option.value"
              type="button"
              class="memo-view-tab"
              :class="{ 'is-active': activeView === option.value }"
              @click="changeView(option.value)"
            >
              {{ option.label }}
              <span v-if="option.value === 'active'">{{ stats.active }}</span>
              <span v-else-if="option.value === 'archived'">{{ stats.archived }}</span>
              <span v-else>{{ stats.total }}</span>
            </button>
          </div>
          <el-input
            v-model="searchQuery"
            class="memo-search"
            clearable
            placeholder="搜索备忘录"
            @keyup.enter="loadMemos"
            @clear="loadMemos"
          >
            <template #prefix><Search /></template>
          </el-input>
        </div>

        <div v-if="loading" class="memo-list-loading">
          <div v-for="line in 5" :key="line" class="memo-loading-line"></div>
        </div>
        <div v-else-if="memos.length" class="memo-list" role="list">
          <article
            v-for="memo in memos"
            :key="memo.id"
            class="memo-list-item"
            :class="{ 'is-selected': memo.id === selectedId }"
            role="listitem"
            tabindex="0"
            @click="selectMemo(memo)"
            @keydown.enter="selectMemo(memo)"
          >
            <div class="memo-list-item-heading">
              <strong>{{ memo.title }}</strong>
              <span v-if="memo.archived" class="memo-archived-mark"><Folder /></span>
            </div>
            <p>{{ memo.excerpt || '还没有写下内容' }}</p>
            <div class="memo-list-item-footer">
              <time>{{ formatDate(memo.updated_at) }}</time>
              <div class="memo-list-item-actions">
                <button type="button" :title="memo.archived ? '恢复备忘录' : '归档备忘录'" @click.stop="changeArchiveState(memo)">
                  <Upload v-if="memo.archived" />
                  <Folder v-else />
                </button>
                <button type="button" title="删除备忘录" @click.stop="removeMemo(memo)"><Delete /></button>
              </div>
            </div>
          </article>
        </div>
        <div v-else class="memo-list-empty">
          <div class="memo-empty-icon"><Notebook /></div>
          <strong>{{ searchQuery ? '没有找到匹配的备忘录' : '这里还没有备忘录' }}</strong>
          <p>{{ searchQuery ? '换个关键词试试看。' : '从一条简单的记录开始。' }}</p>
          <el-button v-if="!searchQuery" type="primary" plain @click="openCreate">新建一条</el-button>
        </div>
      </aside>

      <section class="memo-editor-pane">
        <div v-if="editorOpen" class="memo-editor-shell">
          <div class="memo-editor-header">
            <div class="memo-editor-heading">
              <span class="memo-editor-badge"><EditPen /></span>
              <div><span>{{ editorLabel }}</span><small>{{ isNewMemo ? '先写下来，之后再慢慢整理。' : `上次编辑于 ${formatDate(selectedMemo?.updated_at)}` }}</small></div>
            </div>
            <div class="memo-editor-actions">
              <el-button v-if="selectedMemo && !selectedMemo.archived" text @click="archiveSelected"><Folder class="button-icon" />归档</el-button>
              <el-button v-if="selectedMemo?.archived" text @click="restoreSelected"><Upload class="button-icon" />恢复</el-button>
              <el-button v-if="selectedMemo" text type="danger" @click="removeMemo(selectedMemo)"><Delete class="button-icon" />删除</el-button>
            </div>
          </div>

          <div class="memo-editor-form">
            <el-input v-model="form.title" class="memo-title-input" maxlength="200" placeholder="给这条备忘录起个标题" />
            <div class="memo-rich-editor">
              <div class="memo-editor-toolbar" role="toolbar" aria-label="富文本工具栏">
                <button type="button" title="加粗" @click="executeFormat('bold')"><strong>B</strong></button>
                <button type="button" title="斜体" @click="executeFormat('italic')"><em>I</em></button>
                <button type="button" title="下划线" @click="executeFormat('underline')"><u>U</u></button>
                <span class="memo-toolbar-divider"></span>
                <button type="button" title="标题" @click="executeFormat('formatBlock', 'h2')">H</button>
                <button type="button" title="项目符号列表" @click="executeFormat('insertUnorderedList')">•≡</button>
                <button type="button" title="编号列表" @click="executeFormat('insertOrderedList')">1≡</button>
                <button type="button" title="引用" @click="executeFormat('formatBlock', 'blockquote')">❝</button>
                <span class="memo-toolbar-divider"></span>
                <button type="button" title="添加链接" @click="createLink">↗</button>
                <button type="button" title="清除格式" @click="executeFormat('removeFormat')">Tx</button>
              </div>
              <div
                ref="contentEditor"
                class="memo-content-editor"
                contenteditable="true"
                role="textbox"
                aria-label="备忘录内容"
                data-placeholder="从这里开始写下你的想法……"
                spellcheck="true"
                @input="syncEditor"
                @blur="syncEditor"
                @keydown="handleEditorKeydown"
              ></div>
            </div>
          </div>

          <div class="memo-editor-footer">
            <span><span class="memo-save-dot" :class="{ 'is-dirty': isDirty }"></span>{{ isDirty ? '有未保存的修改' : '已保存到本地' }}</span>
            <div>
              <el-button v-if="isDirty" @click="cancelChanges">取消修改</el-button>
              <el-button type="primary" :loading="saving" @click="saveMemo">保存备忘录</el-button>
            </div>
          </div>
        </div>
        <div v-else class="memo-editor-empty">
          <div class="memo-editor-empty-mark"><Notebook /></div>
          <h2>选一条备忘录开始</h2>
          <p>或者新建一条，把脑海里的想法先放下来。</p>
          <el-button type="primary" @click="openCreate"><DocumentAdd class="button-icon" />新建备忘录</el-button>
        </div>
      </section>
    </section>
  </div>
</template>

<style scoped>
.memos-page {
  max-width: 1440px;
}

.memos-error {
  display: flex;
  align-items: center;
  gap: 9px;
  margin: -8px 0 18px;
  padding: 11px 14px;
  color: #8b5a26;
  font-size: 13px;
  background: #fff8e7;
  border: 1px solid #f2dfb2;
  border-radius: var(--ds-radius-md);
}

.memos-error > svg {
  width: 16px;
  flex: 0 0 auto;
}

.memos-error .el-button {
  margin-left: auto;
}

.memos-summary-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 18px;
}

.memo-summary-card {
  display: flex;
  align-items: center;
  gap: 13px;
  min-height: 102px;
  padding: 18px;
}

.memo-summary-icon {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 12px;
}

.memo-summary-icon svg {
  width: 19px;
}

.memo-summary-card.is-total .memo-summary-icon {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.memo-summary-card.is-active .memo-summary-icon {
  color: #5d9076;
  background: #e8f4ed;
}

.memo-summary-card.is-archived .memo-summary-icon {
  color: #a4772e;
  background: #fff1ca;
}

.memo-summary-card span,
.memo-summary-card strong,
.memo-summary-card small {
  display: block;
}

.memo-summary-card span {
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.memo-summary-card strong {
  margin: 2px 0 1px;
  color: var(--ds-color-text-primary);
  font-size: 23px;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.03em;
}

.memo-summary-card small {
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.memos-workspace {
  display: grid;
  grid-template-columns: 340px minmax(0, 1fr);
  min-height: 650px;
  overflow: hidden;
}

.memo-list-pane {
  display: flex;
  min-width: 0;
  flex-direction: column;
  border-right: 1px solid var(--ds-color-border);
  background: var(--ds-color-surface-muted);
}

.memo-list-header {
  padding: 18px 16px 14px;
  border-bottom: 1px solid var(--ds-color-border);
}

.memo-view-tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 13px;
}

.memo-view-tab {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 7px 9px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  cursor: pointer;
  background: transparent;
  border-radius: 8px;
}

.memo-view-tab:hover,
.memo-view-tab.is-active {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.memo-view-tab.is-active {
  font-weight: 700;
}

.memo-view-tab span {
  min-width: 17px;
  padding: 1px 4px;
  color: inherit;
  font-size: 10px;
  text-align: center;
  background: rgba(82, 107, 155, 0.1);
  border-radius: 999px;
}

.memo-search {
  width: 100%;
}

.memo-list {
  flex: 1;
  overflow-y: auto;
}

.memo-list-item {
  display: block;
  width: 100%;
  padding: 16px 17px 14px;
  text-align: left;
  cursor: pointer;
  background: transparent;
  border-bottom: 1px solid var(--ds-color-border);
  border-left: 3px solid transparent;
  transition: background-color 160ms ease, border-color 160ms ease;
}

.memo-list-item:hover {
  background: #ffffff;
}

.memo-list-item.is-selected {
  background: #ffffff;
  border-left-color: var(--ds-color-accent);
}

.memo-list-item-heading,
.memo-list-item-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.memo-list-item-heading strong {
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.memo-list-item p {
  display: -webkit-box;
  min-height: 38px;
  margin: 7px 0 10px;
  overflow: hidden;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  line-height: 1.6;
  text-overflow: ellipsis;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.memo-archived-mark {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  color: #b38337;
}

.memo-archived-mark svg {
  width: 14px;
}

.memo-list-item-footer time {
  color: #a3adbf;
  font-size: 10px;
}

.memo-list-item-actions {
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity 160ms ease;
}

.memo-list-item:hover .memo-list-item-actions,
.memo-list-item.is-selected .memo-list-item-actions {
  opacity: 1;
}

.memo-list-item-actions button {
  display: grid;
  width: 24px;
  height: 24px;
  place-items: center;
  color: var(--ds-color-text-tertiary);
  cursor: pointer;
  background: transparent;
  border-radius: 6px;
}

.memo-list-item-actions button:hover {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.memo-list-item-actions button:last-child:hover {
  color: var(--ds-color-danger);
  background: #fff0f0;
}

.memo-list-item-actions svg {
  width: 13px;
}

.memo-list-empty,
.memo-editor-empty {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  padding: 40px 22px;
  color: var(--ds-color-text-tertiary);
  text-align: center;
}

.memo-list-empty {
  min-height: 360px;
}

.memo-empty-icon,
.memo-editor-empty-mark {
  display: grid;
  place-items: center;
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
  border-radius: 16px;
}

.memo-empty-icon {
  width: 44px;
  height: 44px;
  margin-bottom: 14px;
}

.memo-empty-icon svg {
  width: 21px;
}

.memo-list-empty strong {
  color: var(--ds-color-text-secondary);
  font-size: 13px;
}

.memo-list-empty p,
.memo-editor-empty p {
  margin: 6px 0 16px;
  font-size: 12px;
}

.memo-list-loading {
  padding: 16px;
}

.memo-loading-line {
  height: 78px;
  margin-bottom: 10px;
  background: linear-gradient(90deg, #edf0f6 25%, #f7f8fb 38%, #edf0f6 63%);
  background-size: 400% 100%;
  border-radius: 10px;
  animation: memo-loading 1.4s ease-in-out infinite;
}

@keyframes memo-loading {
  0% { background-position: 100% 0; }
  100% { background-position: -100% 0; }
}

.memo-editor-pane {
  display: flex;
  min-width: 0;
  min-height: 650px;
  background: #ffffff;
}

.memo-editor-shell {
  display: flex;
  width: 100%;
  flex-direction: column;
}

.memo-editor-header,
.memo-editor-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.memo-editor-header {
  min-height: 72px;
  padding: 16px 24px;
  border-bottom: 1px solid var(--ds-color-border);
}

.memo-editor-heading {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.memo-editor-badge {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 auto;
  place-items: center;
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
  border-radius: 10px;
}

.memo-editor-badge svg {
  width: 16px;
}

.memo-editor-heading span,
.memo-editor-heading small {
  display: block;
}

.memo-editor-heading span {
  color: var(--ds-color-text-primary);
  font-size: 13px;
  font-weight: 700;
}

.memo-editor-heading small {
  margin-top: 2px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.memo-editor-actions {
  display: flex;
  align-items: center;
  flex: 0 0 auto;
  gap: 2px;
}

.memo-editor-form {
  display: flex;
  flex: 1;
  min-height: 0;
  flex-direction: column;
  padding: 26px 34px 18px;
}

.memo-title-input :deep(.el-input__wrapper) {
  min-height: 52px;
  padding: 0;
  box-shadow: none;
}

.memo-title-input :deep(.el-input__inner) {
  color: var(--ds-color-text-primary);
  font-size: 25px;
  font-weight: 700;
  letter-spacing: -0.04em;
}

.memo-title-input :deep(.el-input__inner::placeholder) {
  color: #c2c9d6;
}

.memo-rich-editor {
  display: flex;
  flex: 1;
  min-height: 0;
  flex-direction: column;
  margin-top: 16px;
  border: 1px solid var(--ds-color-border);
  border-radius: 12px;
  overflow: hidden;
}

.memo-editor-toolbar {
  display: flex;
  align-items: center;
  gap: 3px;
  padding: 8px 10px;
  background: var(--ds-color-surface-muted);
  border-bottom: 1px solid var(--ds-color-border);
}

.memo-editor-toolbar button {
  display: grid;
  min-width: 30px;
  height: 30px;
  place-items: center;
  padding: 0 7px;
  color: var(--ds-color-text-secondary);
  font-size: 13px;
  cursor: pointer;
  background: transparent;
  border-radius: 7px;
}

.memo-editor-toolbar button:hover {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.memo-toolbar-divider {
  width: 1px;
  height: 18px;
  margin: 0 5px;
  background: var(--ds-color-border);
}

.memo-content-editor {
  flex: 1;
  min-height: 330px;
  padding: 22px 24px;
  color: var(--ds-color-text-primary);
  font-size: 14px;
  line-height: 1.85;
  outline: none;
  overflow-y: auto;
}

.memo-content-editor:empty::before {
  color: #c2c9d6;
  content: attr(data-placeholder);
  pointer-events: none;
}

.memo-content-editor :deep(h2) {
  margin: 16px 0 7px;
  font-size: 20px;
}

.memo-content-editor :deep(p),
.memo-content-editor :deep(div) {
  margin: 0 0 8px;
}

.memo-content-editor :deep(ul),
.memo-content-editor :deep(ol) {
  margin: 7px 0;
  padding-left: 24px;
}

.memo-content-editor :deep(blockquote) {
  margin: 12px 0;
  padding: 5px 14px;
  color: var(--ds-color-text-tertiary);
  border-left: 3px solid var(--ds-color-accent);
}

.memo-content-editor :deep(a) {
  color: var(--ds-color-primary);
  text-decoration: underline;
}

.memo-editor-footer {
  padding: 14px 34px 22px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.memo-editor-footer > span {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}

.memo-save-dot {
  width: 7px;
  height: 7px;
  background: var(--ds-color-success);
  border-radius: 50%;
}

.memo-save-dot.is-dirty {
  background: var(--ds-color-warning);
}

.memo-editor-empty-mark {
  width: 62px;
  height: 62px;
  margin-bottom: 17px;
  border-radius: 20px;
}

.memo-editor-empty-mark svg {
  width: 28px;
}

.memo-editor-empty h2 {
  margin: 0;
  color: var(--ds-color-text-secondary);
  font-size: 16px;
}

@media (max-width: 1180px) {
  .memos-workspace {
    grid-template-columns: 300px minmax(0, 1fr);
  }

  .memo-editor-form {
    padding-right: 24px;
    padding-left: 24px;
  }

  .memo-editor-footer {
    padding-right: 24px;
    padding-left: 24px;
  }
}
</style>
