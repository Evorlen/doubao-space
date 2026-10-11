<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowLeft,
  ArrowRight,
  Close,
  Delete,
  Edit,
  Loading,
  Picture,
  Refresh,
  Upload,
  VideoCamera,
  WarningFilled,
} from '@element-plus/icons-vue'
import {
  deleteDoubaoAlbumItem,
  getDoubaoAlbum,
  updateDoubaoAlbumItem,
  uploadDoubaoAlbum,
} from '../../api'

const maxFileSize = 100 * 1024 * 1024
const maxNoteLength = 500

const fileInput = ref(null)
const items = ref([])
const stats = ref({ total: 0, photos: 0, videos: 0 })
const loading = ref(true)
const loadError = ref('')
const uploading = ref(false)
const uploadTotal = ref(0)
const uploadCompleted = ref(0)
const uploadNote = ref('')
const dragging = ref(false)
const editDialogVisible = ref(false)
const editingId = ref(null)
const editNote = ref('')
const savingNote = ref(false)
const deletingId = ref(null)
const previewItem = ref(null)

const uploadProgressLabel = computed(() => {
  if (!uploadTotal.value) return '正在准备上传…'
  return `正在上传 ${uploadCompleted.value}/${uploadTotal.value}`
})

async function loadAlbum() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await getDoubaoAlbum()
    items.value = response.items || []
    stats.value = response.stats || { total: 0, photos: 0, videos: 0 }
  } catch (error) {
    items.value = []
    stats.value = { total: 0, photos: 0, videos: 0 }
    loadError.value = error.message
  } finally {
    loading.value = false
  }
}

function openPicker() {
  if (!uploading.value) fileInput.value?.click()
}

function handleFileChange(event) {
  const files = Array.from(event.target.files || [])
  event.target.value = ''
  uploadFiles(files)
}

function startDragging() {
  dragging.value = true
}

function stopDragging() {
  dragging.value = false
}

function handleDrop(event) {
  dragging.value = false
  uploadFiles(Array.from(event.dataTransfer?.files || []))
}

function isSupportedFile(file) {
  return file.type.startsWith('image/') || file.type.startsWith('video/')
}

async function uploadFiles(files) {
  if (uploading.value || !files.length) return

  const accepted = []
  let rejected = 0
  for (const file of files) {
    if (!isSupportedFile(file) || file.size > maxFileSize || file.size === 0) {
      rejected += 1
      continue
    }
    accepted.push(file)
  }
  if (rejected) {
    ElMessage.warning(`${rejected} 个文件未上传，请确认是照片或视频且大小不超过 100 MB。`)
  }
  if (!accepted.length) return

  uploading.value = true
  uploadTotal.value = accepted.length
  uploadCompleted.value = 0
  let uploaded = 0
  try {
    for (const file of accepted) {
      try {
        await uploadDoubaoAlbum(file, uploadNote.value)
        uploaded += 1
      } catch (error) {
        ElMessage.error(`${file.name}：${error.message}`)
      } finally {
        uploadCompleted.value += 1
      }
    }
    await loadAlbum()
    if (uploaded) {
      uploadNote.value = ''
      ElMessage.success(`已添加 ${uploaded} 个相册记录。`)
    }
  } finally {
    uploading.value = false
    uploadTotal.value = 0
    uploadCompleted.value = 0
  }
}

function openEdit(item) {
  editingId.value = item.id
  editNote.value = item.note || ''
  editDialogVisible.value = true
}

async function saveNote() {
  if (editNote.value.length > maxNoteLength) {
    ElMessage.warning(`备注不能超过 ${maxNoteLength} 个字。`)
    return
  }
  savingNote.value = true
  try {
    const updated = await updateDoubaoAlbumItem(editingId.value, editNote.value)
    const item = items.value.find((entry) => entry.id === editingId.value)
    if (item) Object.assign(item, updated)
    editDialogVisible.value = false
    ElMessage.success('备注已更新。')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    savingNote.value = false
  }
}

async function removeItem(item) {
  try {
    await ElMessageBox.confirm(
      `删除“${item.name}”后，原始文件也会从本地相册中移除。`,
      '删除相册记录',
      { confirmButtonText: '删除', cancelButtonText: '保留', type: 'warning' },
    )
  } catch {
    return
  }

  deletingId.value = item.id
  try {
    await deleteDoubaoAlbumItem(item.id)
    await loadAlbum()
    ElMessage.success('相册记录已删除。')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    deletingId.value = null
  }
}

