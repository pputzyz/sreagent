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

const { form, loading, testing, load, saveAndTest, markSaved } = useConfigForm({
  load: () => larkBotApi.getConfig().then(r => r.data.data),
  save: (f) => larkBotApi.updateConfig({ ...f }),
  test: () => larkBotApi.testBotAPI().then(res => {
    message.success(res.data.data?.message || t('settings.larkBotAPIOK'))
  }),
})

// Per-section saving state
const savingCredentials = ref(false)
const savingBehavior = ref(false)
const savingCommands = ref(false)

// Bot status
const botStatusLoading = ref(false)
const botStatus = ref<{ configured: boolean; app_id: string; webhook_set: boolean; commands_enabled: boolean; natural_language_enabled: boolean; debug_mode: boolean; connection_mode?: string; event_source_status?: string } | null>(null)

const resolveOptions = [
  { label: () => t('settings.larkResolveUpdate'), value: 'update' },
  { label: () => t('settings.larkResolveDelete'), value: 'delete' },
]

async function saveSection(section: 'credentials' | 'behavior' | 'commands') {
  // Validate: if bot_enabled, require app_id and app_secret
  if (section === 'credentials' && form.bot_enabled) {
    if (!form.app_id?.trim()) {
      message.warning(t('settings.larkAppIdRequired'))
      return
    }
    if (!form.app_secret?.trim()) {
      message.warning(t('settings.larkAppSecretRequired'))
      return
    }
  }
  const savingRef = section === 'credentials' ? savingCredentials : section === 'behavior' ? savingBehavior : savingCommands
  savingRef.value = true
  try {
    await larkBotApi.updateConfig({ ...form })
    markSaved()
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    savingRef.value = false
  }
}

async function fetchBotStatus() {
  botStatusLoading.value = true
  try {
    const res = await larkBotApi.getBotStatus()
    botStatus.value = res.data.data ?? null
  } catch (err: unknown) {
    botStatus.value = null
    message.error(getErrorMessage(err))
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
            <NFormItem :label="t('settings.larkAppId')" :required="form.bot_enabled">
              <NInput v-model:value="form.app_id" :placeholder="t('settings.larkAppIdPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.larkAppSecret')" :required="form.bot_enabled">
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
          <div class="section-footer">
            <NButton type="primary" size="small" :loading="savingCredentials" @click="saveSection('credentials')">
              <template #icon><NIcon :component="SaveOutline" /></template>
              {{ t('common.save') }}
            </NButton>
          </div>
        </section>

        <!-- Section 1.5: Connection & Card -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.larkConnection') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.larkConnectionDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.larkDomain')">
              <NSelect v-model:value="form.domain" :options="[
                { label: 'Lark (International)', value: 'larksuite' },
                { label: 'Feishu (China)', value: 'feishu' },
              ]" />
            </NFormItem>
            <NFormItem :label="t('settings.larkConnectionMode')">
              <NSelect v-model:value="form.connection_mode" :options="[
                { label: 'WebSocket', value: 'websocket' },
                { label: 'HTTP Callback', value: 'http_callback' },
              ]" />
            </NFormItem>
            <NFormItem :label="t('settings.larkCardInteraction')">
              <NSelect v-model:value="form.card_interaction_mode" :options="[
                { label: 'Open URL (no callback)', value: 'open_url' },
                { label: 'Callback (HTTP)', value: 'callback_http' },
                { label: 'Callback (WS)', value: 'callback_ws' },
              ]" />
            </NFormItem>
            <NFormItem :label="t('settings.larkCardSchema')">
              <NSelect v-model:value="form.card_schema_version" :options="[
                { label: 'Card 2.0 + CardKit', value: 'v2' },
                { label: 'Card 1.0 (Legacy)', value: 'v1' },
              ]" />
            </NFormItem>
          </div>
          <div class="section-footer">
            <NButton type="primary" size="small" :loading="savingCredentials" @click="saveSection('credentials')">
              <template #icon><NIcon :component="SaveOutline" /></template>
              {{ t('common.save') }}
            </NButton>
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
          <div class="section-footer">
            <NButton type="primary" size="small" :loading="savingBehavior" @click="saveSection('behavior')">
              <template #icon><NIcon :component="SaveOutline" /></template>
              {{ t('common.save') }}
            </NButton>
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
            <NFormItem :label="t('settings.larkBotTools')" class="full-row">
              <div style="width: 100%">
                <NInput v-model:value="form.bot_allowed_tools" :placeholder="t('settings.larkBotToolsPlaceholder')" />
                <p class="form-desc">{{ t('settings.larkBotToolsDesc') }}</p>
              </div>
            </NFormItem>
          </div>
          <div class="section-footer">
            <NButton type="primary" size="small" :loading="savingCommands" @click="saveSection('commands')">
              <template #icon><NIcon :component="SaveOutline" /></template>
              {{ t('common.save') }}
            </NButton>
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
            <div v-if="botStatus.connection_mode === 'websocket'" class="bot-status-row">
              <span class="bot-status-label">{{ t('settings.larkStatusWS') }}</span>
              <NTag :type="botStatus.event_source_status === 'connected' ? 'success' : botStatus.event_source_status === 'reconnecting' ? 'warning' : 'error'" size="small" :bordered="false">
                {{ botStatus.event_source_status || t('settings.larkStatusWSNotStarted') }}
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
.section-footer {
  display: flex; justify-content: flex-end;
  padding-top: 16px; margin-top: 16px;
  border-top: var(--sre-hairline);
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
