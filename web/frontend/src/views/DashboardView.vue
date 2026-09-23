<script setup>
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import {
  ArrowRight,
  Calendar,
  Check,
  CircleCheck,
  Clock,
  DataAnalysis,
  Timer,
} from '@element-plus/icons-vue'
import { getHealth } from '../api'

const router = useRouter()
const health = ref(null)
const healthLoading = ref(true)
const healthError = ref('')

const todayLabel = computed(() => {
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'long',
    day: 'numeric',
    weekday: 'long',
  }).format(new Date())
})

const stats = [
  { label: '今日聚焦', value: '3', note: '1 项即将到期', icon: Timer, tone: 'blue' },
  { label: '待完成任务', value: '8', note: '比昨天少 2 项', icon: Check, tone: 'green' },
  { label: '本周日程', value: '5', note: '最近是周六聚会', icon: Calendar, tone: 'warm' },
]

const agenda = [
  { time: '09:30', title: '整理年度规划', type: '任务', tone: 'blue' },
  { time: '14:00', title: '专注阅读 45 分钟', type: '个人', tone: 'warm' },
  { time: '18:00', title: '和自己确认计划', type: '任务', tone: 'green' },
]

const focusItems = [
  { title: '确定下半年的三个关键词', due: '今天 16:00', priority: '高优先级', tone: 'danger' },
  { title: '整理到可执行的月度计划', due: '明天 18:00', priority: '中优先级', tone: 'warning' },
  { title: '准备轻食和饮料清单', due: '周五 12:00', priority: '低优先级', tone: 'muted' },
]

async function refreshHealth() {
  healthLoading.value = true
  healthError.value = ''
  try {
    health.value = await getHealth()
  } catch (error) {
    health.value = null
    healthError.value = error.message
  } finally {
    healthLoading.value = false
  }
}

function goToTasks() {
  router.push('/dashboard/tasks')
}

onMounted(refreshHealth)
</script>

