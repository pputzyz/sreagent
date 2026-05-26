<script setup lang="ts">
import { ref } from 'vue'
import { NTabs, NTabPane } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import OIDCConfig from '@/pages/settings/OIDCConfig.vue'
import LDAPConfig from '@/pages/settings/LDAPConfig.vue'
import OAuth2Config from '@/pages/settings/OAuth2Config.vue'

const { t } = useI18n()
const activeTab = ref('oidc')
</script>

<template>
  <NTabs v-model:value="activeTab" type="line" animated>
    <NTabPane name="oidc" :tab="t('settings.ssoTabOidc')">
      <OIDCConfig />
    </NTabPane>
    <NTabPane name="ldap" :tab="t('settings.ssoTabLdap')">
      <LDAPConfig />
    </NTabPane>
    <NTabPane name="oauth2" :tab="t('settings.ssoTabOauth2')">
      <OAuth2Config />
    </NTabPane>
  </NTabs>
</template>
