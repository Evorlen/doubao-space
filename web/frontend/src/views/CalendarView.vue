<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Apple,
  Calendar,
  Check,
  Delete,
  Edit,
  Plus,
  Refresh,
  WarningFilled,
} from '@element-plus/icons-vue'
import {
  createRecipeFood,
  deleteRecipeFood,
  getRecipe,
  updateRecipeFood,
} from '../api'
import SchedulePanel from '../components/calendar/SchedulePanel.vue'

const categoryOptions = ['主食', '营养补充', '零食', '护理', '其他']
const weekdayLabels = ['', '周一', '周二', '周三', '周四', '周五', '周六', '周日']

const plan = ref({ foods: [], weekdays: [], today: 1 })
const loading = ref(true)
const loadError = ref('')
const dialogVisible = ref(false)
const editingId = ref(null)
const saving = ref(false)
const savingFoodId = ref(null)
const deletingFoodId = ref(null)
const form = ref(createEmptyForm())

const foods = computed(() => plan.value.foods || [])
const weekdays = computed(() => {
  if (plan.value.weekdays?.length) return plan.value.weekdays
  return buildFallbackWeekdays()
})
const today = computed(() => plan.value.today || getLocalWeekday())
const todayMeta = computed(() => weekdays.value.find((day) => day.value === today.value) || weekdays.value[0])
const todayLabel = computed(() => todayMeta.value?.label || '今天')
const todayFoods = computed(() => foods.value.filter((food) => food.days.includes(today.value)))
const scheduledCount = computed(() => foods.value.reduce((total, food) => total + food.days.length, 0))
const weekRange = computed(() => {
  if (!plan.value.week_start || !plan.value.week_end) return '本周循环食谱'
  return `${formatDate(plan.value.week_start)} — ${formatDate(plan.value.week_end)}`
})
const dialogTitle = computed(() => (editingId.value ? '编辑食物' : '添加食物'))

function createEmptyForm() {
  return {
    name: '',
    category: '其他',
    note: '',
    days: [1, 2, 3, 4, 5, 6, 7],
  }
}

function getLocalWeekday() {
  const value = new Date().getDay()
  return value === 0 ? 7 : value
}

function buildFallbackWeekdays() {
  const now = new Date()
  const current = getLocalWeekday()
  const monday = new Date(now)
  monday.setDate(now.getDate() - current + 1)
  return Array.from({ length: 7 }, (_, index) => {
    const date = new Date(monday)
    date.setDate(monday.getDate() + index)
    return {
      value: index + 1,
      label: weekdayLabels[index + 1],
      date: date.toISOString().slice(0, 10),
      date_label: `${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')}`,
      is_today: index + 1 === current,
    }
  })
}

function formatDate(value) {
  const parts = value.split('-')
  return parts.length === 3 ? `${parts[1]}/${parts[2]}` : value
}

function isScheduled(food, day) {
  return food.days.includes(day.value)
}

function categoryClass(category) {
  return {
    主食: 'is-main',
    '营养补充': 'is-supplement',
    零食: 'is-snack',
    护理: 'is-care',
  }[category] || 'is-other'
}

