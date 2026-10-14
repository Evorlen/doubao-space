<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Collection,
  Delete,
  Film,
  Plus,
  Reading,
  Refresh,
  Search,
  Star,
  StarFilled,
  Upload,
  VideoCamera,
  WarningFilled,
} from '@element-plus/icons-vue'
import EntertainmentImportDialog from '../components/entertainment/EntertainmentImportDialog.vue'
import {
  createEntertainmentItem,
  deleteEntertainmentItem,
  getEntertainment,
  updateEntertainmentItem,
} from '../api'

const typeOptions = [
  { value: '', label: '全部' },
  { value: 'novel', label: '小说', icon: Reading },
  { value: 'series', label: '电视剧', icon: VideoCamera },
  { value: 'movie', label: '电影', icon: Film },
]

const statusValues = ['planned', 'in_progress', 'completed', 'paused', 'dropped']
const sortOptions = [
  { value: 'updated', label: '最近更新' },
  { value: 'created', label: '最近添加' },
  { value: 'rating', label: '评分最高' },
  { value: 'completed', label: '最近完成' },
  { value: 'title', label: '按名称' },
]

const statusLabels = {
  novel: {
    planned: '想读',
    in_progress: '在读',
    completed: '已读',
    paused: '搁置',
    dropped: '弃读',
  },
  series: {
    planned: '想看',
    in_progress: '在看',
    completed: '已看',
    paused: '搁置',
    dropped: '弃看',
  },
  movie: {
    planned: '想看',
    in_progress: '在看',
    completed: '已看',
    paused: '搁置',
    dropped: '弃看',
  },
}

const items = ref([])
const stats = ref({ total: 0, in_progress: 0, completed: 0, favorites: 0, unrated: 0 })
const loading = ref(true)
const loadError = ref('')
const searchQuery = ref('')
const typeFilter = ref('')
const statusFilter = ref('')
const ratingFilter = ref('')
const favoriteOnly = ref(false)
const sortKey = ref('updated')
const importDialogVisible = ref(false)
const drawerVisible = ref(false)
const editingId = ref(null)
const saving = ref(false)
const form = ref(createEmptyForm())

const isEditing = computed(() => Boolean(editingId.value))
const drawerTitle = computed(() => (isEditing.value ? '编辑作品' : '添加作品'))
const isMovie = computed(() => form.value.media_type === 'movie')
const formStatusOptions = computed(() => statusValues.map((value) => ({
  value,
  label: statusLabelFor(form.value.media_type, value),
})))
const recentCompleted = computed(() => items.value
  .filter((item) => item.status === 'completed')
  .sort((left, right) => dateValue(right.completed_on) - dateValue(left.completed_on))
  .slice(0, 3))
const favoriteItems = computed(() => items.value
  .filter((item) => item.favorite)
  .sort((left, right) => right.rating - left.rating)
  .slice(0, 3))

function createEmptyForm() {
  return {
    media_type: 'novel',
    title: '',
    creator: '',
    release_year: null,
    source: '',
    status: 'planned',
    progress_current: 0,
    progress_total: null,
    rating: 0,
    review: '',
    note: '',
    favorite: false,
    tags: [],
    started_on: '',
    completed_on: '',
  }
}

function statusLabelFor(mediaType, status) {
  return statusLabels[mediaType]?.[status] || statusLabels.movie[status] || status
}

function filterStatusLabel(status) {
  if (!typeFilter.value) {
    return {
      planned: '计划中',
      in_progress: '进行中',
      completed: '已完成',
      paused: '搁置',
      dropped: '已放弃',
    }[status]
  }
  return statusLabelFor(typeFilter.value, status)
}

function typeMeta(mediaType) {
  return typeOptions.find((option) => option.value === mediaType) || typeOptions[0]
}

function dateValue(value) {
  if (!value) return 0
  const parsed = new Date(value.length > 10 ? value.replace(' ', 'T') : `${value}T00:00:00`)
  return Number.isNaN(parsed.getTime()) ? 0 : parsed.getTime()
}

function formatDate(value) {
  if (!value) return '未记录'
  const date = value.slice(0, 10).split('-')
  return date.length === 3 ? `${date[0]}/${date[1]}/${date[2]}` : value
}

function formatUpdated(item) {
  return item.completed_on ? `完成于 ${formatDate(item.completed_on)}` : `更新于 ${formatDate(item.updated_at)}`
}

