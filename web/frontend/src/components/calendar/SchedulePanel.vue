<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Calendar,
  Check,
  Clock,
  Delete,
  Edit,
  Plus,
  Refresh,
  Timer,
  WarningFilled,
} from '@element-plus/icons-vue'
import {
  completeSchedule,
  createSchedule,
  deleteSchedule,
  getSchedules,
  updateSchedule,
  completePlan,
  createPlan,
  deletePlan,
  getPlans,
  updatePlan,
} from '../../api'

const props = defineProps({
  variant: {
    type: String,
    default: 'doubao',
  },
})

const isPersonal = computed(() => props.variant === 'personal')
const copy = computed(() => {
  if (isPersonal.value) {
    return {
      eyebrow: 'PERSONAL PLAN',
      heading: '个人计划',
      description: '把阅读、运动、学习等需要周期打卡的事情交给计划记住。',
      kind: '计划',
      refresh: '刷新计划',
      add: '添加计划',
      errorPrefix: '计划暂时无法读取',
      cycleLabel: '周期计划',
      emptyAdd: '添加第一个周期计划',
      itemPlaceholder: '例如：阅读、运动、学习英语',
      notePlaceholder: '例如：每次完成后记下进度，给自己一点反馈',
      example: '例如从今天开始、每 7 天运动一次，完成后再从完成日重新计算。',
      addedSuccess: '计划已加入个人计划。',
      dueNote: '今天完成这件事，让自己离目标更近一点。',
      emptyDueTitle: '今天没有需要处理的计划',
      emptyDueDescription: '可以放心做自己的事，下一次提醒会按计划出现。',
      upcomingEmpty: '当前计划都在今天的提醒里。',
      ruleTitle: '计划规则很简单',
      ruleDescription: '逾期不会被跳过；完成当天会成为新的起点，下一次提醒从这一天往后计算间隔天数。',
      dialogIntro: '设置一件需要周期完成的事，计划会自动算出下一次提醒。',
    }
  }
  return {
    eyebrow: 'DOUBAO SCHEDULE',
    heading: '豆包日程',
    description: '把剪指甲、驱虫、清理猫砂盆这类周期事情交给日历记住。',
    kind: '日程',
    refresh: '刷新日程',
    add: '添加日程',
    errorPrefix: '日程暂时无法读取',
    cycleLabel: '周期日程',
    emptyAdd: '添加第一个周期日程',
    itemPlaceholder: '例如：剪指甲、驱虫、清理猫砂盆',
    notePlaceholder: '例如：洗完澡后剪，或记得准备零食安抚豆包',
    example: '例如从今天开始、每 30 天剪一次指甲，完成后再从完成日重新计算。',
    addedSuccess: '日程已加入豆包。',
    dueNote: '今天完成这件事，豆包会更舒服。',
    emptyDueTitle: '今天没有需要处理的日程',
    emptyDueDescription: '可以放心做自己的事，下一次提醒会按计划出现。',
    upcomingEmpty: '当前日程都在今天的提醒里。',
    ruleTitle: '提醒规则很简单',
    ruleDescription: '逾期不会被跳过；完成当天会成为新的起点，下一次提醒从这一天往后计算间隔天数。',
    dialogIntro: '设置一件豆包需要定期完成的事，日历会自动算出下一次提醒。',
  }
})

const plan = ref({ today: '', today_label: '', schedules: [], due_count: 0, upcoming_count: 0 })
const loading = ref(true)
const loadError = ref('')
const dialogVisible = ref(false)
const editingId = ref(null)
const saving = ref(false)
const completingId = ref(null)
const deletingId = ref(null)
const form = ref(createEmptyForm())

const schedules = computed(() => plan.value.schedules || [])
const dueSchedules = computed(() => schedules.value.filter((schedule) => schedule.is_due))
const upcomingSchedules = computed(() => schedules.value.filter((schedule) => !schedule.is_due))
const nextSchedule = computed(() => upcomingSchedules.value[0] || null)
const dialogTitle = computed(() => (editingId.value ? `编辑${copy.value.kind}` : `添加${copy.value.kind}`))
const todayLabel = computed(() => plan.value.today_label || '今天')

function getPlanData() {
  return isPersonal.value ? getPlans() : getSchedules()
}

