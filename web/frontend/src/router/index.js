import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/dashboard',
    },
    {
      path: '/dashboard',
      component: () => import('../layouts/AppLayout.vue'),
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('../views/DashboardView.vue'),
          meta: {
            title: '工作台',
            description: '今天，也给自己留一点从容。',
          },
        },
        {
          path: 'tasks',
          name: 'tasks',
          component: () => import('../views/TasksView.vue'),
          meta: {
            title: '任务',
            description: '把复杂的事情拆成清晰的下一步。',
          },
        },
        {
          path: 'tasks/:id',
          name: 'task-detail',
          component: () => import('../views/TaskDetailView.vue'),
          meta: {
            title: '任务详情',
            description: '查看任务上下文和执行进度。',
          },
        },
        {
          path: 'calendar',
          name: 'calendar',
          component: () => import('../views/CalendarView.vue'),
          meta: {
            title: '日程管理',
            description: '让时间有安排，也留有余地。',
          },
        },
        {
          path: 'notes',
          name: 'notes',
          component: () => import('../views/NotesView.vue'),
          meta: {
            title: '内容收集',
            description: '先收下来，再慢慢整理成自己的东西。',
          },
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('../views/SettingsView.vue'),
          meta: {
            title: '设置',
            description: '管理偏好、数据和本地运行环境。',
          },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/dashboard',
    },
  ],
})

export default router
