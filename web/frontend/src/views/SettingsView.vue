<script setup>
import { ref } from 'vue'
import {
  Bell,
  Check,
  DataAnalysis,
  Download,
  Folder,
  Lock,
  Monitor,
  Setting,
  Upload,
} from '@element-plus/icons-vue'

const appearance = ref('light')
const reduceMotion = ref(false)
const dailyReminder = ref(true)
</script>

<template>
  <div class="page-view settings-page">
    <section class="page-heading">
      <div class="page-heading-copy">
        <p class="eyebrow">YOUR SPACE</p>
        <h1>设置</h1>
        <p>让豆包空间更符合你的工作和生活节奏。</p>
      </div>
    </section>

    <section class="settings-layout">
      <nav class="settings-nav" aria-label="设置分类">
        <a href="#appearance" class="settings-nav-item is-active"><Monitor />外观偏好</a>
        <a href="#notifications" class="settings-nav-item"><Bell />提醒与通知</a>
        <a href="#data" class="settings-nav-item"><DataAnalysis />数据管理</a>
        <a href="#about" class="settings-nav-item"><Setting />关于应用</a>
      </nav>

      <div class="settings-content">
        <section id="appearance" class="surface-card settings-card">
          <div class="settings-card-heading">
            <div>
              <h2>外观偏好</h2>
              <p>选择一个让你愿意每天打开的界面。</p>
            </div>
            <Monitor class="settings-heading-icon" />
          </div>
          <div class="setting-row setting-row-column">
            <div>
              <strong>界面主题</strong>
              <span>当前版本优先支持明亮主题。</span>
            </div>
            <div class="theme-options">
              <button class="theme-option is-selected" type="button" @click="appearance = 'light'">
                <span class="theme-preview light-preview"><i></i><b></b></span>
                <span>明亮</span>
                <Check v-if="appearance === 'light'" />
              </button>
              <button class="theme-option" type="button" @click="appearance = 'dark'">
                <span class="theme-preview dark-preview"><i></i><b></b></span>
                <span>深色</span>
                <span v-if="appearance === 'dark'" class="theme-check"><Check /></span>
              </button>
            </div>
          </div>
          <div class="setting-row">
            <div>
              <strong>减少动效</strong>
              <span>降低页面切换和状态变化的动画。</span>
            </div>
            <el-switch v-model="reduceMotion" />
          </div>
        </section>

        <section id="notifications" class="surface-card settings-card">
          <div class="settings-card-heading">
            <div>
              <h2>提醒与通知</h2>
              <p>提醒应该帮助你回到当下，而不是制造新的打扰。</p>
            </div>
            <Bell class="settings-heading-icon" />
          </div>
          <div class="setting-row">
            <div>
              <strong>每日开始提醒</strong>
              <span>在每天第一次打开工作台时显示今日概览。</span>
            </div>
            <el-switch v-model="dailyReminder" />
          </div>
          <div class="setting-row is-disabled">
            <div>
              <strong>截止时间提醒</strong>
              <span>日程模块和通知规则确定后开放。</span>
            </div>
            <el-switch disabled />
          </div>
        </section>

        <section id="data" class="surface-card settings-card">
          <div class="settings-card-heading">
            <div>
              <h2>数据管理</h2>
              <p>数据保存在本地，备份和迁移都由你决定。</p>
            </div>
            <Lock class="settings-heading-icon" />
          </div>
          <div class="data-path-row">
            <span class="data-path-icon"><Folder /></span>
            <div>
              <strong>当前数据目录</strong>
              <span>项目开发目录 / data</span>
            </div>
            <el-button text>打开目录</el-button>
          </div>
          <div class="data-actions">
            <el-button><Download class="button-icon" />导出数据</el-button>
            <el-button><Upload class="button-icon" />导入备份</el-button>
          </div>
          <div class="data-tip"><Lock /><span>数据目录不会被上传到任何远程服务。</span></div>
        </section>

        <section id="about" class="about-card">
          <div class="about-mark">豆</div>
          <div>
            <strong>豆包空间</strong>
            <span>一个从本地开始的个人工作台</span>
          </div>
          <small>v0.1.0 · 本地开发版</small>
        </section>
      </div>
    </section>
  </div>
</template>

<style scoped>
.settings-page {
  max-width: 1060px;
}

