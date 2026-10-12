<script setup>
import { computed } from 'vue'
import {
  ArrowDown,
  ArrowRight,
  CircleCheck,
  Clock,
  MoreFilled,
} from '@element-plus/icons-vue'

defineOptions({ name: 'TaskRow' })

const props = defineProps({
  task: {
    type: Object,
    required: true,
  },
  depth: {
    type: Number,
    default: 0,
  },
  expandedIds: {
    type: Object,
    required: true,
  },
  showCompleted: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['toggle', 'open', 'complete', 'command'])

const priorityLabels = {
  urgent: '紧急',
  high: '高优先级',
  medium: '中优先级',
  low: '低优先级',
}

const visibleChildren = computed(() => {
  const children = Array.isArray(props.task.children) ? props.task.children : []
  return props.showCompleted ? children : children.filter((child) => !child.completed)
})

const hasChildren = computed(() => visibleChildren.value.length > 0)
const isExpanded = computed(() => props.expandedIds.has(props.task.id))
const priority = computed(() => props.task.effectivePriority || props.task.effective_priority || props.task.priority || 'medium')
const priorityLabel = computed(() => priorityLabels[priority.value] || '中优先级')

function toggle() {
  if (hasChildren.value) {
    emit('toggle', props.task.id)
  }
}

function handleCommand(command) {
  emit('command', { command, task: props.task })
}
</script>

<template>
  <div class="task-tree-node">
    <div
      class="task-row"
      :class="{ 'is-complete': task.completed, 'has-children': hasChildren }"
      :style="{ '--task-depth': depth }"
      tabindex="0"
      @dblclick="emit('open', task)"
      @keydown.enter="emit('open', task)"
    >
      <button
        class="task-expand-button"
        type="button"
        :aria-label="isExpanded ? '折叠子任务' : '展开子任务'"
        :disabled="!hasChildren"
        @click.stop="toggle"
      >
        <ArrowDown v-if="isExpanded" />
        <ArrowRight v-else-if="hasChildren" />
        <span v-else class="task-expand-placeholder"></span>
      </button>

      <button
        class="task-check-button"
        type="button"
        :aria-label="task.completed ? '任务已完成' : '标记任务完成'"
        @click.stop="emit('complete', task)"
      >
        <CircleCheck v-if="task.completed" />
        <span v-else class="task-check-ring"></span>
      </button>

      <div class="task-row-content" @click="emit('open', task)">
        <div class="task-title-line">
          <span class="task-title">{{ task.title }}</span>
          <span v-for="tag in task.tags?.slice(0, 2)" :key="tag" class="task-tag">{{ tag }}</span>
          <span v-if="task.tags?.length > 2" class="task-tag task-tag-more">+{{ task.tags.length - 2 }}</span>
        </div>
        <div class="task-meta-line">
          <span class="task-due" :class="{ 'is-overdue': task.dueTone === 'danger' }">
            <Clock />
            {{ task.due || '无截止时间' }}
          </span>
          <span v-if="task.progress" class="task-progress">{{ task.progress }}</span>
        </div>
      </div>

      <span class="task-priority" :class="`is-${priority}`">{{ priorityLabel }}</span>

      <el-dropdown trigger="click" @command="handleCommand">
        <button class="task-more-button" type="button" aria-label="更多操作" @click.stop>
          <MoreFilled />
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="open">打开详情</el-dropdown-item>
            <el-dropdown-item command="add-child" :disabled="task.completed || task.depth >= 5">添加子任务</el-dropdown-item>
            <el-dropdown-item command="import-child" :disabled="task.completed || task.depth >= 5">导入子任务</el-dropdown-item>
            <el-dropdown-item command="move" :disabled="task.completed">移动任务</el-dropdown-item>
            <el-dropdown-item command="delete" divided>删除任务</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>

    <div v-if="isExpanded" class="task-children">
      <TaskRow
        v-for="child in visibleChildren"
        :key="child.id"
        :task="child"
        :depth="depth + 1"
        :expanded-ids="expandedIds"
        :show-completed="showCompleted"
        @toggle="emit('toggle', $event)"
        @open="emit('open', $event)"
        @complete="emit('complete', $event)"
        @command="emit('command', $event)"
      />
    </div>
  </div>
</template>

<style scoped>
.task-tree-node {
  position: relative;
}

.task-tree-node + .task-tree-node {
  border-top: 1px solid #f0f2f7;
}

.task-row {
  display: grid;
  grid-template-columns: 24px 26px minmax(0, 1fr) 100px 34px;
  align-items: center;
  min-height: 68px;
  gap: 10px;
  padding: 8px 10px 8px calc(10px + var(--task-depth) * 27px);
  border-radius: 10px;
  transition: background-color 160ms ease;
}

.task-row:hover,
.task-row:focus-visible {
  background: #f7f9fd;
}

.task-expand-button,
.task-check-button,
.task-more-button {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  color: var(--ds-color-text-tertiary);
  cursor: pointer;
  background: transparent;
  border-radius: 8px;
}

.task-expand-button:hover,
.task-more-button:hover {
  color: var(--ds-color-primary);
  background: var(--ds-color-primary-light);
}

.task-expand-button:disabled {
  cursor: default;
}

.task-expand-button svg,
.task-more-button svg {
  width: 16px;
  height: 16px;
}

.task-expand-placeholder {
  width: 6px;
  height: 6px;
  background: #cbd2e0;
  border-radius: 50%;
}

.task-check-button {
  color: var(--ds-color-primary);
}

.task-check-button:hover {
  color: var(--ds-color-primary-hover);
}

.task-check-button svg {
  width: 20px;
  height: 20px;
}

.task-check-ring {
  display: block;
  width: 18px;
  height: 18px;
  border: 1.5px solid #bdc7d8;
  border-radius: 50%;
}

.task-row-content {
  min-width: 0;
  cursor: pointer;
}

.task-title-line {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 7px;
}

.task-title {
  overflow: hidden;
  color: var(--ds-color-text-primary);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-row.is-complete .task-title {
  color: var(--ds-color-text-disabled);
  text-decoration: line-through;
}

.task-tag {
  flex: 0 0 auto;
  padding: 3px 7px;
  color: var(--ds-color-primary);
  font-size: 11px;
  background: var(--ds-color-primary-light);
  border-radius: 5px;
}

.task-tag-more {
  color: var(--ds-color-text-tertiary);
  background: #f0f2f6;
}

.task-meta-line {
  display: flex;
  align-items: center;
  gap: 13px;
  margin-top: 6px;
  color: var(--ds-color-text-tertiary);
  font-size: 12px;
}

.task-due,
.task-progress {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.task-due svg {
  width: 13px;
  height: 13px;
}

.task-due.is-overdue {
  color: var(--ds-color-danger);
}

.task-progress {
  padding-left: 13px;
  border-left: 1px solid var(--ds-color-border);
}

.task-priority {
  justify-self: center;
  padding: 5px 8px;
  font-size: 11px;
  white-space: nowrap;
  border-radius: 6px;
}

.task-priority.is-urgent,
.task-priority.is-high {
  color: #c0525b;
  background: #fff0f0;
}

.task-priority.is-medium {
  color: #9b6e24;
  background: #fff8e8;
}

.task-priority.is-low {
  color: var(--ds-color-text-tertiary);
  background: #f1f3f7;
}

.task-more-button {
  justify-self: end;
}

.task-children {
  position: relative;
}

.task-children::before {
  position: absolute;
  top: 0;
  bottom: 22px;
  left: 39px;
  width: 1px;
  content: '';
  background: #e8ecf4;
}
</style>