function ratingLabel(rating) {
  return rating ? `${rating} 星` : '未评分'
}

function progressLabel(item) {
  if (item.progress_label) return item.progress_label
  if (item.media_type === 'movie') return ''
  return '还没有记录进度'
}

async function loadItems() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await getEntertainment({
      type: typeFilter.value,
      status: statusFilter.value,
      search: searchQuery.value.trim(),
      rating: ratingFilter.value,
      favorite: favoriteOnly.value,
      sort: sortKey.value,
    })
    items.value = Array.isArray(response?.items) ? response.items : []
    stats.value = response?.stats || { total: 0, in_progress: 0, completed: 0, favorites: 0, unrated: 0 }
  } catch (error) {
    items.value = []
    loadError.value = error.message
  } finally {
    loading.value = false
  }
}

function clearFilters() {
  searchQuery.value = ''
  typeFilter.value = ''
  statusFilter.value = ''
  ratingFilter.value = ''
  favoriteOnly.value = false
  sortKey.value = 'updated'
}

function openCreate() {
  editingId.value = null
  form.value = createEmptyForm()
  drawerVisible.value = true
}

function openEdit(item) {
  editingId.value = item.id
  form.value = {
    media_type: item.media_type,
    title: item.title,
    creator: item.creator || '',
    release_year: item.release_year ?? null,
    source: item.source || '',
    status: item.status,
    progress_current: item.progress_current || 0,
    progress_total: item.progress_total ?? null,
    rating: item.rating || 0,
    review: item.review || '',
    note: item.note || '',
    favorite: Boolean(item.favorite),
    tags: [...(item.tags || [])],
    started_on: item.started_on || '',
    completed_on: item.completed_on || '',
  }
  drawerVisible.value = true
}

function handleTypeChange() {
  if (form.value.media_type === 'movie') {
    form.value.progress_current = 0
    form.value.progress_total = null
  }
  if (form.value.media_type !== 'novel' && form.value.status === 'planned') {
    form.value.status = 'planned'
  }
}

async function saveItem() {
  if (!form.value.title.trim()) {
    ElMessage.warning('请先填写作品名称。')
    return
  }
  if (form.value.progress_current < 0 || (form.value.progress_total && form.value.progress_current > form.value.progress_total)) {
    ElMessage.warning('请检查作品进度。')
    return
  }

  saving.value = true
  try {
    const payload = {
      ...form.value,
      title: form.value.title.trim(),
      creator: form.value.creator.trim(),
      source: form.value.source.trim(),
      release_year: form.value.release_year ? Number(form.value.release_year) : null,
      progress_current: Number(form.value.progress_current || 0),
      progress_total: isMovie.value || !form.value.progress_total ? null : Number(form.value.progress_total),
      rating: Number(form.value.rating || 0),
      review: form.value.review.trim(),
      note: form.value.note.trim(),
    }
    if (isEditing.value) {
      await updateEntertainmentItem(editingId.value, payload)
    } else {
      await createEntertainmentItem(payload)
    }
    drawerVisible.value = false
    await loadItems()
    ElMessage.success(isEditing.value ? '作品已更新。' : '作品已加入娱乐收藏。')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    saving.value = false
  }
}

async function removeItem(item) {
  try {
    await ElMessageBox.confirm(
      `删除“${item.title}”后，它的评分、评价和标签也会一起移除。`,
      '删除作品',
      { confirmButtonText: '删除', cancelButtonText: '保留', type: 'warning' },
    )
  } catch {
    return
  }

  try {
    await deleteEntertainmentItem(item.id)
    if (editingId.value === item.id) drawerVisible.value = false
    await loadItems()
    ElMessage.success('作品已删除。')
  } catch (error) {
    ElMessage.error(error.message)
  }
}

watch([typeFilter, statusFilter, ratingFilter, favoriteOnly, sortKey], loadItems)
onMounted(loadItems)
</script>