.settings-layout {
  display: grid;
  grid-template-columns: 190px minmax(0, 1fr);
  align-items: start;
  gap: 28px;
}

.settings-nav {
  position: sticky;
  top: 24px;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.settings-nav-item {
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 40px;
  padding: 0 11px;
  color: var(--ds-color-text-tertiary);
  font-size: 13px;
  border-radius: 8px;
}

.settings-nav-item:hover,
.settings-nav-item.is-active {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.settings-nav-item.is-active {
  font-weight: 700;
}

.settings-nav-item svg {
  width: 16px;
}

.settings-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.settings-card {
  padding: 25px;
}

.settings-card-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--ds-color-border);
}

.settings-card-heading h2 {
  margin: 0;
  font-size: 16px;
}

.settings-card-heading p {
  margin: 5px 0 0;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.settings-heading-icon {
  width: 20px;
  color: #c1cad9;
}

.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  min-height: 72px;
  border-bottom: 1px solid #eff1f6;
}

.setting-row:last-child {
  border-bottom: 0;
}

.setting-row strong,
.setting-row span {
  display: block;
}

.setting-row strong {
  color: var(--ds-color-text-primary);
  font-size: 13px;
}

.setting-row span {
  margin-top: 4px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.setting-row.is-disabled {
  opacity: 0.55;
}

.setting-row-column {
  display: block;
  padding: 20px 0;
}

.theme-options {
  display: flex;
  gap: 12px;
  margin-top: 16px;
}

.theme-option {
  position: relative;
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 8px 12px 8px 8px;
  color: var(--ds-color-text-secondary);
  font-size: 12px;
  cursor: pointer;
  background: #ffffff;
  border: 1px solid var(--ds-color-border);
  border-radius: 9px;
}

.theme-option.is-selected {
  color: var(--ds-color-primary);
  border-color: var(--ds-color-primary);
  box-shadow: 0 0 0 3px rgba(82, 107, 155, 0.1);
}

.theme-preview {
  display: flex;
  flex-direction: column;
  width: 28px;
  height: 21px;
  gap: 3px;
  padding: 4px;
  border-radius: 4px;
}

.theme-preview i,
.theme-preview b {
  display: block;
  height: 3px;
  border-radius: 2px;
}

.light-preview {
  background: #f5f7fb;
}

.light-preview i {
  width: 12px;
  background: #526b9b;
}

.light-preview b {
  width: 17px;
  background: #d3daea;
}

.dark-preview {
  background: #1f2a44;
}

.dark-preview i {
  width: 12px;
  background: #f7c873;
}

.dark-preview b {
  width: 17px;
  background: #526b9b;
}

.theme-option > svg,
.theme-check svg {
  width: 14px;
  margin-left: 2px;
}

.theme-option > svg {
  color: var(--ds-color-primary);
}

.data-path-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 0;
  border-bottom: 1px solid #eff1f6;
}

.data-path-icon {
  display: grid;
  width: 37px;
  height: 37px;
  place-items: center;
  color: #a87a2d;
  background: #fff4d7;
  border-radius: 10px;
}

.data-path-icon svg {
  width: 18px;
}

.data-path-row strong,
.data-path-row span:not(.data-path-icon) {
  display: block;
}

.data-path-row strong {
  font-size: 13px;
}

.data-path-row span:not(.data-path-icon) {
  margin-top: 4px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.data-path-row .el-button {
  margin-left: auto;
}

.data-actions {
  display: flex;
  gap: 10px;
  padding: 18px 0 6px;
}

.data-tip {
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.data-tip svg {
  width: 13px;
}

.about-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 15px 17px;
  color: var(--ds-color-text-secondary);
  background: #eef1f7;
  border-radius: var(--ds-radius-lg);
}

.about-mark {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  color: var(--ds-color-brand-900);
  font-size: 14px;
  font-weight: 800;
  background: var(--ds-color-accent);
  border-radius: 10px;
}

.about-card strong,
.about-card span {
  display: block;
}

.about-card strong {
  font-size: 12px;
}

.about-card span {
  margin-top: 2px;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

.about-card small {
  margin-left: auto;
  color: var(--ds-color-text-tertiary);
  font-size: 11px;
}

@media (max-width: 860px) {
  .settings-layout {
    grid-template-columns: 1fr;
  }

  .settings-nav {
    position: static;
    flex-direction: row;
    overflow-x: auto;
  }
}
</style>