async function loadPlan() {
  loading.value = true
  loadError.value = ''
  try {
    plan.value = await getRecipe()
  } catch (error) {
    plan.value = { foods: [], weekdays: [], today: getLocalWeekday() }
    loadError.value = error.message
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.value = createEmptyForm()
  dialogVisible.value = true
}

function openEdit(food) {
  editingId.value = food.id
  form.value = {
    name: food.name,
    category: food.category || '其他',
    note: food.note || '',
    days: [...food.days],
  }
  dialogVisible.value = true
}

function toggleFormDay(day) {
  const days = new Set(form.value.days)
  if (days.has(day)) days.delete(day)
  else days.add(day)
  form.value.days = [...days].sort((left, right) => left - right)
}

async function saveFood() {
  if (!form.value.name.trim()) {
    ElMessage.warning('请先填写食物名称。')
    return
  }
  if (!form.value.days.length) {
    ElMessage.warning('请至少选择一天。')
    return
  }

  saving.value = true
  try {
    const payload = {
      name: form.value.name.trim(),
      category: form.value.category,
      note: form.value.note.trim(),
      days: [...form.value.days],
    }
    if (editingId.value) await updateRecipeFood(editingId.value, payload)
    else await createRecipeFood(payload)
    await loadPlan()
    dialogVisible.value = false
    ElMessage.success(editingId.value ? '食物安排已更新。' : '食物已加入食谱。')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    saving.value = false
  }
}

async function toggleDay(food, day) {
  const originalDays = [...food.days]
  const nextDays = originalDays.includes(day.value)
    ? originalDays.filter((value) => value !== day.value)
    : [...originalDays, day.value].sort((left, right) => left - right)
  if (!nextDays.length) {
    ElMessage.warning('每种食物至少要安排一天；如果不再需要，可以直接删除。')
    return
  }

  food.days = nextDays
  savingFoodId.value = food.id
  try {
    const updated = await updateRecipeFood(food.id, {
      name: food.name,
      category: food.category,
      note: food.note,
      days: nextDays,
    })
    Object.assign(food, updated)
  } catch (error) {
    food.days = originalDays
    ElMessage.error(error.message)
  } finally {
    savingFoodId.value = null
  }
}

async function removeFood(food) {
  try {
    await ElMessageBox.confirm(
      `删除“${food.name}”后，它在每周食谱中的安排也会一起移除。`,
      '删除食物',
      { confirmButtonText: '删除', cancelButtonText: '保留', type: 'warning' },
    )
  } catch {
    return
  }

  deletingFoodId.value = food.id
  try {
    await deleteRecipeFood(food.id)
    await loadPlan()
    ElMessage.success('食物已从食谱中移除。')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    deletingFoodId.value = null
  }
}

onMounted(loadPlan)
</script>

<template>
  <div class="page-view page-view-wide recipe-page">
    <section class="page-heading recipe-heading">
      <div class="page-heading-copy">
        <p class="eyebrow">DOUBAO SPACE</p>
        <h1>豆包</h1>
        <p>记录豆包的食谱和周期日程，把每天的照顾安排得轻松一点。</p>
      </div>
      <div class="page-heading-actions">
        <el-button class="soft-button" :loading="loading" @click="loadPlan">
          <Refresh class="button-icon" />刷新食谱
        </el-button>
        <el-button type="primary" @click="openCreate">
          <Plus class="button-icon" />添加食物
        </el-button>
      </div>
    </section>

    <section class="recipe-welcome">
      <div class="cat-avatar" aria-hidden="true">
        <span class="cat-ear cat-ear-left"></span>
        <span class="cat-ear cat-ear-right"></span>
        <span class="cat-eye cat-eye-left"></span>
        <span class="cat-eye cat-eye-right"></span>
        <span class="cat-nose"></span>
        <span class="cat-whisker cat-whisker-left"></span>
        <span class="cat-whisker cat-whisker-right"></span>
      </div>
      <div class="recipe-welcome-copy">
        <div class="recipe-welcome-kicker">DOUBAO'S WEEKLY RHYTHM</div>
        <h2>把豆包的日常照顾，安排得刚刚好。</h2>
        <p>食谱和周期日程放在一起，今天该吃什么、该做什么一眼就能看到。</p>
      </div>
      <div class="today-summary-box">
        <span>今天</span>
        <strong>{{ todayLabel }}</strong>
        <small>{{ todayMeta?.date_label || '正在读取日期' }}</small>
      </div>
    </section>

    <div v-if="loadError" class="recipe-error" role="alert">
      <WarningFilled />
      <span>食谱暂时无法读取：{{ loadError }}</span>
      <el-button text type="primary" @click="loadPlan">重新加载</el-button>
    </div>

    <SchedulePanel />

    <section class="recipe-overview-grid">
      <div class="surface-card today-card">
        <div class="card-heading">
          <div>
            <p class="card-overline">TODAY'S BOWL</p>
            <h2>{{ todayLabel }}吃什么</h2>
            <p>{{ todayMeta?.date_label || '' }} · {{ todayFoods.length }} 项安排</p>
          </div>
          <div class="today-card-icon"><Apple /></div>
        </div>

        <div v-if="loading" class="today-loading">正在读取豆包的今日食谱…</div>
        <div v-else-if="todayFoods.length" class="today-food-list">
          <div v-for="food in todayFoods" :key="food.id" class="today-food-item">
            <span class="today-food-check"><Check /></span>
            <div>
              <strong>{{ food.name }}</strong>
              <span>{{ food.category }}<template v-if="food.note"> · {{ food.note }}</template></span>
            </div>
          </div>
        </div>
        <div v-else class="today-empty">
          <span>今天还没有安排食物。</span>
          <el-button text type="primary" @click="openCreate">添加第一项</el-button>
        </div>
      </div>

      <div class="surface-card rhythm-card">
        <div class="card-heading">
          <div>
            <p class="card-overline">WEEKLY PLAN</p>
            <h2>这周的节奏</h2>
            <p>{{ weekRange }}</p>
          </div>
          <Calendar class="rhythm-card-icon" />
        </div>
        <div class="rhythm-stats">
          <div><strong>{{ foods.length }}</strong><span>种食物</span></div>
          <div><strong>{{ scheduledCount }}</strong><span>次安排</span></div>
          <div><strong>7</strong><span>天循环</span></div>
        </div>
        <div class="rhythm-note">
          <span class="rhythm-note-dot"></span>
          <span>今天的列会保持明亮，其他日期轻轻退后。</span>
        </div>
      </div>
    </section>

    <section id="doubao-recipe" class="surface-card recipe-table-card">
      <div class="card-heading recipe-table-heading">
        <div>
          <p class="card-overline">WEEKLY RECIPE</p>
          <h2>豆包每周食谱</h2>
          <p>点击单元格即可调整某种食物在哪些天出现。</p>
        </div>
        <div class="table-hint"><span></span>今天突出显示</div>
      </div>

      <div v-if="loading" class="recipe-table-loading">
        <div class="loading-line loading-line-wide"></div>
        <div v-for="line in 4" :key="line" class="loading-line"></div>
      </div>
      <div v-else-if="foods.length === 0" class="recipe-empty-state">
        <div class="empty-apple"><Apple /></div>
        <strong>还没有豆包食谱</strong>
        <p>先添加豆包常吃的食物，再为它选择每周的日子。</p>
        <el-button type="primary" @click="openCreate"><Plus class="button-icon" />添加第一种食物</el-button>
      </div>
      <div v-else class="recipe-table-scroll">
        <div class="recipe-table">
          <div class="recipe-table-row recipe-table-header">
            <div class="food-column-heading">食物 / 备注</div>
            <div
              v-for="day in weekdays"
              :key="day.value"
              class="recipe-day-heading"
              :class="{ 'is-today': day.value === today }"
            >
              <strong>{{ day.label }}</strong>
              <small>{{ day.date_label }}</small>
              <em v-if="day.value === today">今天</em>
            </div>
          </div>

          <div v-for="food in foods" :key="food.id" class="recipe-table-row recipe-table-body-row">
            <div class="food-cell">
              <span class="food-color-mark" :class="categoryClass(food.category)"></span>
              <div class="food-cell-copy">
                <div class="food-name-line">
                  <strong>{{ food.name }}</strong>
                  <span class="food-category" :class="categoryClass(food.category)">{{ food.category }}</span>
                </div>
                <small>{{ food.note || '还没有备注' }}</small>
              </div>
              <div class="food-actions">
                <button type="button" title="编辑食物" aria-label="编辑食物" @click="openEdit(food)">
                  <Edit />
                </button>
                <button
                  type="button"
                  title="删除食物"
                  aria-label="删除食物"
                  :disabled="deletingFoodId === food.id"
                  @click="removeFood(food)"
                >
                  <Delete />
                </button>
              </div>
            </div>
            <button
              v-for="day in weekdays"
              :key="`${food.id}-${day.value}`"
              type="button"
              class="recipe-cell"
              :class="{
                'is-today': day.value === today,
                'is-other-day': day.value !== today,
                'is-scheduled': isScheduled(food, day),
              }"
              :aria-label="`${food.name}，${day.label}${isScheduled(food, day) ? '已安排' : '未安排'}`"
              :aria-pressed="isScheduled(food, day)"
              :disabled="savingFoodId === food.id"
              @click="toggleDay(food, day)"
            >
              <span v-if="isScheduled(food, day)" class="scheduled-mark"><Check /></span>
              <span v-else class="empty-mark"></span>
            </button>
          </div>
        </div>
      </div>

      <div class="recipe-table-footer">
        <span><span class="footer-mark is-scheduled"></span>已安排</span>
        <span><span class="footer-mark is-today"></span>今日列</span>
        <span class="recipe-table-footer-copy">至少为每种食物保留一天安排</span>
      </div>
    </section>

    <section class="recipe-note-strip">
      <div class="recipe-note-icon"><Calendar /></div>
      <div>
        <strong>食谱是豆包日常的第一层</strong>
        <p>后续还可以在这里继续加入喂食提醒、驱虫和护理安排。</p>
      </div>
      <span>持续完善中</span>
    </section>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="520px" destroy-on-close>
      <div class="recipe-dialog-intro">记录一种食物，再告诉豆包每周哪几天可以吃。</div>
      <el-form class="recipe-form" label-position="top" @submit.prevent="saveFood">
        <el-form-item label="食物名称" required>
          <el-input v-model="form.name" maxlength="100" show-word-limit placeholder="例如：鱼油、罐头、化毛片" />
        </el-form-item>
        <el-form-item label="食物分类">
          <el-select v-model="form.category" class="recipe-form-control">
            <el-option v-for="category in categoryOptions" :key="category" :label="category" :value="category" />
          </el-select>
        </el-form-item>
        <el-form-item label="每周安排" required>
          <div class="form-day-picker">
            <button
              v-for="day in weekdays"
              :key="day.value"
              type="button"
              class="form-day-button"
              :class="{ 'is-selected': form.days.includes(day.value) }"
              :aria-pressed="form.days.includes(day.value)"
              @click="toggleFormDay(day.value)"
            >
              <strong>{{ day.label }}</strong>
              <small>{{ day.date_label }}</small>
            </button>
          </div>
          <div class="form-help">可以选择一天或多天；保存后也能直接点击表格调整。</div>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.note" type="textarea" :rows="3" maxlength="300" show-word-limit placeholder="例如：晚餐后半粒，或需要拌在罐头里" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveFood">保存食谱</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.recipe-page {
  max-width: 1440px;
}