<template>
  <div class="page-view page-view-wide entertainment-page">
    <section class="page-heading">
      <div class="page-heading-copy">
        <p class="eyebrow">ENTERTAINMENT SPACE</p>
        <h1>娱乐</h1>
        <p>把想读、想看和已经喜欢的作品放在一起，慢慢积累自己的片单和书单。</p>
      </div>
      <div class="page-heading-actions">
        <el-button class="soft-button" :loading="loading" @click="loadItems">
          <Refresh class="button-icon" />刷新
        </el-button>
        <el-button class="soft-button" @click="importDialogVisible = true">
          <Upload class="button-icon" />导入 Excel
        </el-button>
        <el-button type="primary" @click="openCreate">
          <Plus class="button-icon" />添加作品
        </el-button>
      </div>
    </section>

    <div v-if="loadError" class="entertainment-error" role="alert">
      <WarningFilled />
      <span>娱乐内容暂时无法读取：{{ loadError }}</span>
      <el-button text type="primary" @click="loadItems">重新加载</el-button>
    </div>

    <section class="entertainment-summary-grid">
      <div class="surface-card entertainment-summary-card is-total">
        <div class="entertainment-summary-icon"><Collection /></div>
        <div><span>全部作品</span><strong>{{ stats.total }}</strong><small>小说、电视剧和电影</small></div>
      </div>
      <div class="surface-card entertainment-summary-card is-progress">
        <div class="entertainment-summary-icon"><VideoCamera /></div>
        <div><span>正在进行</span><strong>{{ stats.in_progress }}</strong><small>保持一点持续的兴趣</small></div>
      </div>
      <div class="surface-card entertainment-summary-card is-completed">
        <div class="entertainment-summary-icon"><StarFilled /></div>
        <div><span>已完成</span><strong>{{ stats.completed }}</strong><small>留下评分和评价</small></div>
      </div>
      <div class="surface-card entertainment-summary-card is-favorite">
        <div class="entertainment-summary-icon"><Star /></div>
        <div><span>收藏</span><strong>{{ stats.favorites }}</strong><small>{{ stats.unrated }} 部还没有评分</small></div>
      </div>
    </section>

    <section class="entertainment-layout">
      <div class="surface-card entertainment-main-card">
        <div class="entertainment-toolbar">
          <div class="entertainment-type-tabs" role="tablist" aria-label="作品类型">
            <button
              v-for="option in typeOptions"
              :key="option.value || 'all'"
              type="button"
              class="entertainment-type-tab"
              :class="{ 'is-active': typeFilter === option.value }"
              @click="typeFilter = option.value"
            >
              <component v-if="option.icon" :is="option.icon" />
              {{ option.label }}
            </button>
          </div>
          <div class="entertainment-toolbar-controls">
            <el-input
              v-model="searchQuery"
              clearable
              class="entertainment-search"
              placeholder="搜索标题、作者或评价"
              @keyup.enter="loadItems"
              @clear="loadItems"
            >
              <template #prefix><Search /></template>
            </el-input>
            <el-select v-model="sortKey" class="entertainment-sort" aria-label="排序方式">
              <el-option v-for="option in sortOptions" :key="option.value" :label="option.label" :value="option.value" />
            </el-select>
          </div>
        </div>

        <div class="entertainment-filter-row">
          <div class="entertainment-status-tabs" role="tablist" aria-label="作品状态">
            <button
              type="button"
              class="entertainment-status-tab"
              :class="{ 'is-active': !statusFilter }"
              @click="statusFilter = ''"
            >全部状态</button>
            <button
              v-for="status in statusValues"
              :key="status"
              type="button"
              class="entertainment-status-tab"
              :class="{ 'is-active': statusFilter === status }"
              @click="statusFilter = status"
            >{{ filterStatusLabel(status) }}</button>
          </div>
          <div class="entertainment-filter-actions">
            <el-select v-model="ratingFilter" clearable placeholder="最低评分" class="entertainment-rating-filter">
              <el-option v-for="rating in [5, 4, 3]" :key="rating" :label="`${rating} 星以上`" :value="rating" />
            </el-select>
            <el-checkbox v-model="favoriteOnly">只看收藏</el-checkbox>
            <el-button v-if="searchQuery || typeFilter || statusFilter || ratingFilter || favoriteOnly" text @click="clearFilters">清除筛选</el-button>
          </div>
        </div>

        <div v-if="loading" class="entertainment-loading-grid">
          <div v-for="line in 4" :key="line" class="entertainment-loading-card"></div>
        </div>
        <div v-else-if="items.length" class="entertainment-items-grid">
          <article
            v-for="item in items"
            :key="item.id"
            class="entertainment-item-card"
            tabindex="0"
            @click="openEdit(item)"
            @keydown.enter="openEdit(item)"
          >
            <div class="entertainment-item-topline">
              <span class="entertainment-type-mark" :class="`is-${item.media_type}`">
                <component :is="typeMeta(item.media_type).icon" />
                {{ item.media_type_label }}
              </span>
              <div class="entertainment-item-actions">
                <span v-if="item.favorite" class="entertainment-favorite" title="已收藏"><StarFilled /></span>
                <button type="button" title="删除作品" aria-label="删除作品" @click.stop="removeItem(item)"><Delete /></button>
              </div>
            </div>

            <h2>{{ item.title }}</h2>
            <p class="entertainment-item-creator">{{ item.creator || '未记录作者或导演' }}<template v-if="item.release_year"> · {{ item.release_year }}</template></p>

            <div v-if="item.progress_label" class="entertainment-progress-block">
              <div class="entertainment-progress-label"><span>{{ item.progress_label }}</span><span>{{ item.progress_percent }}%</span></div>
              <div class="entertainment-progress-track"><span :style="{ width: `${item.progress_percent}%` }"></span></div>
            </div>
            <div v-else class="entertainment-progress-placeholder">
              {{ item.media_type === 'movie' ? '电影无需记录进度' : progressLabel(item) }}
            </div>

            <div class="entertainment-item-rating">
              <span class="entertainment-stars" :aria-label="ratingLabel(item.rating)">
                <StarFilled v-for="star in item.rating" :key="`filled-${star}`" class="is-filled" />
                <Star v-for="star in 5 - item.rating" :key="`empty-${star}`" />
              </span>
              <span>{{ ratingLabel(item.rating) }}</span>
            </div>
            <p v-if="item.review" class="entertainment-item-review">“{{ item.review }}”</p>

            <div class="entertainment-item-footer">
              <span class="entertainment-status-pill" :class="`is-${item.status}`">{{ item.status_label }}</span>
              <span>{{ formatUpdated(item) }}</span>
            </div>
            <div v-if="item.tags?.length" class="entertainment-item-tags">
              <span v-for="tag in item.tags.slice(0, 3)" :key="tag">{{ tag }}</span>
              <span v-if="item.tags.length > 3">+{{ item.tags.length - 3 }}</span>
            </div>
          </article>
        </div>
        <div v-else class="entertainment-empty-state">
          <div class="entertainment-empty-icon"><Film /></div>
          <strong>{{ searchQuery || typeFilter || statusFilter ? '没有符合条件的作品' : '还没有收藏作品' }}</strong>
          <p>{{ searchQuery || typeFilter || statusFilter ? '换个关键词或清除筛选，再找找看。' : '先记录一本想读的书，或者一部想看的电影。' }}</p>
          <el-button type="primary" @click="searchQuery || typeFilter || statusFilter ? clearFilters() : openCreate">
            {{ searchQuery || typeFilter || statusFilter ? '清除筛选' : '添加第一部作品' }}
          </el-button>
        </div>
      </div>

      <aside class="entertainment-side-column">
        <div class="entertainment-inspiration-card">
          <div class="entertainment-inspiration-icon"><StarFilled /></div>
          <h2>给喜欢的东西留个位置</h2>
          <p>不急着看完，也可以先记下来。等下一次想起时，它还在这里。</p>
          <el-button type="primary" @click="openCreate">添加作品 <Plus class="entertainment-button-icon" /></el-button>
        </div>

        <div class="surface-card entertainment-side-card">
          <div class="entertainment-side-heading"><div><p class="entertainment-overline">RECENTLY FINISHED</p><h3>最近完成</h3></div><span>{{ recentCompleted.length }}</span></div>
          <div v-if="recentCompleted.length" class="entertainment-side-list">
            <button v-for="item in recentCompleted" :key="item.id" type="button" @click="openEdit(item)">
              <span class="entertainment-side-item-icon" :class="`is-${item.media_type}`"><component :is="typeMeta(item.media_type).icon" /></span>
              <span><strong>{{ item.title }}</strong><small>{{ item.status_label }} · {{ ratingLabel(item.rating) }}</small></span>
            </button>
          </div>
          <p v-else class="entertainment-side-empty">完成作品后，会在这里留下记录。</p>
        </div>

        <div class="surface-card entertainment-side-card">
          <div class="entertainment-side-heading"><div><p class="entertainment-overline">FAVORITES</p><h3>我的收藏</h3></div><span>{{ favoriteItems.length }}</span></div>
          <div v-if="favoriteItems.length" class="entertainment-side-list">
            <button v-for="item in favoriteItems" :key="item.id" type="button" @click="openEdit(item)">
              <span class="entertainment-side-item-icon is-favorite"><StarFilled /></span>
              <span><strong>{{ item.title }}</strong><small>{{ item.media_type_label }} · {{ ratingLabel(item.rating) }}</small></span>
            </button>
          </div>
          <p v-else class="entertainment-side-empty">在作品详情里点亮收藏。</p>
        </div>
      </aside>
    </section>

    <el-drawer v-model="drawerVisible" :title="drawerTitle" direction="rtl" size="620px" destroy-on-close>
      <div class="entertainment-drawer-intro">把作品放进自己的列表里，之后再慢慢补上进度和评价。</div>
      <el-form class="entertainment-form" label-position="top" @submit.prevent="saveItem">
        <div class="entertainment-form-row">
          <el-form-item label="作品类型" required>
            <el-select v-model="form.media_type" class="full-width" @change="handleTypeChange">
              <el-option v-for="option in typeOptions.slice(1)" :key="option.value" :label="option.label" :value="option.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="当前状态" required>
            <el-select v-model="form.status" class="full-width">
              <el-option v-for="option in formStatusOptions" :key="option.value" :label="option.label" :value="option.value" />
            </el-select>
          </el-form-item>
        </div>

        <el-form-item label="作品名称" required>
          <el-input v-model="form.title" maxlength="120" show-word-limit placeholder="例如：海街日记、夜航西飞" />
        </el-form-item>

        <div class="entertainment-form-row">
          <el-form-item :label="form.media_type === 'novel' ? '作者' : '导演'">
            <el-input v-model="form.creator" maxlength="100" placeholder="可以稍后补充" />
          </el-form-item>
          <el-form-item label="年份">
            <el-input-number v-model="form.release_year" :min="1000" :max="9999" :controls="false" class="full-width" placeholder="可选" />
          </el-form-item>
        </div>

        <el-form-item label="来源">
          <el-input v-model="form.source" maxlength="100" placeholder="例如：微信读书、Netflix、纸书" />
        </el-form-item>

        <div v-if="!isMovie" class="entertainment-form-row">
          <el-form-item :label="form.media_type === 'novel' ? '当前章节' : '当前集数'">
            <el-input-number v-model="form.progress_current" :min="0" :max="100000" controls-position="right" class="full-width" />
          </el-form-item>
          <el-form-item :label="form.media_type === 'novel' ? '总章节' : '总集数'">
            <el-input-number v-model="form.progress_total" :min="1" :max="100000" controls-position="right" class="full-width" placeholder="可选" />
          </el-form-item>
        </div>

        <el-form-item label="评分">
          <div class="entertainment-rate-control">
            <el-rate v-model="form.rating" :max="5" :texts="['不太喜欢', '一般', '还不错', '很喜欢', '强烈推荐']" show-text />
            <span v-if="form.rating === 0" class="entertainment-rate-hint">还没有评分</span>
          </div>
        </el-form-item>

        <el-form-item label="标签">
          <el-select v-model="form.tags" class="full-width" multiple filterable allow-create default-first-option placeholder="添加标签，例如：悬疑、治愈">
            <el-option v-for="tag in form.tags" :key="tag" :label="tag" :value="tag" />
          </el-select>
        </el-form-item>

        <div class="entertainment-form-row">
          <el-form-item label="开始日期">
            <el-date-picker v-model="form.started_on" type="date" value-format="YYYY-MM-DD" placeholder="可选" class="full-width" />
          </el-form-item>
          <el-form-item label="完成日期">
            <el-date-picker v-model="form.completed_on" type="date" value-format="YYYY-MM-DD" placeholder="完成时自动记录" class="full-width" />
          </el-form-item>
        </div>

        <el-form-item label="个人评价">
          <el-input v-model="form.review" type="textarea" :rows="5" maxlength="2000" show-word-limit placeholder="看完或读完之后，想留下什么？" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.note" type="textarea" :rows="3" maxlength="500" show-word-limit placeholder="记录版本、观看进度或下一步想法" />
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="form.favorite">加入我的收藏</el-checkbox>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="entertainment-drawer-footer">
          <el-button v-if="isEditing" text type="danger" @click="removeItem({ id: editingId, title: form.title })">删除作品</el-button>
          <span></span>
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="saveItem">保存作品</el-button>
        </div>
      </template>
    </el-drawer>

    <EntertainmentImportDialog v-model="importDialogVisible" @imported="loadItems" />
  </div>
