<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NButton, NIcon, NSwitch, NInput, NFormItem, NSpin, NSelect, NTag, NSpace } from 'naive-ui'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { PulseOutline, SaveOutline } from '@vicons/ionicons5'
import { larkBotApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import { useConfigForm } from '@/composables'

const message = useMessage()
const { t } = useI18n()

const { form, loading, saving, load, saveAndTest } = useConfigForm({
  load: () => larkBotApi.getConfig().then(r => r.data.data),
  save: (f) => larkBotApi.updateConfig({ ...f }),
  test: () => larkBotApi.testBotAPI().then(res => {
    message.success(res.data.data?.message || t('settings.larkBotAPIOK'))
  }),
  autoSaveKeys: ['bot_enabled', 'update_on_state_change', 'delete_only_in_business_hours', 'commands_enabled', 'natural_language_enabled', 'debug_mode'],
})

const botStatusLoading = ref(false)
const botStatus = ref<{ configured: boolean; app_id: string; webhook_set: boolean; commands_enabled: boolean; natural_language_enabled: boolean; debug_mode: boolean } | null>(null)

const resolveOptions = [
  { label: () => t('settings.larkResolveUpdate'), value: 'update' },
  { label: () => t('settings.larkResolveDelete'), value: 'delete' },
]

async function fetchBotStatus() {
  botStatusLoading.value = true
  try {
    const res = await larkBotApi.getBotStatus()
    botStatus.value = res.data.data ?? null
  } catch {
    botStatus.value = null
  } finally {
    botStatusLoading.value = false
  }
}

onMounted(() => {
  load()
  fetchBotStatus()
})
</script>

<template>
  <NSpin :show="loading">
    <div class="sre-config-page">
      <header class="sre-config-header">
        <div>
          <h2 class="sre-config-header-title">{{ t('settings.larkBotTitle') }}</h2>
          <p class="sre-config-header-sub">{{ t('settings.larkBotSubtitle') }} <code>/lark/event</code></p>
        </div>
        <div class="sre-config-header-actions">
          <NButton type="primary" size="small" :loading="saving" @click="save">
            <template #icon><NIcon :component="SaveOutline" /></template>
            {{ t('common.save') }}
          </NButton>
        </div>
      </header>

      <div class="config-sections sre-stagger">
        <!-- Section 1: App Credentials -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.larkAppCredentials') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.larkAppCredentialsDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.larkBotEnabled')" class="full-row">
              <NSwitch v-model:value="form.bot_enabled" />
            </NFormItem>
            <NFormItem :label="t('settings.larkAppId')">
              <NInput v-model:value="form.app_id" :placeholder="t('settings.larkAppIdPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.larkAppSecret')">
              <NInput v-model:value="form.app_secret" type="password" show-password-on="click" :placeholder="t('settings.larkAppSecretPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.larkVerificationToken')">
              <NInput v-model:value="form.verification_token" :placeholder="t('settings.larkTokenPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.larkEncryptKey')">
              <NInput v-model:value="form.encrypt_key" type="password" show-password-on="click" :placeholder="t('settings.larkEncryptKeyPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.larkDefaultWebhook')" class="full-row">
              <NInput v-model:value="form.default_webhook" :placeholder="t('settings.larkWebhookPlaceholder')" />
            </NFormItem>
          </div>
        </section>

        <!-- Section 2: Behavior -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.larkBehavior') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.larkBehaviorDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.larkResolveStrategy')">
              <NSelect v-model:value="form.resolve_strategy" :options="resolveOptions" />
            </NFormItem>
            <NFormItem :label="t('settings.larkUpdateOnStateChange')">
              <div>
                <NSwitch v-model:value="form.update_on_state_change" />
                <p class="form-desc">{{ t('settings.larkUpdateOnStateChangeDesc') }}</p>
              </div>
            </NFormItem>
            <NFormItem :label="t('settings.larkDeleteBusinessHours')">
              <div>
                <NSwitch v-model:value="form.delete_only_in_business_hours" />
                <p class="form-desc">{{ t('settings.larkDeleteBusinessHoursDesc') }}</p>
              </div>
            </NFormItem>
            <NFormItem v-if="form.delete_only_in_business_hours" :label="t('settings.larkBusinessHoursStart')">
              <NInput v-model:value="form.business_hours_start" placeholder="09:00" style="width: 120px" />
            </NFormItem>
            <NFormItem v-if="form.delete_only_in_business_hours" :label="t('settings.larkBusinessHoursEnd')">
              <NInput v-model:value="form.business_hours_end" placeholder="18:00" style="width: 120px" />
            </NFormItem>
          </div>
        </section>

        <!-- Section 3: Commands -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.larkCommands') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.larkCommandsDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.larkCommandsEnabled')">
              <div>
                <NSwitch v-model:value="form.commands_enabled" />
                <p class="form-desc">{{ t('settings.larkCommandsEnabledDesc') }}</p>
              </div>
            </NFormItem>
            <NFormItem :label="t('settings.larkNLEnabled')">
              <div>
                <NSwitch v-model:value="form.natural_language_enabled" />
                <p class="form-desc">{{ t('settings.larkNLEnabledDesc') }}</p>
              </div>
            </NFormItem>
          </div>
        </section>

        <!-- Section 4: Debug -->
        <section class="sre-config-section">
          <div class="section-header-row">
            <div>
              <h3 class="sre-config-section-title" style="margin: 0">{{ t('settings.larkDebug') }}</h3>
              <p class="sre-config-section-desc" style="margin-top: 4px">{{ t('settings.larkDebugDesc') }}</p>
            </div>
            <NSpace :size="8">
              <NButton size="small" quaternary :loading="testing" @click="saveAndTest">
                <template #icon><NIcon :component="PulseOutline" /></template>
                {{ t('settings.larkTestBotAPI') }}
              </NButton>
              <NButton size="small" quaternary :loading="botStatusLoading" @click="fetchBotStatus">
                {{ t('settings.larkBotStatus') }}
              </NButton>
            </NSpace>
          </div>

          <div v-if="botStatus" class="bot-status-list">
            <div class="bot-status-row">
              <span class="bot-status-label">{{ t('settings.larkStatusConfigured') }}</span>
              <NTag :type="botStatus.configured ? 'success' : 'warning'" size="small" :bordered="false">
                {{ botStatus.configured ? t('common.yes') : t('common.no') }}
              </NTag>
            </div>
            <div class="bot-status-row">
              <span class="bot-status-label">{{ t('settings.larkStatusAppId') }}</span>
              <code class="bot-status-value">{{ botStatus.app_id || '-' }}</code>
            </div>
            <div class="bot-status-row">
              <span class="bot-status-label">{{ t('settings.larkStatusWebhook') }}</span>
              <NTag :type="botStatus.webhook_set ? 'success' : 'default'" size="small" :bordered="false">
                {{ botStatus.webhook_set ? t('settings.larkStatusSet') : t('settings.larkStatusNotSet') }}
              </NTag>
            </div>
            <div class="bot-status-row">
              <span class="bot-status-label">{{ t('settings.larkStatusCommands') }}</span>
              <NTag :type="botStatus.commands_enabled ? 'success' : 'default'" size="small" :bordered="false">
                {{ botStatus.commands_enabled ? t('common.enabled') : t('common.disabled') }}
              </NTag>
            </div>
            <div class="bot-status-row">
              <span class="bot-status-label">{{ t('settings.larkStatusNL') }}</span>
              <NTag :type="botStatus.natural_language_enabled ? 'success' : 'default'" size="small" :bordered="false">
                {{ botStatus.natural_language_enabled ? t('common.enabled') : t('common.disabled') }}
              </NTag>
            </div>
            <div class="bot-status-row">
              <span class="bot-status-label">{{ t('settings.larkStatusDebugMode') }}</span>
              <NTag :type="botStatus.debug_mode ? 'warning' : 'default'" size="small" :bordered="false">
                {{ botStatus.debug_mode ? t('common.on') : t('common.off') }}
              </NTag>
            </div>
          </div>
        </section>
      </div>
    </div>
  </NSpin>
</template>

<style scoped>
.section-header-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}
.bot-status-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.bot-status-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.bot-status-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  white-space: nowrap;
}
.bot-status-value {
  font-size: 13px;
  color: var(--sre-text-primary);
}
.form-desc {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin-top: 4px;
  line-height: 1.5;
}
</style>