.recipe-heading {
  margin-bottom: 24px;
}

.recipe-welcome {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 22px;
  min-height: 146px;
  padding: 28px 32px;
  color: #ffffff;
  background: linear-gradient(112deg, #273858 0%, #3c5279 58%, #5f789f 100%);
  border-radius: var(--ds-radius-xl);
  box-shadow: var(--ds-shadow-hero);
  overflow: hidden;
}

.cat-avatar {
  position: relative;
  width: 82px;
  height: 82px;
  flex: 0 0 auto;
  background: #f8d487;
  border: 5px solid rgba(255, 255, 255, 0.3);
  border-radius: 31px 31px 35px 35px;
  box-shadow: 0 12px 22px rgba(18, 31, 58, 0.22);
}

.cat-ear {
  position: absolute;
  top: -14px;
  width: 30px;
  height: 31px;
  background: #f8d487;
  border: 5px solid rgba(255, 255, 255, 0.3);
  border-bottom: 0;
  border-radius: 16px 16px 3px 3px;
  transform: rotate(29deg);
}

.cat-ear-left {
  left: 4px;
}

.cat-ear-right {
  right: 4px;
  transform: rotate(-29deg);
}

.cat-eye {
  position: absolute;
  top: 30px;
  width: 7px;
  height: 10px;
  background: #334363;
  border-radius: 50%;
}

.cat-eye-left {
  left: 22px;
}

.cat-eye-right {
  right: 22px;
}

.cat-nose {
  position: absolute;
  top: 48px;
  left: 35px;
  width: 10px;
  height: 8px;
  background: #d78775;
  border-radius: 50% 50% 45% 45%;
}

.cat-whisker {
  position: absolute;
  top: 53px;
  width: 20px;
  height: 1px;
  background: rgba(51, 67, 99, 0.55);
}

.cat-whisker-left {
  left: 0;
  transform: rotate(9deg);
}

.cat-whisker-right {
  right: 0;
  transform: rotate(-9deg);
}

.recipe-welcome-copy {
  min-width: 0;
}

.recipe-welcome-kicker,
.card-overline {
  color: #aebddd;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.14em;
  line-height: 1.3;
}

.recipe-welcome-copy h2 {
  margin: 8px 0 6px;
  font-size: 24px;
  letter-spacing: -0.04em;
}

.recipe-welcome-copy p {
  max-width: 620px;
  margin: 0;
  color: #d5def0;
  font-size: 13px;
  line-height: 1.65;
}

.today-summary-box {
  display: flex;
  min-width: 112px;
  flex-direction: column;
  align-items: center;
  padding: 14px 18px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.13);
  border-radius: 14px;
}