function formatDate(value) {
  if (!value) return ''
  const raw = value.includes('T') ? value : value.replace(' ', 'T') + 'Z'
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return value.slice(0, 16)
  return date.toLocaleString('zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatSize(size) {
  if (!size) return '0 B'
  if (size < 1024 * 1024) return `${Math.max(1, Math.round(size / 1024))} KB`
  return `${(size / 1024 / 1024).toFixed(size >= 10 * 1024 * 1024 ? 0 : 1)} MB`
}

const previewIndex = computed(() => {
  if (!previewItem.value) return -1
  return items.value.findIndex((item) => item.id === previewItem.value.id)
})

const previewPosition = computed(() => {
  if (previewIndex.value < 0) return ''
  return `${previewIndex.value + 1} / ${items.value.length}`
})

function openPreview(item) {
  previewItem.value = item
}

function closePreview() {
  previewItem.value = null
}

function movePreview(offset) {
  if (!items.value.length || previewIndex.value < 0) return
  const nextIndex = (previewIndex.value + offset + items.value.length) % items.value.length
  previewItem.value = items.value[nextIndex]
}

function handlePreviewKeydown(event) {
  if (!previewItem.value) return
  if (event.key === 'Escape') closePreview()
  if (event.key === 'ArrowLeft') movePreview(-1)
  if (event.key === 'ArrowRight') movePreview(1)
}

onMounted(() => {
  loadAlbum()
  window.addEventListener('keydown', handlePreviewKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handlePreviewKeydown)
})
</script>

<template>
  <section id="doubao-album" class="surface-card doubao-album">
    <div class="album-heading">
      <div>
        <p class="card-overline">DOUBAO ALBUM</p>
        <h2>豆包相册</h2>
        <p>把豆包的照片和视频留在这里，也给每一段日常写一句备注。</p>
      </div>
      <div class="album-heading-actions">
        <div class="album-stats" aria-label="相册统计">
          <span><strong>{{ stats.total }}</strong>全部</span>
          <span><strong>{{ stats.photos }}</strong>照片</span>
          <span><strong>{{ stats.videos }}</strong>视频</span>
        </div>
        <el-button class="soft-button" :loading="loading" @click="loadAlbum">
          <Refresh class="button-icon" />刷新
        </el-button>
        <el-button type="primary" :loading="uploading" @click="openPicker">
          <Upload class="button-icon" />上传照片 / 视频
        </el-button>
        <input
          ref="fileInput"
          class="album-file-input"
          type="file"
          accept="image/*,video/*"
          multiple
          @change="handleFileChange"
        />
      </div>
    </div>

    <div
      class="album-upload-zone"
      :class="{ 'is-dragging': dragging }"
      @dragenter.prevent="startDragging"
      @dragover.prevent="startDragging"
      @dragleave.prevent="stopDragging"
      @drop.prevent="handleDrop"
    >
      <div class="album-upload-icon"><Upload /></div>
      <div class="album-upload-copy">
        <strong>拖放照片或视频到这里</strong>
        <span>支持一次选择多个文件，单个文件最大 100 MB。</span>
        <button type="button" class="album-upload-link" @click="openPicker">从电脑选择文件</button>
      </div>
      <div class="album-upload-note">
        <label for="album-upload-note">上传备注 <span>可选</span></label>
        <el-input
          id="album-upload-note"
          v-model="uploadNote"
          maxlength="500"
          placeholder="例如：今天第一次晒太阳"
          @click.stop
        />
      </div>
    </div>

    <div v-if="uploading" class="album-upload-progress">
      <Loading class="is-loading" />
      <span>{{ uploadProgressLabel }}</span>
      <el-progress :percentage="uploadTotal ? Math.round(uploadCompleted / uploadTotal * 100) : 0" :show-text="false" />
    </div>

    <div v-if="loadError" class="album-error" role="alert">
      <WarningFilled />
      <span>相册暂时无法读取：{{ loadError }}</span>
      <el-button text type="primary" @click="loadAlbum">重新加载</el-button>
    </div>

    <div v-if="loading" class="album-loading-grid" aria-label="正在加载相册">
      <div v-for="index in 4" :key="index" class="album-loading-card"></div>
    </div>
    <div v-else-if="items.length" class="album-grid">
      <article v-for="item in items" :key="item.id" class="album-card">
        <div class="album-media-frame">
          <img
            v-if="item.media_type === 'photo'"
            :src="item.media_url"
            :alt="item.note || item.name"
            loading="lazy"
            @click="openPreview(item)"
          />
          <video v-else :src="item.media_url" controls preload="metadata" playsinline></video>
          <span class="album-media-badge">
            <Picture v-if="item.media_type === 'photo'" />
            <VideoCamera v-else />
            {{ item.media_type_label }}
          </span>
          <button type="button" class="album-media-preview-action" @click.stop="openPreview(item)">
            <Picture v-if="item.media_type === 'photo'" />
            <VideoCamera v-else />
            {{ item.media_type === 'photo' ? '查看大图' : '大图播放' }}
          </button>
        </div>
        <div class="album-card-body">
          <div class="album-card-meta">
            <span>{{ formatDate(item.created_at) }}</span>
            <span>{{ formatSize(item.size) }}</span>
          </div>
          <p class="album-item-note" :class="{ 'is-empty': !item.note }">
            {{ item.note || '还没有备注，给这段日常留句话吧。' }}
          </p>
          <div class="album-file-name" :title="item.name">{{ item.name }}</div>
          <div class="album-card-actions">
            <button type="button" @click="openEdit(item)"><Edit />编辑备注</button>
            <button type="button" :disabled="deletingId === item.id" @click="removeItem(item)"><Delete />删除</button>
          </div>
        </div>
      </article>
    </div>
    <div v-else class="album-empty-state">
      <div class="album-empty-icon"><Picture /></div>
      <strong>还没有豆包相册记录</strong>
      <p>上传一张照片或一段视频，开始收集豆包的可爱日常。</p>
      <el-button type="primary" @click="openPicker"><Upload class="button-icon" />上传第一份记录</el-button>
    </div>

    <el-dialog v-model="editDialogVisible" title="编辑相册备注" width="480px" destroy-on-close>
      <el-input
        v-model="editNote"
        type="textarea"
        :rows="5"
        maxlength="500"
        show-word-limit
        placeholder="写下这张照片或这段视频里的豆包故事"
      />
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingNote" @click="saveNote">保存备注</el-button>
      </template>
    </el-dialog>

    <Transition name="album-preview-fade">
      <div
        v-if="previewItem"
        class="album-preview-overlay"
        role="dialog"
        aria-modal="true"
        :aria-label="previewItem.media_type === 'photo' ? '照片大图预览' : '视频大图播放'"
        @click.self="closePreview"
      >
        <div class="album-preview-shell">
          <div class="album-preview-toolbar">
            <div class="album-preview-title">
              <strong>{{ previewItem.note || previewItem.name }}</strong>
              <span>{{ previewItem.media_type_label }} · {{ previewPosition }}</span>
            </div>
            <button type="button" class="album-preview-close" aria-label="关闭预览" @click="closePreview">
              <Close />
            </button>
          </div>

          <button type="button" class="album-preview-nav is-prev" aria-label="上一项" @click="movePreview(-1)">
            <ArrowLeft />
          </button>
          <div class="album-preview-stage">
            <img
              v-if="previewItem.media_type === 'photo'"
              :src="previewItem.media_url"
              :alt="previewItem.note || previewItem.name"
            />
            <video
              v-else
              :src="previewItem.media_url"
              controls
              autoplay
              playsinline
              preload="auto"
            ></video>
          </div>
          <button type="button" class="album-preview-nav is-next" aria-label="下一项" @click="movePreview(1)">
            <ArrowRight />
          </button>

          <div class="album-preview-footer">
            <span>{{ previewItem.name }}</span>
            <span>{{ formatDate(previewItem.created_at) }} · {{ formatSize(previewItem.size) }}</span>
          </div>
        </div>
      </div>
    </Transition>
  </section>
</template>

<style scoped>
.doubao-album {
  margin-top: 20px;
  padding: 24px;
}

.album-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
}

