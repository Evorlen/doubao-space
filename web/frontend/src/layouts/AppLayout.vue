<script setup>
import { computed } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import {
  Calendar,
  CircleCheck,
  Bowl,
  Film,
  HomeFilled,
  List,
  Search,
  Setting,
  Timer,
} from '@element-plus/icons-vue'

const route = useRoute()

const navigation = [
  { label: '工作台', to: '/dashboard', icon: HomeFilled },
  { label: '任务', to: '/dashboard/tasks', icon: List },
  { label: '豆包', to: '/dashboard/doubao', icon: Bowl },
  { label: '日历', to: '/dashboard/calendar', icon: Calendar },
  { label: '个人计划', to: '/dashboard/plans', icon: CircleCheck },
  { label: '娱乐', to: '/dashboard/entertainment', icon: Film },
]

const pageTitle = computed(() => route.meta.title || '豆包空间')

function isActive(item) {
  if (item.to === '/dashboard') {
    return route.path === item.to
  }
  return route.path.startsWith(item.to)
}
</script>

<template>
  <el-container class="app-shell">
    <el-aside width="232px" class="app-sidebar">
      <div class="brand-lockup">
        <div class="brand-mark">豆</div>
        <div>
          <div class="brand-name">豆包空间</div>
          <div class="brand-caption">个人工作台</div>
        </div>
      </div>

      <div class="sidebar-label">我的空间</div>
      <nav class="primary-nav" aria-label="主导航">
        <RouterLink
          v-for="item in navigation"
          :key="item.to"
          :to="item.to"
          class="nav-item"
          :class="{ 'is-active': isActive(item) }"
        >
          <component :is="item.icon" class="nav-icon" />
          <span class="nav-label">{{ item.label }}</span>
          <span v-if="item.badge" class="nav-badge">{{ item.badge }}</span>
        </RouterLink>
      </nav>

      <div class="sidebar-spacer"></div>

      <div class="sidebar-focus-card">
        <div class="sidebar-focus-icon"><Timer /></div>
        <div class="sidebar-focus-title">给今天留一点空间</div>
        <div class="sidebar-focus-copy">一次只处理一件重要的事。</div>
      </div>

      <RouterLink
        to="/dashboard/settings"
        class="nav-item sidebar-settings"
        :class="{ 'is-active': isActive({ to: '/dashboard/settings' }) }"
      >
        <Setting class="nav-icon" />
        <span class="nav-label">设置</span>
      </RouterLink>

      <div class="sidebar-footer">
        <span class="sidebar-status-dot"></span>
        <span>本地运行 · 私人数据</span>
      </div>
    </el-aside>

    <el-container class="content-shell">
      <el-header class="app-topbar">
        <div class="topbar-context">
          <span class="topbar-kicker">DOUBAO SPACE</span>
          <span class="topbar-separator">/</span>
          <span>{{ pageTitle }}</span>
        </div>

        <div class="topbar-actions">
          <button class="icon-button topbar-search" type="button" aria-label="搜索">
            <Search />
          </button>
          <span class="local-pill"><span class="local-pill-dot"></span>本地运行</span>
          <button class="avatar-button" type="button" aria-label="个人资料">JZ</button>
        </div>
      </el-header>

      <el-main class="app-main">
        <RouterView />
      </el-main>
    </el-container>
  </el-container>
</template>