.today-summary-box span,
.today-summary-box small {
  color: #c8d3e8;
  font-size: 11px;
}

.today-summary-box strong {
  margin: 4px 0 2px;
  color: #f8d487;
  font-size: 20px;
}

.recipe-error {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-top: 16px;
  padding: 11px 14px;
  color: #8b5a26;
  font-size: 13px;
  background: #fff8e7;
  border: 1px solid #f2dfb2;
  border-radius: var(--ds-radius-md);
}

.recipe-error svg {
  width: 16px;
  flex: 0 0 auto;
}

.recipe-error .el-button {
  margin-left: auto;
}

.recipe-overview-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(330px, 0.85fr);
  gap: 20px;
  margin-top: 20px;
}

.today-card,
.rhythm-card,
.recipe-table-card {
  padding: 24px;
}

.card-overline {
  margin: 0 0 6px;
  color: var(--ds-color-primary);
}

.card-heading h2 {
  margin: 0;
  color: var(--ds-color-text-primary);
  font-size: 18px;
  letter-spacing: -0.03em;
}

.card-heading p:last-child {
  margin: 4px 0 0;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.today-card .card-heading,
.rhythm-card .card-heading,
.recipe-table-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.today-card-icon {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  color: #a9772e;
  background: #fff4d7;
  border-radius: 13px;
}

.today-card-icon svg {
  width: 21px;
}

.today-food-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 20px;
}