</template>

<style scoped>
.entertainment-page {
  max-width: 1440px;
}

.entertainment-error {
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

.entertainment-error > svg {
  width: 16px;
  flex: 0 0 auto;
}

.entertainment-error .el-button {
  margin-left: auto;
}

.entertainment-summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 18px;
}

.entertainment-summary-card {
  display: flex;
  align-items: center;
  gap: 13px;
  min-height: 105px;
  padding: 18px;
}

.entertainment-summary-icon {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 12px;
}

.entertainment-summary-icon svg {
  width: 19px;
}

.entertainment-summary-card.is-total .entertainment-summary-icon {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.entertainment-summary-card.is-progress .entertainment-summary-icon {
  color: #5d9076;
  background: #e8f4ed;
}

.entertainment-summary-card.is-completed .entertainment-summary-icon {
  color: #a4772e;
  background: #fff1ca;
}

.entertainment-summary-card.is-favorite .entertainment-summary-icon {
  color: #bb737a;
  background: #fbe7e8;
}

.entertainment-summary-card span,
.entertainment-summary-card strong,
.entertainment-summary-card small {
  display: block;
}

.entertainment-summary-card span {
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.entertainment-summary-card strong {
  margin: 2px 0 1px;
  color: var(--ds-color-text-primary);
  font-size: 23px;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.03em;
}

.entertainment-summary-card small {
  overflow: hidden;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.entertainment-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  gap: 18px;
}

.entertainment-main-card {
  min-width: 0;
  min-height: 580px;
  padding: 22px;
}

.entertainment-toolbar,
.entertainment-filter-row,
.entertainment-side-heading,
.entertainment-drawer-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}

.entertainment-toolbar {
  padding-bottom: 18px;
  border-bottom: 1px solid var(--ds-color-border);
}

.entertainment-type-tabs,
.entertainment-status-tabs,
.entertainment-toolbar-controls,
.entertainment-filter-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.entertainment-type-tab,
.entertainment-status-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 11px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  cursor: pointer;
  background: transparent;
  border-radius: 8px;
}

