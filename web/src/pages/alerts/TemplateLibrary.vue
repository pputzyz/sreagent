<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTabs, NTabPane } from 'naive-ui'
import Presets from '@/pages/alerts/Presets.vue'
import RuleTemplates from '@/pages/alerts/RuleTemplates.vue'

const { t } = useI18n()
const activeTab = ref('presets')
</script>

<template>
  <div class="template-library-page">
    <n-tabs v-model:value="activeTab" type="line" animated>
      <n-tab-pane name="presets" :tab="t('templateLibrary.builtinPresets')">
        <Presets />
      </n-tab-pane>
      <n-tab-pane name="templates" :tab="t('templateLibrary.myTemplates')">
        <RuleTemplates />
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<style scoped>
.template-library-page {
  /* Inherit page width from child components */
}
</style>