<template>
  <div class="page-view dashboard-page">
    <section class="dashboard-intro">
      <div class="page-heading-copy">
        <p class="eyebrow">{{ todayLabel }}</p>
        <h1>把今天安排得刚刚好。</h1>
        <p>你好，今天也不必一次完成所有事情。先从最重要的那一件开始。</p>
      </div>
      <div class="page-heading-actions">
        <el-button class="soft-button" @click="goToTasks"><Timer class="button-icon" />查看任务</el-button>
        <el-button type="primary" @click="goToTasks"><span class="button-plus">＋</span>新建任务</el-button>
      </div>
    </section>

    <section class="dashboard-hero">
      <div class="hero-copy-block">
        <div class="hero-overline">A QUIET PLACE FOR YOUR DAY</div>
        <h2>让计划成为支持，<br />而不是额外的压力。</h2>
        <p>把想法放在这里，给每一件事一个清晰的位置。</p>
        <RouterLink to="/dashboard/tasks" class="hero-link">从任务开始 <ArrowRight /></RouterLink>
      </div>
      <div class="hero-visual" aria-hidden="true">
        <div class="hero-orbit hero-orbit-back"></div>
        <div class="hero-orbit hero-orbit-front"></div>
        <div class="hero-sun"><CircleCheck /></div>
        <span class="hero-spark hero-spark-one"></span>
        <span class="hero-spark hero-spark-two"></span>
      </div>
    </section>

    <section class="dashboard-stat-grid">
      <div v-for="stat in stats" :key="stat.label" class="dashboard-stat-card">
        <div class="stat-icon" :class="`is-${stat.tone}`"><component :is="stat.icon" /></div>
        <div class="stat-content">
          <div class="stat-label">{{ stat.label }}</div>
          <div class="stat-number">{{ stat.value }}</div>
          <div class="stat-note">{{ stat.note }}</div>
        </div>
        <DataAnalysis class="stat-decoration" />
      </div>
    </section>

    <section class="dashboard-content-grid">
      <div class="surface-card focus-card">
        <div class="card-heading">
          <div>
            <h2>今天先做这些</h2>
            <p>来自任务树中最值得被看见的几件事。</p>
          </div>
          <RouterLink to="/dashboard/tasks" class="muted-link">查看全部 <ArrowRight /></RouterLink>
        </div>
        <div class="focus-list">
          <div v-for="item in focusItems" :key="item.title" class="focus-item">
            <span class="focus-check"></span>
            <div class="focus-item-main">
              <div class="focus-item-title">{{ item.title }}</div>
              <div class="focus-item-meta">
                <span :class="`is-${item.tone}`"><Clock />{{ item.due }}</span>
                <span>{{ item.priority }}</span>
              </div>
            </div>
            <ArrowRight class="focus-item-arrow" />
          </div>
        </div>
      </div>

      <div class="surface-card rhythm-card">
        <div class="card-heading">
          <div>
            <h2>今天的节奏</h2>
            <p>留一点空白，事情会更容易完成。</p>
          </div>
          <span class="status-inline"><span class="status-inline-dot"></span>平稳</span>
        </div>
        <div class="agenda-list">
          <div v-for="item in agenda" :key="item.time" class="agenda-item">
            <div class="agenda-time">{{ item.time }}</div>
            <div class="agenda-line" :class="`is-${item.tone}`"></div>
            <div class="agenda-item-body">
              <div class="agenda-title">{{ item.title }}</div>
              <div class="agenda-type">{{ item.type }}</div>
            </div>
          </div>
        </div>
        <RouterLink to="/dashboard/calendar" class="calendar-link">打开日程 <ArrowRight /></RouterLink>
      </div>
    </section>

    <section class="dashboard-footer-grid">
      <div class="system-status-card surface-card">
        <div class="card-heading">
          <div>
            <h2>本地运行状态</h2>
            <p>数据保存在你自己的设备上。</p>
          </div>
          <el-button text type="primary" :loading="healthLoading" @click="refreshHealth">重新检查</el-button>
        </div>
        <div v-if="health" class="system-status-row">
          <span class="system-status-icon is-online"><CircleCheck /></span>
          <div>
            <strong>后端与数据库运行正常</strong>
            <span>豆包空间服务已连接 · {{ health.version }}</span>
          </div>
        </div>
        <div v-else-if="healthError" class="system-status-row">
          <span class="system-status-icon is-error"><Clock /></span>
          <div>
            <strong>暂时无法连接后端</strong>
            <span>{{ healthError }}</span>
          </div>
        </div>
        <div v-else class="system-status-row is-loading">
          <span class="system-status-icon"><Clock /></span>
          <div><strong>正在检查运行状态</strong></div>
        </div>
      </div>

      <div class="small-quote-card">
        <div class="quote-mark">“</div>
        <p>做少一点，但把重要的事做好。</p>
        <span>— 给今天的自己</span>
      </div>
    </section>
  </div>
</template>

<style scoped>
.dashboard-intro {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 26px;
}

.dashboard-intro h1 {
  margin: 0;
  color: var(--ds-color-text-primary);
  font-size: 31px;
  font-weight: 700;
  letter-spacing: -0.05em;
  line-height: 1.15;
}

.dashboard-intro p:not(.eyebrow) {
  max-width: 540px;
  margin: 10px 0 0;
  color: var(--ds-color-text-tertiary);
  line-height: 1.7;
}

.button-plus {
  margin-right: 4px;
  font-size: 18px;
  line-height: 0;
  vertical-align: -1px;
}