.entertainment-type-tab svg {
  width: 15px;
}

.entertainment-type-tab:hover,
.entertainment-type-tab.is-active,
.entertainment-status-tab:hover,
.entertainment-status-tab.is-active {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.entertainment-type-tab.is-active,
.entertainment-status-tab.is-active {
  font-weight: 700;
}

.entertainment-search {
  width: 220px;
}

.entertainment-sort {
  width: 118px;
}

.entertainment-filter-row {
  align-items: flex-start;
  padding: 15px 0 2px;
}

.entertainment-status-tabs {
  flex-wrap: wrap;
}

.entertainment-status-tab {
  padding: 6px 9px;
  font-size: 11px;
}

.entertainment-filter-actions {
  flex: 0 0 auto;
}

.entertainment-rating-filter {
  width: 118px;
}

.entertainment-items-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  padding-top: 18px;
}

.entertainment-item-card {
  min-width: 0;
  min-height: 250px;
  padding: 17px;
  cursor: pointer;
  background: var(--ds-color-surface-muted);
  border: 1px solid transparent;
  border-radius: 13px;
  transition: border-color 160ms ease, transform 160ms ease, box-shadow 160ms ease;
}

.entertainment-item-card:hover,
.entertainment-item-card:focus-visible {
  border-color: var(--ds-color-border-strong);
  box-shadow: var(--ds-shadow-card);
  transform: translateY(-2px);
}