function updatePlanData(id, payload) {
  return isPersonal.value ? updatePlan(id, payload) : updateSchedule(id, payload)
}

function createPlanData(payload) {
  return isPersonal.value ? createPlan(payload) : createSchedule(payload)
}

function completePlanData(id) {
  return isPersonal.value ? completePlan(id) : completeSchedule(id)
}

function deletePlanData(id) {
  return isPersonal.value ? deletePlan(id) : deleteSchedule(id)
}

function createEmptyForm() {
  return {
    title: '',
    note: '',
    start_date: localDateValue(),
    interval_days: 30,
  }
}

function localDateValue(value = new Date()) {
  const date = new Date(value)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function formatDateLabel(value) {
  if (!value) return '未设置'
  const parts = value.split('-')
  return parts.length === 3 ? `${Number(parts[1])}月${Number(parts[2])}日` : value
}

function intervalLabel(days) {
  return `每 ${days} 天一次`
}

function dueLabel(schedule) {
  if (schedule.overdue_days > 0) return `已逾期 ${schedule.overdue_days} 天，今天仍提醒`
  return '今天到期'
}

function dueTone(schedule) {
  return schedule.overdue_days > 0 ? 'is-overdue' : 'is-due-today'
}

async function loadSchedules() {
  loading.value = true
  loadError.value = ''
  try {
    plan.value = await getPlanData()
  } catch (error) {
    plan.value = { today: '', today_label: '', schedules: [], due_count: 0, upcoming_count: 0 }
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

function openEdit(schedule) {
  editingId.value = schedule.id
  form.value = {
    title: schedule.title,
    note: schedule.note || '',
    start_date: schedule.start_date,
    interval_days: schedule.interval_days,
  }
  dialogVisible.value = true
}

async function saveSchedule() {
  if (!form.value.title.trim()) {
    ElMessage.warning(`请先填写${copy.value.kind}名称。`)
    return
  }
  if (!form.value.start_date) {
    ElMessage.warning('请选择开始日期。')
    return
  }
  if (!form.value.interval_days || form.value.interval_days < 1) {
    ElMessage.warning('间隔天数至少为 1 天。')
    return
  }

  saving.value = true
  const wasEditing = Boolean(editingId.value)
  try {
    const payload = {
      title: form.value.title.trim(),
      note: form.value.note.trim(),
      start_date: form.value.start_date,
      interval_days: Number(form.value.interval_days),
    }
    if (wasEditing) await updatePlanData(editingId.value, payload)
    else await createPlanData(payload)
    await loadSchedules()
    dialogVisible.value = false
    ElMessage.success(wasEditing ? `${copy.value.kind}已更新。` : copy.value.addedSuccess)
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    saving.value = false
  }
}

async function complete(schedule) {
  completingId.value = schedule.id
  try {
    const updated = await completePlanData(schedule.id)
    await loadSchedules()
    ElMessage.success(`“${schedule.title}”已完成，下次提醒是 ${formatDateLabel(updated.next_due_date)}。`)
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    completingId.value = null
  }
}

async function removeSchedule(schedule) {
  try {
    await ElMessageBox.confirm(
      `删除“${schedule.title}”后，它的周期提醒和完成记录都会移除。`,
      `删除${copy.value.kind}`,
      { confirmButtonText: '删除', cancelButtonText: '保留', type: 'warning' },
    )
  } catch {
    return
  }

  deletingId.value = schedule.id
  try {
    await deletePlanData(schedule.id)
    await loadSchedules()
    ElMessage.success(`${copy.value.kind}已删除。`)
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    deletingId.value = null
  }
}

onMounted(loadSchedules)
</script>

<template>
  <section :id="isPersonal ? 'personal-plans' : 'doubao-schedule'" class="schedule-section" :class="{ 'is-personal': isPersonal }">
    <div class="schedule-section-heading">
      <div>
        <p class="schedule-eyebrow">{{ copy.eyebrow }}</p>
        <h2>{{ copy.heading }}</h2>
        <p>{{ copy.description }}</p>
      </div>
      <div class="schedule-heading-actions">
        <el-button class="soft-button" :loading="loading" @click="loadSchedules">
          <Refresh class="button-icon" />{{ copy.refresh }}
        </el-button>
        <el-button type="primary" @click="openCreate">
          <Plus class="button-icon" />{{ copy.add }}
        </el-button>
      </div>
    </div>

    <div v-if="loadError" class="schedule-error" role="alert">
      <WarningFilled />
      <span>{{ copy.errorPrefix }}：{{ loadError }}</span>
      <el-button text type="primary" @click="loadSchedules">重新加载</el-button>
    </div>

    <div class="schedule-summary-grid">
      <div class="surface-card schedule-summary-card is-alert">
        <div class="schedule-summary-icon"><Timer /></div>
        <div>
          <span>今天要做</span>
          <strong>{{ plan.due_count }}</strong>
          <small>{{ plan.due_count ? '逾期也会继续提醒' : `今天没有待处理${copy.kind}` }}</small>
        </div>
      </div>
      <div class="surface-card schedule-summary-card is-total">
        <div class="schedule-summary-icon"><Calendar /></div>
        <div>
          <span>{{ copy.cycleLabel }}</span>
          <strong>{{ schedules.length }}</strong>
          <small>{{ todayLabel }} · 本地保存</small>
        </div>
      </div>
      <div class="surface-card schedule-summary-card is-next">
        <div class="schedule-summary-icon"><Clock /></div>
        <div>
          <span>下一个提醒</span>
          <strong>{{ nextSchedule ? formatDateLabel(nextSchedule.next_due_date) : '—' }}</strong>
          <small>{{ nextSchedule ? nextSchedule.title : copy.emptyAdd }}</small>
        </div>
      </div>
    </div>

    <div class="schedule-board">
      <div class="surface-card schedule-due-card">
        <div class="schedule-card-heading">
          <div>
            <p class="schedule-card-overline">TODAY</p>
            <h3>今天要做</h3>
            <p>到期后每天都会留在这里，直到你点击完成。</p>
          </div>
          <span class="schedule-status-pill" :class="{ 'is-clear': dueSchedules.length === 0 }">
            {{ dueSchedules.length ? `${dueSchedules.length} 项提醒` : '今天清空' }}
          </span>
        </div>

        <div v-if="loading" class="schedule-loading-list">
          <div v-for="line in 2" :key="line" class="schedule-loading-item"></div>
        </div>
        <div v-else-if="dueSchedules.length" class="schedule-due-list">
          <article v-for="schedule in dueSchedules" :key="schedule.id" class="schedule-due-item" :class="dueTone(schedule)">
            <div class="due-item-mark"><Timer /></div>
            <div class="due-item-main">
              <div class="due-item-title-line">
                <strong>{{ schedule.title }}</strong>
                <span class="due-item-label">{{ dueLabel(schedule) }}</span>
              </div>
              <p>{{ schedule.note || copy.dueNote }}</p>
              <div class="due-item-meta">
                <span>原提醒日 {{ formatDateLabel(schedule.next_due_date) }}</span>
                <span>{{ intervalLabel(schedule.interval_days) }}</span>
              </div>
            </div>
            <div class="due-item-actions">
              <button type="button" :title="`编辑${copy.kind}`" :aria-label="`编辑${copy.kind}`" @click="openEdit(schedule)"><Edit /></button>
              <button
                type="button"
                :title="`删除${copy.kind}`"
                :aria-label="`删除${copy.kind}`"
                :disabled="deletingId === schedule.id"
                @click="removeSchedule(schedule)"
              ><Delete /></button>
              <el-button type="primary" :loading="completingId === schedule.id" @click="complete(schedule)">
                <Check class="button-icon" />完成
              </el-button>
            </div>
          </article>
        </div>
        <div v-else class="schedule-empty-state">
          <div class="schedule-empty-icon"><Check /></div>
          <strong>{{ schedules.length ? copy.emptyDueTitle : `还没有周期${copy.kind}` }}</strong>
          <p>{{ schedules.length ? copy.emptyDueDescription : `先添加一件需要定期完成的${copy.kind}。` }}</p>
          <el-button v-if="!schedules.length" text type="primary" @click="openCreate">{{ copy.emptyAdd }}</el-button>
        </div>
      </div>

      <div class="surface-card schedule-upcoming-card">
        <div class="schedule-card-heading">
          <div>
            <p class="schedule-card-overline">UP NEXT</p>
            <h3>接下来</h3>
            <p>还没到提醒日的周期安排。</p>
          </div>
          <span class="upcoming-count">{{ upcomingSchedules.length }} 项</span>
        </div>

        <div v-if="upcomingSchedules.length" class="schedule-upcoming-list">
          <article v-for="schedule in upcomingSchedules" :key="schedule.id" class="schedule-upcoming-item">
            <div class="upcoming-date-box">
              <strong>{{ formatDateLabel(schedule.next_due_date) }}</strong>
              <span>{{ schedule.days_until_due }} 天后</span>
            </div>
            <div class="upcoming-item-main">
              <strong>{{ schedule.title }}</strong>
              <span>{{ intervalLabel(schedule.interval_days) }}<template v-if="schedule.last_completed_on"> · 上次 {{ formatDateLabel(schedule.last_completed_on) }} 完成</template></span>
            </div>
            <div class="upcoming-item-actions">
              <button type="button" :title="`编辑${copy.kind}`" :aria-label="`编辑${copy.kind}`" @click="openEdit(schedule)"><Edit /></button>
              <button
                type="button"
                :title="`删除${copy.kind}`"
                :aria-label="`删除${copy.kind}`"
                :disabled="deletingId === schedule.id"
                @click="removeSchedule(schedule)"
              ><Delete /></button>
            </div>
          </article>
        </div>
        <div v-else class="upcoming-empty">
          <span>{{ schedules.length ? copy.upcomingEmpty : `添加周期${copy.kind}后，下一次日期会显示在这里。` }}</span>
        </div>
      </div>
    </div>

    <div class="schedule-rule-strip">
      <div class="schedule-rule-icon"><Clock /></div>
      <div>
        <strong>{{ copy.ruleTitle }}</strong>
        <p>{{ copy.ruleDescription }}</p>
      </div>
    </div>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="520px" destroy-on-close>
      <div class="schedule-dialog-intro">{{ copy.dialogIntro }}</div>
      <el-form class="schedule-form" label-position="top" @submit.prevent="saveSchedule">
        <el-form-item :label="`${copy.kind}名称`" required>
          <el-input v-model="form.title" maxlength="100" show-word-limit :placeholder="copy.itemPlaceholder" />
        </el-form-item>
        <div class="schedule-form-row">
          <el-form-item label="从哪一天开始" required>
            <el-date-picker
              v-model="form.start_date"
              type="date"
              value-format="YYYY-MM-DD"
              placeholder="选择开始日期"
              class="schedule-form-control"
            />
          </el-form-item>
          <el-form-item label="每隔几天" required>
            <el-input-number v-model="form.interval_days" :min="1" :max="3650" controls-position="right" class="schedule-form-control" />
          </el-form-item>
        </div>
        <div class="schedule-form-example">
          <Calendar />
          <span>{{ copy.example }}</span>
        </div>
        <el-form-item label="备注">
          <el-input v-model="form.note" type="textarea" :rows="3" maxlength="300" show-word-limit :placeholder="copy.notePlaceholder" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveSchedule">保存{{ copy.kind }}</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<style scoped>
.schedule-section {
  margin-top: 34px;
}

.schedule-section-heading,
.schedule-card-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
}

.schedule-section-heading {
  margin-bottom: 18px;
}

.schedule-eyebrow,
.schedule-card-overline {
  margin: 0 0 6px;
  color: var(--ds-color-primary);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.14em;
  line-height: 1.3;
}

.schedule-section-heading h2 {
  margin: 0;
  color: var(--ds-color-text-primary);
  font-size: 22px;
  letter-spacing: -0.04em;
}

.schedule-section-heading p:last-child,
.schedule-card-heading p:last-child {
  margin: 5px 0 0;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  line-height: 1.6;
}

.schedule-heading-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 0 0 auto;
}

.schedule-error {
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

.schedule-error > svg {
  width: 16px;
  flex: 0 0 auto;
}

.schedule-error .el-button {
  margin-left: auto;
}

.schedule-summary-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

.schedule-summary-card {
  display: flex;
  align-items: center;
  gap: 13px;
  min-height: 104px;
  padding: 18px;
}

.schedule-summary-icon {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 12px;
}

.schedule-summary-icon svg {
  width: 19px;
}

.schedule-summary-card.is-alert .schedule-summary-icon {
  color: #a4772e;
  background: #fff1ca;
}

.schedule-summary-card.is-total .schedule-summary-icon {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.schedule-summary-card.is-next .schedule-summary-icon {
  color: #5d9076;
  background: #e8f4ed;
}

.schedule-summary-card span,
.schedule-summary-card strong,
.schedule-summary-card small {
  display: block;
}

.schedule-summary-card span {
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.schedule-summary-card strong {
  margin: 2px 0 1px;
  color: var(--ds-color-text-primary);
  font-size: 23px;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.03em;
}

.schedule-summary-card small {
  overflow: hidden;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.schedule-board {
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(360px, 0.85fr);
  gap: 16px;
  margin-top: 16px;
}

.schedule-due-card,
.schedule-upcoming-card {
  min-width: 0;
  padding: 22px;
}

.schedule-card-heading h3 {
  margin: 0;
  color: var(--ds-color-text-primary);
  font-size: 18px;
  letter-spacing: -0.03em;
}

.schedule-card-overline {
  margin-bottom: 5px;
}

.schedule-status-pill,
.upcoming-count {
  padding: 6px 9px;
  color: #9a7029;
  font-size: 11px;
  white-space: nowrap;
  background: #fff5d9;
  border: 1px solid #f2dfb2;
  border-radius: var(--ds-radius-pill);
}

.schedule-status-pill.is-clear {
  color: #56876c;
  background: #eaf6ee;
  border-color: #d7ebde;
}

.upcoming-count {
  color: var(--ds-color-text-tertiary);
  background: var(--ds-color-surface-muted);
  border-color: var(--ds-color-border);
}

.schedule-due-list,
.schedule-upcoming-list {
  display: grid;
  gap: 10px;
  margin-top: 20px;
}

.schedule-due-item {
  display: grid;
  grid-template-columns: 32px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 14px;
  background: #fffaf0;
  border: 1px solid #f3e5c3;
  border-left: 3px solid var(--ds-color-accent);
  border-radius: 12px;
}

.schedule-due-item.is-overdue {
  background: #fff7f4;
  border-color: #f4dfd7;
  border-left-color: var(--ds-color-danger);
}

.due-item-mark {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  color: #9a7029;
  background: #ffedbd;
  border-radius: 10px;
}

.is-overdue .due-item-mark {
  color: #b56569;
  background: #fbe3e1;
}

.due-item-mark svg {
  width: 17px;
}

.due-item-main {
  min-width: 0;
}

.due-item-title-line {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 8px;
}

.due-item-title-line strong {
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.due-item-label {
  padding: 3px 6px;
  flex: 0 0 auto;
  color: #9a7029;
  font-size: 10px;
  background: #fff0cb;
  border-radius: 5px;
}

.is-overdue .due-item-label {
  color: #b56569;
  background: #fbe5e2;
}

.due-item-main p {
  overflow: hidden;
  margin: 5px 0 7px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.due-item-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
}

.due-item-actions,
.upcoming-item-actions {
  display: flex;
  align-items: center;
  gap: 3px;
  flex: 0 0 auto;
}

.due-item-actions > button,
.upcoming-item-actions button {
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  color: var(--ds-color-text-tertiary);
  cursor: pointer;
  background: transparent;
  border-radius: 7px;
}

.due-item-actions > button:hover,
.upcoming-item-actions button:hover {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.due-item-actions > button:nth-child(2):hover,
.upcoming-item-actions button:last-child:hover {
  color: var(--ds-color-danger);
  background: #fff1f1;
}

.due-item-actions > button:disabled,
.upcoming-item-actions button:disabled {
  cursor: wait;
  opacity: 0.5;
}

.due-item-actions > button svg,
.upcoming-item-actions button svg {
  width: 14px;
}

.due-item-actions .el-button {
  min-width: 76px;
  margin-left: 4px;
}

.schedule-upcoming-item {
  display: grid;
  grid-template-columns: 78px minmax(0, 1fr) auto;
  align-items: center;
  gap: 11px;
  min-height: 66px;
  padding: 10px 0;
  border-bottom: 1px solid #eef0f5;
}

.schedule-upcoming-item:last-child {
  border-bottom: 0;
}

.upcoming-date-box {
  display: flex;
  min-height: 43px;
  flex-direction: column;
  justify-content: center;
  padding: 6px 9px;
  background: var(--ds-color-primary-light);
  border-radius: 9px;
}

.upcoming-date-box strong,
.upcoming-date-box span {
  display: block;
}

.upcoming-date-box strong {
  color: var(--ds-color-primary);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.upcoming-date-box span {
  margin-top: 2px;
  color: var(--ds-color-text-tertiary);
  font-size: 10px;
}

.upcoming-item-main {
  min-width: 0;
}

.upcoming-item-main strong,
.upcoming-item-main span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.upcoming-item-main strong {
  color: var(--ds-color-text-primary);
  font-size: 13px;
}

.upcoming-item-main span {
  margin-top: 4px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.schedule-empty-state,
.upcoming-empty {
  display: flex;
  min-height: 176px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 22px;
  text-align: center;
}

.schedule-empty-icon {
  display: grid;
  width: 42px;
  height: 42px;
  margin-bottom: 10px;
  place-items: center;
  color: #5d9275;
  background: #e8f4ed;
  border-radius: 13px;
}

.schedule-empty-icon svg {
  width: 20px;
}

.schedule-empty-state strong {
  font-size: 13px;
}

.schedule-empty-state p {
  max-width: 280px;
  margin: 5px 0 4px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  line-height: 1.6;
}

.upcoming-empty {
  min-height: 170px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.schedule-rule-strip {
  display: flex;
  align-items: center;
  gap: 13px;
  margin-top: 16px;
  padding: 14px 17px;
  background: #f8f5ed;
  border: 1px solid #eee4cb;
  border-radius: var(--ds-radius-lg);
}

.schedule-rule-icon {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 auto;
  place-items: center;
  color: #a87a2d;
  background: #fff4d7;
  border-radius: 9px;
}

.schedule-rule-icon svg {
  width: 16px;
}

.schedule-rule-strip strong,
.schedule-rule-strip p {
  display: block;
}

.schedule-rule-strip strong {
  font-size: 12px;
}

.schedule-rule-strip p {
  margin: 3px 0 0;
  color: #8b7956;
  font-size: 11px;
}

.schedule-loading-list {
  display: grid;
  gap: 10px;
  margin-top: 20px;
}

.schedule-loading-item {
  height: 86px;
  background: linear-gradient(90deg, #f2f4f8 20%, #fafbfc 45%, #f2f4f8 70%);
  background-size: 200% 100%;
  border-radius: 12px;
  animation: schedule-loading 1.4s ease-in-out infinite;
}

@keyframes schedule-loading {
  from { background-position: 100% 0; }
  to { background-position: -100% 0; }
}

.schedule-dialog-intro {
  margin: -4px 0 20px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.schedule-form-row {
  display: grid;
  grid-template-columns: 1.1fr 0.9fr;
  gap: 14px;
}

.schedule-form-control {
  width: 100%;
}

.schedule-form-example {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: -2px 0 18px;
  padding: 10px 12px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  line-height: 1.55;
  background: var(--ds-color-surface-muted);
  border-radius: 8px;
}

.schedule-form-example svg {
  width: 14px;
  flex: 0 0 auto;
  margin-top: 2px;
  color: var(--ds-color-primary);
}

@media (max-width: 1100px) {
  .schedule-board {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .schedule-section-heading {
    display: block;
  }

  .schedule-heading-actions {
    margin-top: 14px;
  }

  .schedule-summary-grid {
    grid-template-columns: 1fr;
  }

  .schedule-summary-card {
    min-height: 86px;
  }

  .schedule-due-item {
    grid-template-columns: 32px minmax(0, 1fr);
  }

  .due-item-actions {
    grid-column: 2;
    justify-content: flex-end;
  }

  .schedule-upcoming-item {
    grid-template-columns: 78px minmax(0, 1fr);
  }

  .upcoming-item-actions {
    grid-column: 2;
    justify-content: flex-end;
  }

  .schedule-form-row {
    grid-template-columns: 1fr;
    gap: 0;
  }
}
</style>
