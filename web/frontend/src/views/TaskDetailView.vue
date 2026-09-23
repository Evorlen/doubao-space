<script setup>
import { useRouter } from 'vue-router'
import {
  ArrowLeft,
  Calendar,
  CircleCheck,
  Edit,
  Flag,
  Link,
  MoreFilled,
  Paperclip,
} from '@element-plus/icons-vue'

const router = useRouter()

const subtasks = [
  { title: '回顾上半年目标完成情况', completed: true, due: '今天 12:00' },
  { title: '确定下半年的三个关键词', completed: false, due: '今天 16:00' },
  { title: '整理到可执行的月度计划', completed: false, due: '明天 18:00' },
  { title: '和自己确认最终版本', completed: true, due: '周五 18:00' },
]

function goBack() {
  router.back()
}
</script>

<template>
  <div class="page-view task-detail-page">
    <button class="detail-back-button" type="button" @click="goBack"><ArrowLeft />返回任务</button>

    <section class="detail-heading">
      <div class="detail-heading-copy">
        <div class="detail-path">个人生活 / 规划</div>
        <h1>整理个人年度规划</h1>
        <div class="detail-heading-meta">
          <span class="detail-status"><span></span>未完成</span>
          <span>更新于今天 09:12</span>
        </div>
      </div>
      <div class="page-heading-actions">
        <el-button class="soft-button"><Edit class="button-icon" />编辑任务</el-button>
        <el-button circle aria-label="更多任务操作"><MoreFilled /></el-button>
      </div>
    </section>

    <section class="detail-layout">
      <div class="detail-main-column">
        <div class="surface-card detail-description-card">
          <div class="detail-section-heading">
            <h2>任务说明</h2>
            <span class="demo-badge">预览</span>
          </div>
          <p class="detail-description">
            回顾过去一段时间真正重要的事情，再把下半年的方向拆成可以在每个月行动的计划。不要追求一次写得完美，先留下一个愿意执行的版本。
          </p>
          <div class="detail-tags">
            <span class="task-tag">生活</span>
            <span class="task-tag">重点</span>
            <span class="task-tag">规划</span>
          </div>
        </div>

        <div class="surface-card detail-subtasks-card">
          <div class="detail-section-heading">
            <div>
              <h2>子任务</h2>
              <p>完成 2 / 4</p>
            </div>
            <el-button text type="primary">添加子任务</el-button>
          </div>
          <div class="detail-progress-bar"><span style="width: 50%"></span></div>
          <div class="subtask-list">
            <div v-for="subtask in subtasks" :key="subtask.title" class="subtask-item" :class="{ 'is-complete': subtask.completed }">
              <span class="subtask-check"><CircleCheck v-if="subtask.completed" /></span>
              <span class="subtask-title">{{ subtask.title }}</span>
              <span class="subtask-due"><Calendar />{{ subtask.due }}</span>
              <MoreFilled class="subtask-more" />
            </div>
          </div>
        </div>

        <div class="surface-card detail-description-card">
          <div class="detail-section-heading">
            <h2>附件与链接</h2>
            <el-button text type="primary">添加内容</el-button>
          </div>
          <div class="detail-assets-empty">
            <span class="asset-empty-icon"><Paperclip /></span>
            <div>
              <strong>还没有附件或链接</strong>
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
              <span class="property-label"><Calendar />截止时间</span>
              <strong class="property-value is-danger">今天 18:00</strong>
            </div>
            <div class="property-row">
              <span class="property-label"><Flag />最终优先级</span>
              <strong class="property-value is-high">高优先级</strong>
            </div>
            <div class="property-row">
              <span class="property-label"><CircleCheck />完成进度</span>
              <strong class="property-value">2 / 4</strong>
            </div>
          </div>
        </div>

        <div class="surface-card detail-note-card">
          <div class="detail-note-icon"><CircleCheck /></div>
          <h3>完成前先确认</h3>
          <p>所有子任务完成后，才可以将这个父任务标记为完成。</p>
        </div>

        <div class="detail-meta-card">
          <span>创建于 2026 年 9 月 18 日</span>
          <span>任务 ID · #0001</span>
        </div>
      </aside>
    </section>
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
  height: 6px;
  margin-top: 20px;
  overflow: hidden;
  background: #eef1f6;
  border-radius: var(--ds-radius-pill);
}

.detail-progress-bar span {
  display: block;
  height: 100%;
  background: var(--ds-color-primary);
  border-radius: inherit;
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
  border-bottom: 1px solid #eff1f6;
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