.entertainment-item-topline,
.entertainment-item-footer,
.entertainment-item-rating,
.entertainment-progress-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.entertainment-type-mark {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 7px;
  font-size: 10px;
  font-weight: 700;
  border-radius: 7px;
}

.entertainment-type-mark svg,
.entertainment-favorite svg {
  width: 14px;
}

.entertainment-type-mark.is-novel {
  color: #7a6c9f;
  background: #f0ebfa;
}

.entertainment-type-mark.is-series {
  color: #527d82;
  background: #e7f4f2;
}

.entertainment-type-mark.is-movie {
  color: #a4772e;
  background: #fff3d7;
}

.entertainment-item-actions {
  display: flex;
  align-items: center;
  gap: 3px;
}

.entertainment-item-actions button {
  display: grid;
  width: 27px;
  height: 27px;
  place-items: center;
  color: var(--ds-color-text-tertiary);
  cursor: pointer;
  background: transparent;
  border-radius: 7px;
}

.entertainment-item-actions button:hover {
  color: var(--ds-color-danger);
  background: #fff1f1;
}

.entertainment-item-actions button svg {
  width: 14px;
}

.entertainment-favorite {
  display: grid;
  width: 27px;
  height: 27px;
  place-items: center;
  color: #bd7a7e;
}