.album-heading h2 {
  margin: 0;
  color: var(--ds-color-text-primary);
  font-size: 18px;
  letter-spacing: -0.03em;
}

.album-heading p:last-child {
  margin: 4px 0 0;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.album-heading-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.album-stats {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-right: 6px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.album-stats span {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
}

.album-stats strong {
  color: var(--ds-color-primary);
  font-size: 16px;
  font-variant-numeric: tabular-nums;
}

.album-file-input {
  display: none;
}

.album-upload-zone {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) minmax(220px, 280px);
  align-items: center;
  gap: 16px;
  margin-top: 22px;
  padding: 17px 18px;
  background: var(--ds-color-surface-muted);
  border: 1px dashed var(--ds-color-border-strong);
  border-radius: 13px;
  transition: background-color 160ms ease, border-color 160ms ease, transform 160ms ease;
}

.album-upload-zone.is-dragging {
  background: #f0f4fd;
  border-color: var(--ds-color-primary);
  transform: translateY(-1px);
}

.album-upload-icon,
.album-empty-icon {
  display: grid;
  place-items: center;
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
  border-radius: 13px;
}

.album-upload-icon {
  width: 44px;
  height: 44px;
}

.album-upload-icon svg {
  width: 21px;
}

.album-upload-copy {
  min-width: 0;
}

.album-upload-copy strong,
.album-upload-copy span {
  display: block;
}

.album-upload-copy strong {
  color: var(--ds-color-text-secondary);
  font-size: 13px;
}

.album-upload-copy span {
  margin-top: 3px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.album-upload-link {
  margin-top: 5px;
  padding: 0;
  color: var(--ds-color-primary);
  cursor: pointer;
  background: transparent;
  font-size: 11px;
}

.album-upload-link:hover {
  color: var(--ds-color-primary-hover);
  text-decoration: underline;
}

.album-upload-note label {
  display: block;
  margin-bottom: 5px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.album-upload-note label span {
  margin-left: 4px;
  color: var(--ds-color-text-disabled);
}

.album-upload-progress {
  display: grid;
  grid-template-columns: auto auto minmax(120px, 240px);
  align-items: center;
  gap: 9px;
  margin-top: 13px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.album-upload-progress svg {
  width: 15px;
  color: var(--ds-color-primary);
}

.album-upload-progress .el-progress {
  width: 100%;
}

.album-error {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-top: 14px;
  padding: 11px 14px;
  color: #8b5a26;
  font-size: 12px;
  background: #fff8e7;
  border: 1px solid #f2dfb2;
  border-radius: var(--ds-radius-md);
}

.album-error svg {
  width: 16px;
  flex: 0 0 auto;
}

.album-error .el-button {
  margin-left: auto;
}

.album-grid,
.album-loading-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
  margin-top: 22px;
}

.album-card {
  min-width: 0;
  overflow: hidden;
  background: #ffffff;
  border: 1px solid var(--ds-color-border);
  border-radius: 14px;
  box-shadow: 0 3px 12px rgba(37, 48, 71, 0.04);
  transition: transform 160ms ease, box-shadow 160ms ease;
}

.album-card:hover {
  box-shadow: var(--ds-shadow-card);
  transform: translateY(-2px);
}

.album-media-frame {
  position: relative;
  aspect-ratio: 4 / 3;
  overflow: hidden;
  background: #253047;
}

.album-media-frame img,
.album-media-frame video {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.album-media-frame img {
  cursor: zoom-in;
}

.album-media-frame video {
  background: #253047;
}

.album-media-badge {
  position: absolute;
  top: 10px;
  left: 10px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 7px;
  color: #ffffff;
  background: rgba(31, 42, 68, 0.72);
  border: 1px solid rgba(255, 255, 255, 0.18);
  border-radius: var(--ds-radius-pill);
  font-size: 10px;
  backdrop-filter: blur(6px);
}

.album-media-badge svg {
  width: 12px;
}

.album-media-preview-action {
  position: absolute;
  right: 10px;
  bottom: 10px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 8px;
  color: #ffffff;
  cursor: pointer;
  background: rgba(31, 42, 68, 0.76);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: var(--ds-radius-pill);
  font-size: 10px;
  opacity: 0;
  transition: opacity 160ms ease, background-color 160ms ease;
  backdrop-filter: blur(6px);
}

.album-media-frame:hover .album-media-preview-action,
.album-media-preview-action:focus-visible {
  opacity: 1;
}

.album-media-preview-action:hover {
  background: rgba(31, 42, 68, 0.94);
}

.album-media-preview-action svg {
  width: 12px;
}

.album-card-body {
  padding: 13px 14px 12px;
}

.album-card-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
  font-variant-numeric: tabular-nums;
}

.album-item-note {
  min-height: 42px;
  margin: 9px 0 5px;
  color: var(--ds-color-text-secondary);
  font-size: 12px;
  line-height: 1.65;
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.album-item-note.is-empty {
  color: var(--ds-color-text-disabled);
}

.album-file-name {
  overflow: hidden;
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.album-card-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 11px;
  padding-top: 9px;
  border-top: 1px solid #f0f2f6;
}

.album-card-actions button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0;
  color: var(--ds-color-text-tertiary);
  cursor: pointer;
  background: transparent;
  font-size: 11px;
}

.album-card-actions button:hover {
  color: var(--ds-color-primary);
}

.album-card-actions button:last-child:hover {
  color: var(--ds-color-danger);
}

.album-card-actions button:disabled {
  cursor: wait;
  opacity: 0.5;
}

.album-card-actions svg {
  width: 13px;
}

.album-loading-card {
  aspect-ratio: 0.83;
  background: linear-gradient(90deg, #f1f3f8 20%, #fafbfc 45%, #f1f3f8 70%);
  background-size: 200% 100%;
  border-radius: 14px;
  animation: album-loading 1.4s ease-in-out infinite;
}

@keyframes album-loading {
  from { background-position: 100% 0; }
  to { background-position: -100% 0; }
}

.album-empty-state {
  display: flex;
  min-height: 245px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 25px;
  text-align: center;
}

.album-empty-icon {
  width: 50px;
  height: 50px;
  margin-bottom: 13px;
}

.album-empty-icon svg {
  width: 24px;
}

.album-empty-state strong {
  font-size: 15px;
}

.album-empty-state p {
  margin: 6px 0 16px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.album-preview-overlay {
  position: fixed;
  z-index: 3000;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 32px;
  background: rgba(14, 21, 38, 0.88);
  backdrop-filter: blur(8px);
}

.album-preview-shell {
  position: relative;
  display: grid;
  width: min(1180px, 100%);
  max-height: calc(100vh - 64px);
  grid-template-rows: auto minmax(0, 1fr) auto;
  color: #ffffff;
}

.album-preview-toolbar,
.album-preview-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.album-preview-toolbar {
  min-height: 50px;
  padding: 0 4px 13px;
}

.album-preview-title {
  min-width: 0;
}

.album-preview-title strong,
.album-preview-title span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.album-preview-title strong {
  font-size: 15px;
}

.album-preview-title span {
  margin-top: 3px;
  color: #afbad1;
  font-size: 11px;
}

.album-preview-close {
  display: grid;
  width: 36px;
  height: 36px;
  flex: 0 0 auto;
  place-items: center;
  color: #ffffff;
  cursor: pointer;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 50%;
  transition: background-color 160ms ease, transform 160ms ease;
}

.album-preview-close:hover {
  background: rgba(255, 255, 255, 0.2);
  transform: rotate(4deg);
}

.album-preview-close svg {
  width: 18px;
}

.album-preview-stage {
  display: grid;
  min-height: 0;
  place-items: center;
  overflow: hidden;
  background: rgba(0, 0, 0, 0.24);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 16px;
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.32);
}

.album-preview-stage img,
.album-preview-stage video {
  display: block;
  width: 100%;
  height: min(72vh, 760px);
  object-fit: contain;
}

.album-preview-stage video {
  background: #0b1020;
}

.album-preview-nav {
  position: absolute;
  z-index: 1;
  top: 50%;
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  color: #ffffff;
  cursor: pointer;
  background: rgba(31, 42, 68, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.18);
  border-radius: 50%;
  transform: translateY(-50%);
  transition: background-color 160ms ease, transform 160ms ease;
}

.album-preview-nav:hover {
  background: rgba(82, 107, 155, 0.96);
}

.album-preview-nav.is-prev {
  left: -21px;
}

.album-preview-nav.is-next {
  right: -21px;
}

.album-preview-nav svg {
  width: 18px;
}

.album-preview-footer {
  padding: 12px 4px 0;
  color: #afbad1;
  font-size: 11px;
}

.album-preview-footer span:first-child {
  max-width: 60%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.album-preview-fade-enter-active,
.album-preview-fade-leave-active {
  transition: opacity 180ms ease;
}

.album-preview-fade-enter-from,
.album-preview-fade-leave-to {
  opacity: 0;
}

@media (max-width: 1180px) {
  .album-grid,
  .album-loading-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .album-heading {
    display: block;
  }

  .album-heading-actions {
    justify-content: flex-start;
    margin-top: 14px;
  }

  .album-upload-zone {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .album-upload-note {
    grid-column: 2;
  }
}

@media (max-width: 640px) {
  .doubao-album {
    padding: 18px;
  }

  .album-stats {
    width: 100%;
    margin-right: 0;
  }

  .album-upload-zone {
    grid-template-columns: 1fr;
    justify-items: start;
  }

  .album-upload-note {
    grid-column: auto;
    width: 100%;
  }

  .album-grid,
  .album-loading-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }

  .album-card-body {
    padding: 11px;
  }

  .album-media-preview-action {
    opacity: 1;
  }

  .album-preview-overlay {
    padding: 14px;
  }

  .album-preview-shell {
    max-height: calc(100vh - 28px);
  }

  .album-preview-stage img,
  .album-preview-stage video {
    height: 68vh;
  }

  .album-preview-nav {
    width: 34px;
    height: 34px;
  }

  .album-preview-nav.is-prev {
    left: 8px;
  }

  .album-preview-nav.is-next {
    right: 8px;
  }

  .album-preview-footer {
    display: block;
    line-height: 1.6;
  }

  .album-preview-footer span {
    display: block;
    max-width: 100% !important;
  }
}
</style>
