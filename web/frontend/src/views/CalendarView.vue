<script setup>
import {
  ArrowLeft,
  ArrowRight,
  Calendar,
  Clock,
  MoreFilled,
  Plus,
} from '@element-plus/icons-vue'

const weekdays = ['一', '二', '三', '四', '五', '六', '日']
const days = [
  { date: 31, muted: true },
  { date: 1 },
  { date: 2 },
  { date: 3, events: 1 },
  { date: 4 },
  { date: 5 },
  { date: 6 },
  { date: 7 },
  { date: 8 },
  { date: 9, events: 1 },
  { date: 10 },
  { date: 11 },
  { date: 12 },
  { date: 13 },
  { date: 14 },
  { date: 15 },
  { date: 16 },
  { date: 17 },
  { date: 18, events: 2 },
  { date: 19 },
  { date: 20 },
  { date: 21 },
  { date: 22 },
  { date: 23 },
  { date: 24, active: true, events: 2 },
  { date: 25 },
  { date: 26 },
  { date: 27 },
  { date: 28 },
  { date: 29 },
  { date: 30 },
  { date: 1, muted: true },
  { date: 2, muted: true },
  { date: 3, muted: true },
  { date: 4, muted: true },
]

const agenda = [
  { time: '09:30', title: '整理年度规划', source: '任务截止', tone: 'blue' },
  { time: '14:00', title: '专注阅读 45 分钟', source: '个人安排', tone: 'warm' },
  { time: '18:00', title: '和自己确认计划', source: '任务截止', tone: 'danger' },
]
</script>

<template>
  <div class="page-view page-view-wide calendar-page">
    <section class="page-heading">
      <div class="page-heading-copy">
        <p class="eyebrow">TIME SPACE</p>
        <h1>日程管理</h1>
        <p>让时间有安排，也留有余地。今天是值得被好好使用的一天。</p>
      </div>
      <div class="page-heading-actions">
        <el-button class="soft-button"><Calendar class="button-icon" />回到今天</el-button>
        <el-button type="primary"><Plus class="button-icon" />新建安排</el-button>
      </div>
    </section>

    <section class="calendar-layout">
      <div class="surface-card month-card">
        <div class="month-card-header">
          <div>
            <div class="month-title">2026 年 9 月</div>
            <div class="month-subtitle">星期四 · 24 日</div>
          </div>
          <div class="month-controls">
            <button class="calendar-icon-button" type="button" aria-label="上个月"><ArrowLeft /></button>
            <button class="calendar-icon-button" type="button" aria-label="下个月"><ArrowRight /></button>
          </div>
        </div>

        <div class="calendar-weekdays">
          <span v-for="weekday in weekdays" :key="weekday">{{ weekday }}</span>
        </div>
        <div class="calendar-grid">
          <div
            v-for="day in days"
            :key="`${day.date}-${day.muted ? 'muted' : 'current'}-${day.active ? 'active' : ''}`"
            class="calendar-day"
            :class="{ 'is-muted': day.muted, 'is-active': day.active }"
          >
            <span class="day-number">{{ day.date }}</span>
            <span v-if="day.events" class="day-events">
              <i v-for="event in day.events" :key="event"></i>
            </span>
          </div>
        </div>

        <div class="calendar-legend">
          <span><i class="legend-dot is-blue"></i>任务</span>
          <span><i class="legend-dot is-warm"></i>个人安排</span>
        </div>
      </div>

      <div class="surface-card agenda-card">
        <div class="card-heading">
          <div>
            <h2>今天的安排</h2>
            <p>9 月 24 日 · 3 项安排</p>
          </div>
          <span class="agenda-day-mark">24</span>
        </div>

        <div class="calendar-agenda-list">
          <div v-for="item in agenda" :key="item.time" class="calendar-agenda-item">
            <div class="calendar-agenda-time">{{ item.time }}</div>
            <div class="calendar-agenda-line" :class="`is-${item.tone}`"></div>
            <div class="calendar-agenda-content">
              <strong>{{ item.title }}</strong>
              <span>{{ item.source }}</span>
            </div>
            <MoreFilled class="calendar-agenda-more" />
          </div>
        </div>

        <div class="calendar-empty-slot">
          <Clock />
          <span>下午还有一段可以自由安排的时间</span>
        </div>
      </div>
    </section>

    <section class="calendar-note-strip">
      <div class="calendar-note-icon"><Calendar /></div>
      <div>
        <strong>任务截止时间会出现在这里</strong>
        <p>日程模块会负责时间呈现和提醒，任务模块继续负责任务本身。</p>
      </div>
      <span class="calendar-preview-label">设计预览</span>
    </section>
  </div>
