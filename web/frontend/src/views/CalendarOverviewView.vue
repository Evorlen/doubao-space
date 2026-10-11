<script setup>
import CalendarBoard from '../components/calendar/CalendarBoard.vue'
</script>

<template>
  <div class="page-view page-view-wide calendar-overview-page">
    <section class="page-heading">
      <div class="page-heading-copy">
        <p class="eyebrow">DOUBAO SPACE</p>
        <h1>日历</h1>
        <p>把任务、豆包日程和个人计划放在同一张日历里，按日期看清接下来要做什么。</p>
      </div>
    </section>

    <CalendarBoard />
  </div>
</template>

<style scoped>
.calendar-overview-page {
  max-width: 1440px;
}
</style>
