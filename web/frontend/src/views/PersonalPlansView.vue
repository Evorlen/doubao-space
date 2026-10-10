<script setup>
import SchedulePanel from '../components/calendar/SchedulePanel.vue'
</script>

<template>
  <div class="page-view page-view-wide personal-plans-page">
    <SchedulePanel variant="personal" />
  </div>
</template>

<style scoped>
.personal-plans-page {
  max-width: 1440px;
}
</style>