.dashboard-hero {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 264px;
  margin-bottom: 18px;
  padding: 38px 48px;
  overflow: hidden;
  color: #ffffff;
  background: linear-gradient(122deg, #3f557f 0%, #526b9b 56%, #7183ad 100%);
  border-radius: var(--ds-radius-xl);
  box-shadow: var(--ds-shadow-hero);
}

.hero-copy-block {
  position: relative;
  z-index: 2;
}

.hero-overline {
  color: #ccd8f0;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.17em;
}

.hero-copy-block h2 {
  margin: 15px 0 10px;
  font-size: 30px;
  font-weight: 650;
  letter-spacing: -0.045em;
  line-height: 1.28;
}

.hero-copy-block p {
  margin: 0;
  color: #dce5f7;
}

.hero-link {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  margin-top: 26px;
  padding-bottom: 4px;
  color: #fff4d9;
  font-size: 13px;
  font-weight: 700;
  border-bottom: 1px solid rgba(255, 244, 217, 0.55);
}

.hero-link svg {
  width: 15px;
}

.hero-visual {
  position: relative;
  width: 280px;
  height: 220px;
  margin-right: 12px;
}

.hero-orbit {
  position: absolute;
  top: 50%;
  left: 50%;
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: 50%;
  transform: translate(-50%, -50%) rotate(-24deg);
}

.hero-orbit-back {
  width: 252px;
  height: 126px;
}

.hero-orbit-front {
  width: 190px;
  height: 190px;
  transform: translate(-50%, -50%) rotate(30deg);
}

.hero-sun {
  position: absolute;
  top: 50%;
  left: 50%;
  display: grid;
  width: 88px;
  height: 88px;
  place-items: center;
  color: var(--ds-color-brand-700);
  background: var(--ds-color-accent);
  border: 11px solid rgba(255, 255, 255, 0.13);
  border-radius: 50%;
  box-shadow: 0 14px 28px rgba(31, 42, 68, 0.18);
  transform: translate(-50%, -50%);
}

.hero-sun svg {
  width: 30px;
}

.hero-spark {
  position: absolute;
  width: 8px;
  height: 8px;
  background: #ffffff;
  border-radius: 50%;
  opacity: 0.72;
}

.hero-spark-one {
  top: 30px;
  right: 45px;
}

.hero-spark-two {
  right: 18px;
  bottom: 30px;
  width: 5px;
  height: 5px;
  background: var(--ds-color-accent);
}

.dashboard-stat-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 18px;
}

.dashboard-stat-card {
  position: relative;
  display: flex;
  align-items: center;
  min-height: 122px;
  gap: 15px;
  padding: 22px 24px;
  overflow: hidden;
  background: var(--ds-color-surface);
  border: 1px solid var(--ds-color-border);
  border-radius: var(--ds-radius-lg);
}

.stat-icon {
  display: grid;
  width: 40px;
  height: 40px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 12px;
}

.stat-icon svg {
  width: 19px;
}

