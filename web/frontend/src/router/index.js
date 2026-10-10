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
            title: '豆包日历',
            description: '照顾豆包的每一天，也把安排变得轻松。',
          },
        },
        {
          path: 'plans',
          name: 'plans',
          component: () => import('../views/PersonalPlansView.vue'),
          meta: {
            title: '个人计划',
            description: '把周期性的小事交给计划记住。',
          },
        },
        {
          path: 'entertainment',
          name: 'entertainment',
          component: () => import('../views/EntertainmentView.vue'),
          meta: {
            title: '娱乐',
            description: '把想读、想看和已经喜欢的作品放在一起。',
          },
        },
        {
          path: 'notes',
          redirect: '/dashboard/entertainment',
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