.today-food-item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 58px;
  padding: 10px 12px;
  background: var(--ds-color-surface-muted);
  border: 1px solid #eef0f5;
  border-radius: 11px;
}

.today-food-check {
  display: grid;
  width: 25px;
  height: 25px;
  flex: 0 0 auto;
  place-items: center;
  color: #966f2e;
  background: #ffedbd;
  border-radius: 8px;
}

.today-food-check svg {
  width: 14px;
}

.today-food-item strong,
.today-food-item span {
  display: block;
}

.today-food-item strong {
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.today-food-item div span {
  margin-top: 2px;
  overflow: hidden;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.today-loading,
.today-empty {
  display: flex;
  align-items: center;
  min-height: 58px;
  margin-top: 20px;
  padding: 12px 14px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  background: var(--ds-color-surface-muted);
  border-radius: 11px;
}

.today-empty .el-button {
  margin-left: auto;
}

.rhythm-card-icon {
  width: 21px;
  color: var(--ds-color-primary);
}

.rhythm-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-top: 21px;
}

.rhythm-stats div {
  padding: 12px 10px;
  text-align: center;
  background: var(--ds-color-surface-muted);
  border-radius: 10px;
}

.rhythm-stats strong,
.rhythm-stats span {
  display: block;
}

.rhythm-stats strong {
  color: var(--ds-color-primary);
  font-size: 22px;
  font-variant-numeric: tabular-nums;
}

.rhythm-stats span {
  margin-top: 2px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.rhythm-note {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 17px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.rhythm-note-dot,
.table-hint span {
  width: 7px;
  height: 7px;
  flex: 0 0 auto;
  background: var(--ds-color-accent);
  border-radius: 50%;
}

.recipe-table-card {
  margin-top: 20px;
}

.table-hint {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 7px 10px;
  color: #92702f;
  font-size: 11px;
  white-space: nowrap;
  background: #fff8e7;
  border: 1px solid #f2e3bd;
  border-radius: var(--ds-radius-pill);
}

.recipe-table-scroll {
  margin-top: 24px;
  overflow-x: auto;
  border: 1px solid var(--ds-color-border);
  border-radius: 13px;
}

.recipe-table {
  min-width: 880px;
}

.recipe-table-row {
  display: grid;
  grid-template-columns: minmax(275px, 1.65fr) repeat(7, minmax(75px, 1fr));
}

.recipe-table-header {
  min-height: 75px;
  background: var(--ds-color-surface-muted);
  border-bottom: 1px solid var(--ds-color-border);
}

.food-column-heading,
.recipe-day-heading {
  display: flex;
  min-width: 0;
  flex-direction: column;
  justify-content: center;
  padding: 12px 13px;
}

.food-column-heading {
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  letter-spacing: 0.03em;
}

.recipe-day-heading {
  position: relative;
  align-items: center;
  text-align: center;
  border-left: 1px solid #eef0f5;
}

.recipe-day-heading strong {
  color: var(--ds-color-text-secondary);
  font-size: 12px;
}

.recipe-day-heading small {
  margin-top: 3px;
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
  font-variant-numeric: tabular-nums;
}

.recipe-day-heading em {
  position: absolute;
  top: 7px;
  right: 7px;
  padding: 2px 4px;
  color: #966f2e;
  font-size: 9px;
  font-style: normal;
  line-height: 1;
  background: #ffedbd;
  border-radius: 4px;
}

.recipe-day-heading.is-today {
  background: #fff8e7;
  box-shadow: inset 0 3px 0 var(--ds-color-accent);
}

.recipe-day-heading.is-today strong {
  color: #8d682b;
}

.recipe-table-body-row {
  min-height: 78px;
  background: #ffffff;
  border-bottom: 1px solid #eef0f5;
}

.recipe-table-body-row:last-child {
  border-bottom: 0;
}

.food-cell {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 11px;
  padding: 12px 14px;
}

.food-color-mark {
  width: 7px;
  height: 34px;
  flex: 0 0 auto;
  background: #aebbd4;
  border-radius: 4px;
}

.food-color-mark.is-main,
.food-category.is-main {
  background: #e9f0ff;
  color: #5870a0;
}

.food-color-mark.is-supplement,
.food-category.is-supplement {
  background: #fff0d0;
  color: #a47627;
}

.food-color-mark.is-snack,
.food-category.is-snack {
  background: #fbe8e7;
  color: #b46c70;
}

.food-color-mark.is-care,
.food-category.is-care {
  background: #e6f4ed;
  color: #5c9273;
}

.food-color-mark.is-other,
.food-category.is-other {
  background: #edf0f5;
  color: #7b879d;
}

.food-cell-copy {
  min-width: 0;
  flex: 1;
}

.food-name-line {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 7px;
}

.food-name-line strong {
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.food-category {
  padding: 3px 6px;
  flex: 0 0 auto;
  font-size: 10px;
  border-radius: 5px;
}

.food-cell-copy small {
  display: block;
  margin-top: 4px;
  overflow: hidden;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.food-actions {
  display: flex;
  gap: 2px;
  margin-left: auto;
  opacity: 0.45;
  transition: opacity 160ms ease;
}

.food-cell:hover .food-actions,
.food-actions:focus-within {
  opacity: 1;
}

.food-actions button {
  display: grid;
  width: 29px;
  height: 29px;
  place-items: center;
  color: var(--ds-color-text-tertiary);
  cursor: pointer;
  background: transparent;
  border-radius: 7px;
}

.food-actions button:hover {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.food-actions button:last-child:hover {
  color: var(--ds-color-danger);
  background: #fff1f1;
}

.food-actions button:disabled {
  cursor: wait;
  opacity: 0.5;
}

.food-actions svg {
  width: 14px;
}

.recipe-cell {
  position: relative;
  display: grid;
  min-width: 0;
  min-height: 78px;
  place-items: center;
  cursor: pointer;
  background: #ffffff;
  border-left: 1px solid #eef0f5;
  transition: background-color 160ms ease, opacity 160ms ease;
}

.recipe-cell.is-other-day {
  opacity: 0.72;
}

.recipe-cell.is-today {
  background: rgba(255, 248, 231, 0.68);
  box-shadow: inset 1px 0 0 #f4e0ae, inset -1px 0 0 #f4e0ae;
  opacity: 1;
}

.recipe-cell:hover,
.recipe-cell.is-today:hover {
  background: var(--ds-color-primary-light);
  opacity: 1;
}

.recipe-cell:disabled {
  cursor: wait;
}

.scheduled-mark,
.empty-mark {
  display: grid;
  width: 27px;
  height: 27px;
  place-items: center;
  border-radius: 9px;
  transition: transform 160ms ease, background-color 160ms ease;
}

.scheduled-mark {
  color: #ffffff;
  background: var(--ds-color-primary);
  box-shadow: 0 4px 9px rgba(82, 107, 155, 0.2);
}

.is-today .scheduled-mark {
  color: #765622;
  background: var(--ds-color-accent);
  box-shadow: 0 4px 9px rgba(213, 166, 71, 0.2);
}

.scheduled-mark svg {
  width: 14px;
}

.empty-mark {
  width: 12px;
  height: 12px;
  border: 1px solid #d6dce8;
  border-radius: 50%;
}

.recipe-cell:hover .scheduled-mark,
.recipe-cell:hover .empty-mark {
  transform: scale(1.08);
}

.recipe-table-footer {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 14px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.recipe-table-footer > span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.footer-mark {
  width: 8px;
  height: 8px;
  border-radius: 3px;
}

.footer-mark.is-scheduled {
  background: var(--ds-color-primary);
}

.footer-mark.is-today {
  background: var(--ds-color-accent);
}

.recipe-table-footer-copy {
  margin-left: auto;
}

.recipe-table-loading {
  display: grid;
  gap: 12px;
  margin-top: 24px;
  padding: 18px;
  border: 1px solid var(--ds-color-border);
  border-radius: 13px;
}

.loading-line {
  height: 48px;
  background: linear-gradient(90deg, #f2f4f8 20%, #fafbfc 45%, #f2f4f8 70%);
  background-size: 200% 100%;
  border-radius: 9px;
  animation: recipe-loading 1.4s ease-in-out infinite;
}

.loading-line-wide {
  height: 36px;
}

@keyframes recipe-loading {
  from { background-position: 100% 0; }
  to { background-position: -100% 0; }
}

.recipe-empty-state {
  display: flex;
  min-height: 250px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 28px;
  text-align: center;
}

.empty-apple {
  display: grid;
  width: 50px;
  height: 50px;
  margin-bottom: 13px;
  place-items: center;
  color: #a9772e;
  background: #fff4d7;
  border-radius: 16px;
}

.empty-apple svg {
  width: 24px;
}

.recipe-empty-state strong {
  font-size: 15px;
}

.recipe-empty-state p {
  margin: 6px 0 16px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.recipe-note-strip {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 20px;
  padding: 17px 20px;
  background: #f8f5ed;
  border: 1px solid #eee4cb;
  border-radius: var(--ds-radius-lg);
}

.recipe-note-icon {
  display: grid;
  width: 35px;
  height: 35px;
  flex: 0 0 auto;
  place-items: center;
  color: #a87a2d;
  background: #fff4d7;
  border-radius: 10px;
}

.recipe-note-icon svg {
  width: 17px;
}

.recipe-note-strip strong,
.recipe-note-strip p {
  display: block;
}

.recipe-note-strip strong {
  font-size: 13px;
}

.recipe-note-strip p {
  margin: 3px 0 0;
  color: #8b7956;
  font-size: 12px;
}

.recipe-note-strip > span {
  margin-left: auto;
  color: #a87a2d;
  font-size: 11px;
}

.recipe-dialog-intro {
  margin: -4px 0 20px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.recipe-form-control {
  width: 100%;
}

.form-day-picker {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 6px;
  width: 100%;
}

.form-day-button {
  display: flex;
  min-height: 55px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 7px 4px;
  color: var(--ds-color-text-secondary);
  cursor: pointer;
  background: var(--ds-color-surface-muted);
  border: 1px solid var(--ds-color-border);
  border-radius: 9px;
  transition: color 160ms ease, background-color 160ms ease, border-color 160ms ease;
}

.form-day-button strong {
  font-size: 12px;
}

.form-day-button small {
  margin-top: 3px;
  color: var(--ds-color-text-tertiary);
  font-size: 9px;
}

.form-day-button.is-selected {
  color: #7f5f27;
  background: #fff4d7;
  border-color: #eacb83;
}

.form-day-button.is-selected small {
  color: #a98238;
}

.form-help {
  margin-top: 8px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

@media (max-width: 1100px) {
  .recipe-welcome {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .today-summary-box {
    grid-column: 2;
    justify-self: start;
    flex-direction: row;
    gap: 8px;
    min-width: 0;
    padding: 7px 10px;
  }

  .today-summary-box strong {
    margin: 0;
  }

  .today-summary-box small {
    margin-left: 2px;
  }
}

@media (max-width: 980px) {
  .recipe-overview-grid {
    grid-template-columns: 1fr;
  }

  .recipe-table-footer-copy {
    display: none !important;
  }
}

@media (max-width: 700px) {
  .recipe-welcome {
    grid-template-columns: 1fr;
    justify-items: start;
    padding: 24px;
  }

  .today-summary-box {
    grid-column: auto;
  }

  .today-food-list {
    grid-template-columns: 1fr;
  }

  .recipe-table-heading {
    display: block;
  }

  .table-hint {
    margin-top: 12px;
  }

  .recipe-note-strip {
    align-items: flex-start;
  }

  .recipe-note-strip > span {
    display: none;
  }
}
</style>