.stat-icon.is-blue {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.stat-icon.is-green {
  color: var(--ds-color-success);
  background: #edf8e9;
}

.stat-icon.is-warm {
  color: #ae7d2e;
  background: #fff7e6;
}

.stat-content {
  position: relative;
  z-index: 1;
}

.stat-label {
  color: var(--ds-color-text-tertiary);
  font-size: 13px;
}

.stat-number {
  margin-top: 2px;
  color: var(--ds-color-text-primary);
  font-size: 28px;
  font-weight: 700;
  letter-spacing: -0.05em;
  line-height: 1.1;
}

.stat-note {
  margin-top: 4px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.stat-decoration {
  position: absolute;
  right: 18px;
  bottom: 13px;
  width: 58px;
  height: 58px;
  color: #eef1f7;
}

.dashboard-content-grid,
.dashboard-footer-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(360px, 0.85fr);
  gap: 18px;
  margin-bottom: 18px;
}

.focus-card,
.rhythm-card,
.system-status-card {
  padding: 24px;
}

.focus-list {
  margin-top: 22px;
}

.focus-item {
  display: flex;
  align-items: center;
  gap: 13px;
  min-height: 66px;
  border-bottom: 1px solid #eff1f6;
}

.focus-item:last-child {
  border-bottom: 0;
}

.focus-check {
  display: block;
  width: 17px;
  height: 17px;
  flex: 0 0 auto;
  border: 1.5px solid #c3ccdc;
  border-radius: 50%;
}

.focus-item-main {
  min-width: 0;
  flex: 1;
}

.focus-item-title {
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.focus-item-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 6px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.focus-item-meta span:first-child {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.focus-item-meta svg {
  width: 13px;
}

.focus-item-meta .is-danger {
  color: var(--ds-color-danger);
}

.focus-item-meta .is-warning {
  color: var(--ds-color-warning);
}

.focus-item-meta .is-muted {
  color: var(--ds-color-text-tertiary);
}

.focus-item-arrow {
  width: 15px;
  color: #c6cedc;
}

.rhythm-card {
  display: flex;
  flex-direction: column;
}

.agenda-list {
  margin-top: 20px;
}

.agenda-item {
  display: grid;
  grid-template-columns: 48px 4px minmax(0, 1fr);
  align-items: stretch;
  min-height: 58px;
  gap: 13px;
}

.agenda-time {
  padding-top: 3px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.agenda-line {
  width: 4px;
  margin: 2px 0;
  background: var(--ds-color-primary);
  border-radius: 4px;
}

.agenda-line.is-warm {
  background: var(--ds-color-accent);
}

.agenda-line.is-green {
  background: var(--ds-color-success);
}

.agenda-item-body {
  padding-bottom: 14px;
}

.agenda-title {
  color: var(--ds-color-text-primary);
  font-size: 13px;
  font-weight: 600;
}

.agenda-type {
  margin-top: 4px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.calendar-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: auto;
  padding-top: 15px;
  color: var(--ds-color-primary);
  font-size: 12px;
  font-weight: 700;
  border-top: 1px solid var(--ds-color-border);
}

.calendar-link svg {
  width: 14px;
}

.dashboard-footer-grid {
  grid-template-columns: minmax(0, 1.15fr) minmax(280px, 0.85fr);
  margin-bottom: 0;
}

.system-status-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 22px;
}

.system-status-icon {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  color: var(--ds-color-success);
  background: #edf8e9;
  border-radius: 11px;
}

.system-status-icon svg {
  width: 19px;
}

.system-status-icon.is-error {
  color: var(--ds-color-danger);
  background: #fff0f0;
}

.system-status-row strong,
.system-status-row span:not(.system-status-icon) {
  display: block;
}

.system-status-row strong {
  font-size: 13px;
}

.system-status-row > div > span {
  margin-top: 3px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.small-quote-card {
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-height: 146px;
  padding: 24px 28px;
  overflow: hidden;
  color: #ffffff;
  background: var(--ds-color-brand-900);
  border-radius: var(--ds-radius-lg);
}

.small-quote-card::after {
  position: absolute;
  right: -30px;
  bottom: -60px;
  width: 170px;
  height: 170px;
  content: '';
  border: 1px solid rgba(255, 255, 255, 0.11);
  border-radius: 50%;
}

.quote-mark {
  position: absolute;
  top: 7px;
  right: 27px;
  color: rgba(247, 200, 115, 0.5);
  font-family: Georgia, serif;
  font-size: 72px;
  line-height: 1;
}

.small-quote-card p {
  position: relative;
  z-index: 1;
  max-width: 280px;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  line-height: 1.6;
}

.small-quote-card span {
  position: relative;
  z-index: 1;
  margin-top: 10px;
  color: #aebbd8;
  font-size: 12px;
}

@media (max-width: 1120px) {
  .hero-visual {
    width: 230px;
    transform: scale(0.86);
  }

  .dashboard-content-grid,
  .dashboard-footer-grid {
    grid-template-columns: minmax(0, 1fr) minmax(300px, 0.8fr);
  }
}
</style>