</template>

<style scoped>
.calendar-page {
  max-width: 1240px;
}

.calendar-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.16fr) minmax(360px, 0.84fr);
  gap: 20px;
}

.month-card,
.agenda-card {
  padding: 26px;
}

.month-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.month-title {
  color: var(--ds-color-text-primary);
  font-size: 19px;
  font-weight: 700;
  letter-spacing: -0.03em;
}

.month-subtitle {
  margin-top: 4px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.month-controls {
  display: flex;
  gap: 5px;
}

.calendar-icon-button {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  color: var(--ds-color-text-tertiary);
  cursor: pointer;
  background: var(--ds-color-surface-muted);
  border-radius: 8px;
}

.calendar-icon-button:hover {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.calendar-icon-button svg {
  width: 15px;
}

.calendar-weekdays,
.calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 7px;
}

.calendar-weekdays {
  margin-top: 28px;
}

.calendar-weekdays span {
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  text-align: center;
}

.calendar-grid {
  margin-top: 9px;
}

.calendar-day {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 52px;
  color: var(--ds-color-text-secondary);
  border: 1px solid transparent;
  border-radius: 10px;
}

.calendar-day:hover {
  background: var(--ds-color-primary-light);
}

.calendar-day.is-muted {
  color: #c7ceda;
}

.calendar-day.is-active {
  color: #ffffff;
  font-weight: 700;
  background: var(--ds-color-primary);
  box-shadow: 0 5px 12px rgba(82, 107, 155, 0.2);
}

.day-number {
  font-size: 13px;
}

.day-events {
  position: absolute;
  bottom: 7px;
  display: flex;
  gap: 3px;
}

.day-events i {
  display: block;
  width: 4px;
  height: 4px;
  background: var(--ds-color-primary);
  border-radius: 50%;
}

.calendar-day.is-active .day-events i {
  background: var(--ds-color-accent);
}

.calendar-legend {
  display: flex;
  gap: 17px;
  margin-top: 22px;
  padding-top: 17px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
  border-top: 1px solid var(--ds-color-border);
}

.calendar-legend span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.legend-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.legend-dot.is-blue {
  background: var(--ds-color-primary);
}

.legend-dot.is-warm {
  background: var(--ds-color-accent);
}

.agenda-day-mark {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  color: var(--ds-color-primary);
  font-size: 17px;
  font-weight: 700;
  background: var(--ds-color-primary-light);
  border-radius: 12px;
}

.calendar-agenda-list {
  margin-top: 24px;
}

.calendar-agenda-item {
  display: grid;
  grid-template-columns: 45px 4px minmax(0, 1fr) 16px;
  align-items: center;
  min-height: 68px;
  gap: 12px;
  border-bottom: 1px solid #eff1f6;
}

.calendar-agenda-time {
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.calendar-agenda-line {
  height: 31px;
  background: var(--ds-color-primary);
  border-radius: 5px;
}

.calendar-agenda-line.is-warm {
  background: var(--ds-color-accent);
}

.calendar-agenda-line.is-danger {
  background: var(--ds-color-danger);
}

.calendar-agenda-content strong,
.calendar-agenda-content span {
  display: block;
}

.calendar-agenda-content strong {
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.calendar-agenda-content span {
  margin-top: 4px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.calendar-agenda-more {
  width: 15px;
  color: #c2cbd9;
}

.calendar-empty-slot {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 22px;
  padding: 13px 14px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  background: var(--ds-color-surface-muted);
  border-radius: 9px;
}

.calendar-empty-slot svg {
  width: 15px;
  color: var(--ds-color-success);
}

.calendar-note-strip {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 20px;
  padding: 17px 20px;
  background: #f8f5ed;
  border: 1px solid #eee4cb;
  border-radius: var(--ds-radius-lg);
}

.calendar-note-icon {
  display: grid;
  width: 35px;
  height: 35px;
  flex: 0 0 auto;
  place-items: center;
  color: #a87a2d;
  background: #fff4d7;
  border-radius: 10px;
}

.calendar-note-icon svg {
  width: 17px;
}

.calendar-note-strip strong,
.calendar-note-strip p {
  display: block;
}

.calendar-note-strip strong {
  font-size: 13px;
}

.calendar-note-strip p {
  margin: 3px 0 0;
  color: #8b7956;
  font-size: 12px;
}

.calendar-preview-label {
  margin-left: auto;
  color: #a87a2d;
  font-size: 11px;
}

@media (max-width: 980px) {
  .calendar-layout {
    grid-template-columns: 1fr;
  }
}
</style>
