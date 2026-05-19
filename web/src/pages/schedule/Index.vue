<script setup lang="ts">
import { ref, shallowRef, reactive, onMounted, onUnmounted, computed, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { scheduleApi, teamApi, userApi, scheduleICalApi } from '@/api'
import type { Schedule, Team, User, OnCallShift } from '@/types'
import { getErrorMessage } from '@/utils/format'

import ScheduleSidebar from './ScheduleSidebar.vue'
import ScheduleModal from './ScheduleModal.vue'
import ShiftModal from './ShiftModal.vue'
import ParticipantsList from './ParticipantsList.vue'
import PageHeader from '@/components/common/PageHeader.vue'

const message = useMessage()
const { t } = useI18n()

// ===== Color palette for users (restrained) =====
const userColors = [
  'var(--sre-info)',
  'var(--sre-success)',
  'var(--sre-warning)',
  'var(--sre-critical)',
  'var(--sre-aurora-3)',
  'var(--sre-brand-accent)',
  'var(--sre-aurora-4)',
  'var(--sre-user-slate)',
]
const userColorMap = ref<Map<number, string>>(new Map())

function getUserColor(userId: number): string {
  if (!userColorMap.value.has(userId)) {
    const idx = userColorMap.value.size % userColors.length
    userColorMap.value.set(userId, userColors[idx])
  }
  return userColorMap.value.get(userId)!
}

function getUserName(userId: number): string {
  const u = users.value.find(u => u.id === userId)
  return u ? (u.display_name || u.username) : `#${userId}`
}

// ===== Data =====
const loading = ref(false)
const schedules = shallowRef<Schedule[]>([])
const teams = shallowRef<Team[]>([])
const users = shallowRef<User[]>([])
const onCallMap = ref<Record<number, User | null>>({})
const selectedSchedule = ref<Schedule | null>(null)

// ===== Week navigation =====
function getMonday(date: Date): Date {
  const d = new Date(date)
  const day = d.getDay()
  const diff = d.getDate() - day + (day === 0 ? -6 : 1)
  d.setDate(diff)
  d.setHours(0, 0, 0, 0)
  return d
}

const currentWeekStart = ref(getMonday(new Date()))

function prevWeek() {
  currentWeekStart.value = new Date(currentWeekStart.value.getTime() - 7 * 86400000)
}
function nextWeek() {
  currentWeekStart.value = new Date(currentWeekStart.value.getTime() + 7 * 86400000)
}
function goToday() {
  currentWeekStart.value = getMonday(new Date())
}

const weekDays = computed(() =>
  Array.from({ length: 7 }, (_, i) => {
    const d = new Date(currentWeekStart.value)
    d.setDate(d.getDate() + i)
    return d
  })
)

const weekRangeLabel = computed(() => {
  const start = weekDays.value[0]
  const end = weekDays.value[6]
  const fmt = (d: Date) => `${d.getMonth() + 1}/${d.getDate()}`
  return `${start.getFullYear()} · ${fmt(start)} – ${fmt(end)}`
})

// Time axis
const timeLabels = Array.from({ length: 13 }, (_, i) => `${(i * 2).toString().padStart(2, '0')}:00`)

// Current time line
const currentTimePercent = ref(0)
function updateCurrentTime() {
  const now = new Date()
  const minutes = now.getHours() * 60 + now.getMinutes()
  currentTimePercent.value = (minutes / (24 * 60)) * 100
}
let currentTimeInterval: ReturnType<typeof setInterval>
onMounted(() => { updateCurrentTime(); currentTimeInterval = setInterval(updateCurrentTime, 60000) })
onUnmounted(() => clearInterval(currentTimeInterval))

function isToday(d: Date): boolean {
  const now = new Date()
  return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth() && d.getDate() === now.getDate()
}

function isWeekend(d: Date): boolean {
  const day = d.getDay()
  return day === 0 || day === 6
}

// ===== Shifts =====
const shifts = shallowRef<OnCallShift[]>([])
const shiftsLoading = ref(false)

async function fetchShifts() {
  if (!selectedSchedule.value) return
  shiftsLoading.value = true
  try {
    const start = weekDays.value[0].toISOString()
    const end = new Date(weekDays.value[6].getTime() + 86400000).toISOString()
    const { data } = await scheduleApi.listShifts(selectedSchedule.value.id, { start, end })
    shifts.value = data.data || []
  } catch {
    shifts.value = []
  } finally {
    shiftsLoading.value = false
  }
}

watch([selectedSchedule, currentWeekStart], () => {
  fetchShifts()
}, { immediate: false })

function getShiftsForDay(day: Date): OnCallShift[] {
  const dayStart = new Date(day)
  dayStart.setHours(0, 0, 0, 0)
  const dayEnd = new Date(day)
  dayEnd.setHours(23, 59, 59, 999)
  return shifts.value.filter(s => {
    const start = new Date(s.start_time)
    const end = new Date(s.end_time)
    return start <= dayEnd && end >= dayStart
  })
}

function isShiftActive(shift: OnCallShift): boolean {
  const now = Date.now()
  return new Date(shift.start_time).getTime() <= now && new Date(shift.end_time).getTime() >= now
}

function shiftStyle(shift: OnCallShift, day: Date): Record<string, string> {
  const dayStart = new Date(day)
  dayStart.setHours(0, 0, 0, 0)
  const dayEnd = new Date(day)
  dayEnd.setHours(24, 0, 0, 0)

  const shiftStart = new Date(shift.start_time)
  const shiftEnd = new Date(shift.end_time)

  const effectiveStart = shiftStart < dayStart ? dayStart : shiftStart
  const effectiveEnd = shiftEnd > dayEnd ? dayEnd : shiftEnd

  const minutesInDay = 24 * 60
  const startMin = (effectiveStart.getTime() - dayStart.getTime()) / 60000
  const endMin = (effectiveEnd.getTime() - dayStart.getTime()) / 60000

  const top = (startMin / minutesInDay) * 100
  const height = Math.max(((endMin - startMin) / minutesInDay) * 100, 2)

  return {
    top: `${top}%`,
    height: `${height}%`,
    '--shift-color': getUserColor(shift.user_id),
  }
}

function formatShiftTime(shift: OnCallShift): string {
  const s = new Date(shift.start_time)
  const e = new Date(shift.end_time)
  const fmt = (d: Date) => `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
  return `${fmt(s)}–${fmt(e)}`
}

// ===== Generate Shifts =====
const showGenerateModal = ref(false)
const generateWeeks = ref(4)
const generating = ref(false)

async function handleGenerateShifts() {
  if (!selectedSchedule.value) return
  generating.value = true
  try {
    await scheduleApi.generateShifts(selectedSchedule.value.id, { weeks: generateWeeks.value })
    message.success(t('schedule.shiftsGenerated'))
    showGenerateModal.value = false
    fetchShifts()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    generating.value = false
  }
}

// ===== Component refs =====
const scheduleModalRef = ref<InstanceType<typeof ScheduleModal> | null>(null)
const shiftModalRef = ref<InstanceType<typeof ShiftModal> | null>(null)
const participantsRef = ref<InstanceType<typeof ParticipantsList> | null>(null)
const activeConfigTab = ref('config')

// ===== Data Fetching =====
async function fetchSchedules() {
  loading.value = true
  try {
    const { data } = await scheduleApi.list({ page: 1, page_size: 100 })
    schedules.value = data.data.list || []
    for (const s of schedules.value) {
      fetchOnCall(s.id)
    }
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

async function fetchOnCall(scheduleId: number) {
  try {
    const { data } = await scheduleApi.getCurrentOnCall(scheduleId)
    onCallMap.value[scheduleId] = data.data
  } catch {
    onCallMap.value[scheduleId] = null
  }
}

async function fetchTeams() {
  try {
    const { data } = await teamApi.list({ page: 1, page_size: 100 })
    teams.value = data.data.list || []
  } catch { /* silent */ }
}

async function fetchUsers() {
  try {
    const { data } = await userApi.list({ page: 1, page_size: 200 })
    users.value = data.data.list || []
  } catch { /* silent */ }
}

function selectSchedule(s: Schedule) {
  selectedSchedule.value = s
  activeConfigTab.value = 'config'
  participantsRef.value?.fetchParticipants()
  fetchShifts()
}

async function handleDeleteSchedule(id: number) {
  try {
    await scheduleApi.delete(id)
    message.success(t('schedule.scheduleDeleted'))
    if (selectedSchedule.value?.id === id) {
      selectedSchedule.value = null
    }
    fetchSchedules()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

function handleExportICal() {
  if (!selectedSchedule.value) return
  const url = scheduleICalApi.exportURL(selectedSchedule.value.id)
  window.open(url, '_blank')
}

function handleCalendarDayClick(day: Date, event: MouseEvent) {
  if (!selectedSchedule.value) return
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const relY = event.clientY - rect.top
  const fraction = relY / rect.height
  const totalMinutes = fraction * 24 * 60
  const hour = Math.floor(totalMinutes / 60)
  shiftModalRef.value?.openCreate(day, hour)
}

function handleScheduleSaved() {
  fetchSchedules()
}

function handleShiftSaved() {
  fetchShifts()
}

const currentOnCall = computed(() =>
  selectedSchedule.value ? onCallMap.value[selectedSchedule.value.id] : null
)

onMounted(() => {
  fetchSchedules()
  fetchTeams()
  fetchUsers()
})
</script>

<template>
  <div class="schedule-page">
    <!-- Page Header -->
    <PageHeader :title="t('schedule.pageTitle')" :subtitle="t('schedule.pageSubtitle') || 'Define rotations and escalation policies'">
      <template #actions>
        <n-button size="small" @click="scheduleModalRef?.openCreate()">+ {{ t('schedule.newSchedule') }}</n-button>
        <n-button
          size="small"
          type="primary"
          :disabled="!selectedSchedule"
          @click="shiftModalRef?.openCreate()"
        >
          + {{ t('schedule.newShift') }}
        </n-button>
      </template>
    </PageHeader>

    <div class="schedule-layout">
      <!-- Left Sidebar -->
      <aside class="schedule-sidebar-wrap">
        <ScheduleSidebar
          :schedules="schedules"
          :loading="loading"
          :selected-id="selectedSchedule?.id ?? null"
          :on-call-map="onCallMap"
          @select="selectSchedule"
          @create="scheduleModalRef?.openCreate()"
        />
      </aside>

      <!-- Right Detail Panel -->
      <section class="schedule-detail">
        <template v-if="selectedSchedule">
          <!-- Detail header: title + on-call + actions -->
          <div class="detail-topbar">
            <div class="detail-title-row">
              <div class="detail-title-block">
                <span class="sre-label-eyebrow">{{ t('schedule.scheduleLabel') || 'Schedule' }}</span>
                <h2 class="detail-title">
                  {{ selectedSchedule.name }}
                  <span v-if="selectedSchedule.team" class="detail-team">
                    <span class="sre-meta-divider"></span>
                    {{ selectedSchedule.team?.name }}
                  </span>
                </h2>
              </div>

              <!-- Current on-call badge -->
              <div v-if="currentOnCall" class="oncall-badge">
                <span class="oncall-eyebrow">{{ t('schedule.currentOnCall') || 'On-call now' }}</span>
                <div class="oncall-info">
                  <span class="sre-dot oncall-dot-current" :style="{ '--dot-color': getUserColor(currentOnCall.id) }"></span>
                  <span class="oncall-name">{{ currentOnCall.display_name || currentOnCall.username }}</span>
                  <span v-if="currentOnCall.email" class="oncall-email tnum">{{ currentOnCall.email }}</span>
                </div>
              </div>
            </div>

            <div class="detail-actions">
              <n-button size="small" tertiary @click="handleExportICal" :title="t('ical.exportHint')">
                {{ t('ical.exportCalendar') }}
              </n-button>
              <n-button size="small" tertiary @click="showGenerateModal = true">
                {{ t('schedule.generateShifts') }}
              </n-button>
              <n-button size="small" @click="scheduleModalRef?.openEdit(selectedSchedule!)">{{ t('common.edit') }}</n-button>
              <n-popconfirm @positive-click="handleDeleteSchedule(selectedSchedule.id)">
                <template #trigger>
                  <n-button size="small" type="error" quaternary>{{ t('common.delete') }}</n-button>
                </template>
                {{ t('schedule.deleteConfirm') }}
              </n-popconfirm>
            </div>
          </div>

          <!-- Week navigation -->
          <div class="week-nav">
            <div class="week-nav-left">
              <n-button size="tiny" quaternary @click="prevWeek">‹</n-button>
              <span class="week-label tnum">{{ weekRangeLabel }}</span>
              <n-button size="tiny" quaternary @click="nextWeek">›</n-button>
              <n-button size="tiny" tertiary @click="goToday">{{ t('schedule.today') }}</n-button>
            </div>
            <div v-if="shiftsLoading" class="week-nav-status">
              <n-spin size="small" />
            </div>
          </div>

          <!-- Calendar grid -->
          <div class="calendar-container">
            <div class="calendar-grid">
              <!-- Header row -->
              <div class="cal-header-row">
                <div class="cal-time-gutter" />
                <div
                  v-for="(day, i) in weekDays"
                  :key="i"
                  class="cal-day-header"
                  :class="{ today: isToday(day), weekend: isWeekend(day) }"
                >
                  <span class="cal-day-name">{{ [t('schedule.mon'),t('schedule.tue'),t('schedule.wed'),t('schedule.thu'),t('schedule.fri'),t('schedule.sat'),t('schedule.sun')][i] }}</span>
                  <span class="cal-day-num tnum" :class="{ today: isToday(day) }">{{ day.getDate() }}</span>
                </div>
              </div>

              <!-- Body -->
              <div class="cal-body">
                <div class="cal-time-gutter-body">
                  <div
                    v-for="label in timeLabels"
                    :key="label"
                    class="cal-time-label tnum"
                  >{{ label }}</div>
                </div>

                <div
                  v-for="(day, dayIdx) in weekDays"
                  :key="dayIdx"
                  class="cal-day-col"
                  :class="{ weekend: isWeekend(day), today: isToday(day) }"
                  @click.self="handleCalendarDayClick(day, $event)"
                >
                  <div
                    v-for="h in 24"
                    :key="h"
                    class="cal-hour-line"
                    :class="{ major: h % 2 === 1 }"
                    :style="{ top: `${((h - 1) / 24) * 100}%` }"
                  />

                  <div
                    v-if="isToday(day)"
                    class="current-time-line"
                    :style="{ top: `${currentTimePercent}%` }"
                  />

                  <div
                    v-for="shift in getShiftsForDay(day)"
                    :key="shift.id"
                    class="shift-block"
                    :class="{ 'is-now': isShiftActive(shift) }"
                    :style="shiftStyle(shift, day)"
                    @click.stop="shiftModalRef?.openEdit(shift)"
                  >
                    <div class="shift-user">{{ getUserName(shift.user_id) }}</div>
                    <div class="shift-time tnum">{{ formatShiftTime(shift) }}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Bottom Config Tabs -->
          <div class="config-tabs">
            <n-tabs v-model:value="activeConfigTab" type="line" size="small" animated>
              <n-tab-pane name="config" :tab="t('schedule.tabConfig')">
                <div class="config-grid">
                  <div class="config-item">
                    <span class="config-label">{{ t('schedule.rotationType') }}</span>
                    <span class="config-value">{{ selectedSchedule.rotation_type }}</span>
                  </div>
                  <div class="config-item">
                    <span class="config-label">{{ t('schedule.timezone') }}</span>
                    <span class="config-value tnum">{{ selectedSchedule.timezone }}</span>
                  </div>
                  <div class="config-item">
                    <span class="config-label">{{ t('schedule.handoffTime') }}</span>
                    <span class="config-value tnum">{{ selectedSchedule.handoff_time }}</span>
                  </div>
                  <div class="config-item">
                    <span class="config-label">{{ t('schedule.team') }}</span>
                    <span class="config-value">{{ selectedSchedule.team?.name || '—' }}</span>
                  </div>
                  <div class="config-item">
                    <span class="config-label">{{ t('schedule.severityFilter') }}</span>
                    <span class="config-value">
                      <template v-if="selectedSchedule.severity_filter">{{ selectedSchedule.severity_filter }}</template>
                      <span v-else class="config-muted">{{ t('schedule.allSeverities') }}</span>
                    </span>
                  </div>
                  <div class="config-item">
                    <span class="config-label">{{ t('common.status') }}</span>
                    <span class="config-value">
                      <span class="sre-dot" :class="selectedSchedule.is_enabled ? 'sre-dot--success' : 'sre-dot--muted'"></span>
                      {{ selectedSchedule.is_enabled ? t('common.active') : t('common.disabled') }}
                    </span>
                  </div>
                </div>
              </n-tab-pane>

              <n-tab-pane name="members" :tab="t('schedule.tabMembers')">
                <ParticipantsList
                  ref="participantsRef"
                  :schedule-id="selectedSchedule.id"
                  :users="users"
                  :get-user-color="getUserColor"
                  :get-user-name="getUserName"
                />
              </n-tab-pane>
            </n-tabs>
          </div>
        </template>

        <!-- Empty state -->
        <div v-else class="empty-state">
          <div class="empty-eyebrow sre-label-eyebrow">{{ t('schedule.empty') || 'No schedule selected' }}</div>
          <p class="empty-text">{{ t('schedule.selectSchedule') }}</p>
          <n-button type="primary" size="small" @click="scheduleModalRef?.openCreate()">
            + {{ t('schedule.newSchedule') }}
          </n-button>
        </div>
      </section>
    </div>

    <!-- Modals -->
    <ScheduleModal
      ref="scheduleModalRef"
      :teams="teams"
      @saved="handleScheduleSaved"
    />

    <ShiftModal
      ref="shiftModalRef"
      :schedule-id="selectedSchedule?.id ?? null"
      :users="users"
      @saved="handleShiftSaved"
    />

    <!-- Generate Shifts Modal -->
    <n-modal v-model:show="showGenerateModal" preset="card" :title="t('schedule.generateShifts')" style="width: 420px" :bordered="false">
      <n-form label-placement="top">
        <n-form-item :label="t('schedule.weeksCount')">
          <n-input-number v-model:value="generateWeeks" :min="1" :max="12" style="width: 100%" />
        </n-form-item>
        <n-text depth="3" style="font-size: 12px">{{ t('schedule.generateHint') }}</n-text>
      </n-form>
      <template #action>
        <n-space justify="end">
          <n-button size="small" @click="showGenerateModal = false">{{ t('common.cancel') }}</n-button>
          <n-button size="small" type="primary" :loading="generating" @click="handleGenerateShifts">
            {{ t('schedule.confirmGenerate') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.schedule-page {
  font-family: var(--sre-font-sans);
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: calc(100vh - 88px);
}

/* Header */
.schedule-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.schedule-title {
  margin: 0;
  font-family: var(--sre-font-sans);
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--sre-text-primary);
  line-height: 1.2;
}

.schedule-subtitle {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--sre-text-secondary);
  line-height: 1.4;
}

.schedule-header-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

/* Layout */
.schedule-layout {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 16px;
  flex: 1;
  min-height: 0;
}

.schedule-sidebar-wrap {
  min-height: 0;
  overflow: hidden;
}

/* Detail panel */
.schedule-detail {
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md, 8px);
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.detail-topbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 20px 14px;
  border-bottom: var(--sre-hairline);
  flex-shrink: 0;
}

.detail-title-row {
  display: flex;
  align-items: center;
  gap: 24px;
  min-width: 0;
  flex: 1;
}

.detail-title-block {
  min-width: 0;
}

.detail-title {
  margin: 2px 0 0;
  font-family: var(--sre-font-sans);
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.005em;
  color: var(--sre-text-primary);
  line-height: 1.3;
  display: flex;
  align-items: center;
  gap: 0;
}

.detail-team {
  display: inline-flex;
  align-items: center;
  font-size: 13px;
  font-weight: 400;
  color: var(--sre-text-secondary);
}

.detail-actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}

/* On-call badge */
.oncall-badge {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 12px;
  background: var(--sre-bg-sunken, rgba(0, 0, 0, 0.02));
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-sm);
  min-width: 0;
}

.oncall-eyebrow {
  font-size: 10px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--sre-text-secondary);
}

.oncall-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  min-width: 0;
}

.oncall-name {
  font-weight: 500;
  color: var(--sre-text-primary);
  white-space: nowrap;
}

.oncall-dot-current {
  background: var(--dot-color, var(--sre-primary));
}

.oncall-email {
  font-size: 12px;
  color: var(--sre-text-secondary);
  font-family: var(--sre-font-mono);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Week nav */
.week-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 20px;
  border-bottom: var(--sre-hairline);
  flex-shrink: 0;
}

.week-nav-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.week-label {
  font-size: 13px;
  font-weight: 500;
  min-width: 160px;
  text-align: center;
  color: var(--sre-text-primary);
}

/* Calendar */
.calendar-container {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  position: relative;
}

.calendar-grid {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.cal-header-row {
  display: grid;
  grid-template-columns: 56px repeat(7, 1fr);
  border-bottom: var(--sre-hairline);
  flex-shrink: 0;
  background: var(--sre-bg-card);
}

.cal-time-gutter {
  border-right: var(--sre-hairline);
}

.cal-day-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 8px 0;
  font-size: 11px;
  color: var(--sre-text-secondary);
  border-right: var(--sre-hairline);
}

.cal-day-header.weekend {
  background: var(--sre-bg-sunken, rgba(0, 0, 0, 0.02));
}

.cal-day-header.today .cal-day-name {
  color: var(--sre-primary);
}

.cal-day-name {
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 10px;
  font-weight: 500;
}

.cal-day-num {
  font-size: 14px;
  font-weight: 500;
  line-height: 1;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--sre-text-primary);
}

.cal-day-num.today {
  background: var(--sre-primary);
  color: var(--sre-text-inverse);
}

.cal-body {
  display: grid;
  grid-template-columns: 56px repeat(7, 1fr);
  flex: 1;
  overflow-y: auto;
  min-height: 0;
  position: relative;
}

.cal-time-gutter-body {
  border-right: var(--sre-hairline);
  position: relative;
  height: 1200px;
  background: var(--sre-bg-card);
}

.cal-time-label {
  position: absolute;
  right: 8px;
  font-size: 10px;
  color: var(--sre-text-secondary);
  transform: translateY(-50%);
  font-family: var(--sre-font-mono);
}

.cal-day-col {
  position: relative;
  height: 1200px;
  border-right: var(--sre-hairline);
  cursor: pointer;
  transition: background var(--sre-duration-fast) var(--sre-ease-out);
}

.cal-day-col.weekend {
  background: var(--sre-bg-sunken, rgba(0, 0, 0, 0.02));
}

.cal-day-col.today {
  background: var(--sre-primary-soft);
}

.cal-day-col:hover {
  background: var(--sre-bg-sunken, rgba(0, 0, 0, 0.025));
}

.cal-hour-line {
  position: absolute;
  left: 0;
  right: 0;
  height: 1px;
  background: var(--sre-border);
  opacity: 0.4;
  pointer-events: none;
}

.cal-hour-line.major {
  opacity: 0.7;
}

.current-time-line {
  position: absolute;
  left: 0;
  right: 0;
  height: 1px;
  background: var(--sre-primary);
  z-index: 10;
  pointer-events: none;
}

.current-time-line::before {
  content: '';
  position: absolute;
  left: -3px;
  top: -2.5px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--sre-primary);
}

/* Shift blocks */
.shift-block {
  position: absolute;
  left: 4px;
  right: 4px;
  border-radius: 4px;
  padding: 4px 6px;
  font-size: 11px;
  line-height: 1.3;
  cursor: pointer;
  overflow: hidden;
  user-select: none;
  z-index: 1;
  box-sizing: border-box;
  background: color-mix(in srgb, var(--shift-color, var(--sre-primary)) 14%, transparent);
  color: var(--shift-color, var(--sre-primary));
  transition: filter var(--sre-duration-fast) var(--sre-ease-out);
}

.shift-block:hover {
  filter: brightness(1.05);
}

.shift-block.is-now {
  font-weight: 500;
  background: var(--shift-color);
  color: var(--sre-text-inverse);
  border-left-color: transparent;
  box-shadow: var(--sre-shadow-xs);
}

.shift-user {
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 11px;
}

.shift-time {
  font-size: 10px;
  opacity: 0.8;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-family: var(--sre-font-mono);
}

/* Config tabs */
.config-tabs {
  flex-shrink: 0;
  border-top: var(--sre-hairline);
  max-height: 240px;
  overflow-y: auto;
  padding: 0 20px 8px;
  background: var(--sre-bg-card);
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px 24px;
  padding: 12px 0 4px;
}

.config-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.config-label {
  font-size: 10px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--sre-text-secondary);
}

.config-value {
  font-size: 13px;
  color: var(--sre-text-primary);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.config-muted {
  color: var(--sre-text-secondary);
}

/* Dot modifier classes (used with global .sre-dot) */
.sre-dot--success {
  background: var(--sre-success);
}
.sre-dot--muted {
  background: var(--sre-text-secondary);
}

/* Empty state */
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 80px 24px;
}

.empty-eyebrow {
  color: var(--sre-text-secondary);
}

.empty-text {
  margin: 0;
  font-size: 14px;
  color: var(--sre-text-secondary);
}

/* Time label positions (every 2 hours, 1200px / 12 = 100px) */
.cal-time-gutter-body .cal-time-label:nth-child(1)  { top: 0px }
.cal-time-gutter-body .cal-time-label:nth-child(2)  { top: 100px }
.cal-time-gutter-body .cal-time-label:nth-child(3)  { top: 200px }
.cal-time-gutter-body .cal-time-label:nth-child(4)  { top: 300px }
.cal-time-gutter-body .cal-time-label:nth-child(5)  { top: 400px }
.cal-time-gutter-body .cal-time-label:nth-child(6)  { top: 500px }
.cal-time-gutter-body .cal-time-label:nth-child(7)  { top: 600px }
.cal-time-gutter-body .cal-time-label:nth-child(8)  { top: 700px }
.cal-time-gutter-body .cal-time-label:nth-child(9)  { top: 800px }
.cal-time-gutter-body .cal-time-label:nth-child(10) { top: 900px }
.cal-time-gutter-body .cal-time-label:nth-child(11) { top: 1000px }
.cal-time-gutter-body .cal-time-label:nth-child(12) { top: 1100px }
.cal-time-gutter-body .cal-time-label:nth-child(13) { top: 1200px }

/* Responsive */
@media (max-width: 960px) {
  .schedule-layout {
    grid-template-columns: 1fr;
  }
  .config-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
