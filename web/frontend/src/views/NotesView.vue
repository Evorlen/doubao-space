<script setup>
import { ref } from 'vue'
import {
  Collection,
  Document,
  Link,
  MoreFilled,
  Plus,
  Search,
  Star,
} from '@element-plus/icons-vue'

const activeCategory = ref('全部')
const categories = ['全部', '稍后阅读', '灵感', '链接']

const notes = [
  {
    title: '好的工具应该让人更接近自己的节奏',
    excerpt: '记录一个关于个人工具的想法：减少整理本身的负担，把注意力留给真正要做的事情。',
    type: '灵感',
    source: '刚刚',
    tone: 'warm',
    icon: Star,
  },
  {
    title: '独立工作者的每周复盘清单',
    excerpt: '下周找时间完整读完，重点关注如何让复盘从记录变成下一步行动。',
    type: '稍后阅读',
    source: '昨天',
    tone: 'blue',
    icon: Document,
  },
  {
    title: 'https://example.com/quiet-tools',
    excerpt: '一个关于安静软件和数字生活的链接，之后整理到参考资料里。',
    type: '链接',
    source: '9 月 20 日',
    tone: 'green',
    icon: Link,
  },
]
</script>

<template>
  <div class="page-view notes-page">
    <section class="page-heading">
      <div class="page-heading-copy">
        <p class="eyebrow">CAPTURE SPACE</p>
        <h1>内容收集</h1>
        <p>先收下来，再慢慢整理成自己的东西。灵感不需要马上变得完整。</p>
      </div>
      <div class="page-heading-actions">
        <el-button class="soft-button"><Search class="button-icon" />搜索内容</el-button>
        <el-button type="primary"><Plus class="button-icon" />快速收集</el-button>
      </div>
    </section>

    <section class="notes-overview-grid">
      <div class="notes-overview-main surface-card">
        <div class="notes-category-row">
          <div class="notes-category-tabs" role="tablist" aria-label="内容分类">
            <button
              v-for="category in categories"
              :key="category"
              class="notes-category-tab"
              :class="{ 'is-active': activeCategory === category }"
              type="button"
              @click="activeCategory = category"
            >
              {{ category }}
            </button>
          </div>
          <span class="notes-count">24 条内容</span>
        </div>

        <div class="notes-grid">
          <article v-for="note in notes" :key="note.title" class="note-card">
            <div class="note-card-topline">
              <span class="note-type-icon" :class="`is-${note.tone}`"><component :is="note.icon" /></span>
              <button class="note-more" type="button" aria-label="更多内容操作"><MoreFilled /></button>
            </div>
            <h2>{{ note.title }}</h2>
            <p>{{ note.excerpt }}</p>
            <div class="note-card-footer">
              <span class="note-type-label">{{ note.type }}</span>
              <span>{{ note.source }}</span>
            </div>
          </article>
        </div>
      </div>

      <aside class="notes-side-column">
        <div class="capture-card">
          <div class="capture-card-icon"><Collection /></div>
          <h2>随手记下</h2>
          <p>还没想好放在哪里？先把它收进来，之后再决定。</p>
          <el-button type="primary">开始记录 <Plus class="capture-button-icon" /></el-button>
        </div>

        <div class="notes-tip-card surface-card">
          <div class="notes-tip-mark">i</div>
          <div>
            <strong>收集和整理是两件事</strong>
            <p>先保护灵感的流动，再给它们找到合适的位置。</p>
          </div>
        </div>
      </aside>
    </section>
  </div>
</template>

<style scoped>
.notes-page {
  max-width: 1240px;
}

.notes-overview-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  gap: 20px;
}

.notes-overview-main {
  min-height: 540px;
  padding: 24px;
}

.notes-category-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--ds-color-border);
}

.notes-category-tabs {
  display: flex;
  gap: 4px;
}

.notes-category-tab {
  padding: 8px 11px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  cursor: pointer;
  background: transparent;
  border-radius: 8px;
}

.notes-category-tab:hover,
.notes-category-tab.is-active {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.notes-category-tab.is-active {
  font-weight: 700;
}

.notes-count {
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.notes-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  padding-top: 18px;
}

.note-card {
  position: relative;
  min-height: 190px;
  padding: 18px;
  background: var(--ds-color-surface-muted);
  border: 1px solid transparent;
  border-radius: 13px;
  transition: border-color 160ms ease, transform 160ms ease, box-shadow 160ms ease;
}

.note-card:hover {
  border-color: var(--ds-color-border-strong);
  box-shadow: var(--ds-shadow-card);
  transform: translateY(-2px);
}

.note-card-topline {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.note-type-icon {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border-radius: 10px;
}

.note-type-icon svg {
  width: 16px;
}

.note-type-icon.is-warm {
  color: #aa7c2f;
  background: #fff3d7;
}

.note-type-icon.is-blue {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.note-type-icon.is-green {
  color: var(--ds-color-success);
  background: #edf8e9;
}

.note-more {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  color: var(--ds-color-text-tertiary);
  cursor: pointer;
  background: transparent;
  border-radius: 8px;
}

.note-more:hover {
  color: var(--ds-color-primary);
  background: #ffffff;
}

.note-more svg {
  width: 16px;
}

.note-card h2 {
  display: -webkit-box;
  margin: 18px 0 8px;
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-size: 14px;
  line-height: 1.5;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.note-card p {
  display: -webkit-box;
  margin: 0;
  overflow: hidden;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  line-height: 1.7;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.note-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 15px;
  color: #a0aabc;
  font-size: 11px;
}

.note-type-label {
  color: var(--ds-color-primary);
}

.notes-side-column {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.capture-card {
  padding: 24px;
  color: #ffffff;
  background: linear-gradient(145deg, #415987, #667bab);
  border-radius: var(--ds-radius-lg);
  box-shadow: var(--ds-shadow-hero);
}

.capture-card-icon {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  color: var(--ds-color-brand-700);
  background: var(--ds-color-accent);
  border-radius: 12px;
}

.capture-card-icon svg {
  width: 19px;
}

.capture-card h2 {
  margin: 20px 0 8px;
  font-size: 20px;
}

.capture-card p {
  margin: 0;
  color: #dae3f5;
  font-size: 12px;
  line-height: 1.75;
}

.capture-card .el-button {
  margin-top: 23px;
  color: var(--ds-color-brand-900);
  background: var(--ds-color-accent);
  border-color: var(--ds-color-accent);
}

.capture-card .el-button:hover {
  background: #f9d68f;
  border-color: #f9d68f;
}

.capture-button-icon {
  width: 14px;
  margin-left: 5px;
}

.notes-tip-card {
  display: flex;
  gap: 12px;
  padding: 18px;
}

.notes-tip-mark {
  display: grid;
  width: 25px;
  height: 25px;
  flex: 0 0 auto;
  place-items: center;
  color: var(--ds-color-primary);
  font-family: Georgia, serif;
  font-size: 16px;
  font-weight: 700;
  background: var(--ds-color-primary-light);
  border-radius: 50%;
}

.notes-tip-card strong {
  display: block;
  font-size: 13px;
}

.notes-tip-card p {
  margin: 5px 0 0;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
  line-height: 1.65;
}

@media (max-width: 980px) {
  .notes-overview-grid {
    grid-template-columns: 1fr;
  }

.notes-side-column {
    display: grid;
    grid-template-columns: 1fr 1fr;
  }
}
</style>