.entertainment-item-card h2 {
  display: -webkit-box;
  margin: 17px 0 4px;
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-size: 16px;
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.entertainment-item-creator,
.entertainment-item-review,
.entertainment-progress-placeholder {
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.entertainment-item-creator {
  overflow: hidden;
  margin: 0;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.entertainment-progress-block {
  margin-top: 17px;
}

.entertainment-progress-label {
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
}

.entertainment-progress-track {
  height: 5px;
  margin-top: 7px;
  overflow: hidden;
  background: #e7ebf2;
  border-radius: 99px;
}

.entertainment-progress-track span {
  display: block;
  height: 100%;
  background: var(--ds-color-primary);
  border-radius: inherit;
}

.entertainment-progress-placeholder {
  min-height: 18px;
  margin-top: 17px;
  color: #a0aabc;
}

.entertainment-item-rating {
  justify-content: flex-start;
  margin-top: 14px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.entertainment-stars {
  display: inline-flex;
  gap: 2px;
  color: #d4d9e3;
}

.entertainment-stars svg {
  width: 13px;
}

.entertainment-stars svg.is-filled {
  color: #d29c42;
}

.entertainment-item-review {
  display: -webkit-box;
  min-height: 34px;
  margin: 10px 0 0;
  overflow: hidden;
  line-height: 1.55;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.entertainment-item-footer {
  margin-top: 15px;
  color: #a0aabc;
  font-size: 10px;
}

.entertainment-status-pill {
  padding: 4px 7px;
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
  border-radius: 6px;
}

.entertainment-status-pill.is-in_progress {
  color: #55836c;
  background: #e8f4ed;
}

.entertainment-status-pill.is-completed {
  color: #9a7029;
  background: #fff2d1;
}

.entertainment-status-pill.is-paused,
.entertainment-status-pill.is-dropped {
  color: var(--ds-color-text-tertiary);
  background: #edf0f5;
}

.entertainment-item-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 11px;
}

.entertainment-item-tags span {
  padding: 3px 6px;
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
  background: #ffffff;
  border-radius: 5px;
}

.entertainment-empty-state,
.entertainment-side-empty {
  color: var(--ds-color-text-tertiary);
  text-align: center;
}

.entertainment-empty-state {
  display: flex;
  min-height: 390px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.entertainment-empty-icon {
  display: grid;
  width: 46px;
  height: 46px;
  margin-bottom: 12px;
  place-items: center;
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
  border-radius: 14px;
}

.entertainment-empty-icon svg {
  width: 22px;
}

.entertainment-empty-state strong {
  color: var(--ds-color-text-primary);
  font-size: 14px;
}

.entertainment-empty-state p {
  max-width: 300px;
  margin: 6px 0 16px;
  font-size: 12px;
  line-height: 1.65;
}

.entertainment-side-column {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.entertainment-inspiration-card {
  padding: 22px;
  color: #ffffff;
  background: linear-gradient(145deg, #5e628c, #8075a0);
  border-radius: var(--ds-radius-lg);
  box-shadow: var(--ds-shadow-hero);
}

.entertainment-inspiration-icon {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  color: #8c6d36;
  background: var(--ds-color-accent);
  border-radius: 12px;
}

.entertainment-inspiration-icon svg {
  width: 19px;
}

.entertainment-inspiration-card h2 {
  margin: 19px 0 7px;
  font-size: 18px;
  line-height: 1.35;
}

.entertainment-inspiration-card p {
  margin: 0;
  color: #eeeaf8;
  font-size: 12px;
  line-height: 1.75;
}

.entertainment-inspiration-card .el-button {
  margin-top: 19px;
  color: var(--ds-color-brand-900);
  background: var(--ds-color-accent);
  border-color: var(--ds-color-accent);
}

.entertainment-button-icon {
  width: 14px;
  margin-left: 5px;
}

.entertainment-side-card {
  padding: 18px;
}

.entertainment-side-heading {
  align-items: flex-start;
}

.entertainment-side-heading h3 {
  margin: 0;
  color: var(--ds-color-text-primary);
  font-size: 16px;
}

.entertainment-side-heading > span {
  min-width: 22px;
  padding: 4px 6px;
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
  text-align: center;
  background: var(--ds-color-surface-muted);
  border-radius: 6px;
}

.entertainment-overline {
  margin: 0 0 4px;
  color: var(--ds-color-primary);
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.13em;
}

.entertainment-side-list {
  display: grid;
  gap: 5px;
  margin-top: 15px;
}

.entertainment-side-list button {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 9px;
  padding: 7px 0;
  color: inherit;
  text-align: left;
  cursor: pointer;
  background: transparent;
  border-bottom: 1px solid #eef0f5;
}

.entertainment-side-list button:last-child {
  border-bottom: 0;
}

.entertainment-side-list button:hover strong {
  color: var(--ds-color-primary);
}

.entertainment-side-item-icon {
  display: grid;
  width: 28px;
  height: 28px;
  flex: 0 0 auto;
  place-items: center;
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
  border-radius: 8px;
}

.entertainment-side-item-icon.is-series {
  color: #527d82;
  background: #e7f4f2;
}

.entertainment-side-item-icon.is-movie {
  color: #a4772e;
  background: #fff3d7;
}

.entertainment-side-item-icon.is-favorite {
  color: #bd7a7e;
  background: #fbe7e8;
}

.entertainment-side-item-icon svg {
  width: 14px;
}

.entertainment-side-list button > span:last-child {
  min-width: 0;
}

.entertainment-side-list strong,
.entertainment-side-list small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.entertainment-side-list strong {
  color: var(--ds-color-text-primary);
  font-size: 12px;
}

.entertainment-side-list small {
  margin-top: 2px;
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
}

.entertainment-side-empty {
  margin: 16px 0 2px;
  font-size: 11px;
  line-height: 1.6;
}

.entertainment-loading-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  padding-top: 18px;
}

.entertainment-loading-card {
  height: 250px;
  background: linear-gradient(90deg, #f2f4f8 20%, #fafbfc 45%, #f2f4f8 70%);
  background-size: 200% 100%;
  border-radius: 13px;
  animation: entertainment-loading 1.4s ease-in-out infinite;
}

@keyframes entertainment-loading {
  from { background-position: 100% 0; }
  to { background-position: -100% 0; }
}

.entertainment-drawer-intro {
  margin: -6px 0 20px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  line-height: 1.6;
}

.entertainment-form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.entertainment-rate-control {
  display: flex;
  align-items: center;
  gap: 10px;
}

.entertainment-rate-control .el-rate {
  height: 25px;
}

.entertainment-rate-hint {
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.entertainment-drawer-footer > span {
  flex: 1;
}

@media (max-width: 1180px) {
  .entertainment-summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .entertainment-layout {
    grid-template-columns: 1fr;
  }

  .entertainment-side-column {
    display: grid;
    grid-template-columns: 1fr 1fr;
  }

  .entertainment-inspiration-card {
    grid-row: span 2;
  }
}

@media (max-width: 800px) {
  .entertainment-toolbar,
  .entertainment-filter-row {
    display: block;
  }

  .entertainment-toolbar-controls,
  .entertainment-filter-actions {
    margin-top: 12px;
  }

  .entertainment-search {
    width: 100%;
  }

  .entertainment-sort {
    flex: 0 0 118px;
  }

  .entertainment-filter-actions {
    flex-wrap: wrap;
  }
}

@media (max-width: 620px) {
  .entertainment-summary-grid,
  .entertainment-items-grid,
  .entertainment-loading-grid,
  .entertainment-side-column {
    grid-template-columns: 1fr;
  }

  .entertainment-inspiration-card {
    grid-row: auto;
  }

  .entertainment-form-row {
    grid-template-columns: 1fr;
    gap: 0;
  }
}
</style>
