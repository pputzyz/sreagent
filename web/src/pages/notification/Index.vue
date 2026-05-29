<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Rules from './Rules.vue'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
</script>

<template>
  <div>
    <PageHeader :title="t('menu.notifyPolicies')" :subtitle="t('notification.subtitle')" />
    <Rules />
  </div>
</template>
