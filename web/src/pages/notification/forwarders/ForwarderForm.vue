<template>
  <n-form
    ref="formRef"
    :model="formData"
    :rules="rules"
    label-placement="left"
    label-width="160"
    require-mark-placement="right-hanging"
    style="padding: 20px"
  >
    <n-tabs v-model:value="activeTab" type="line">
      <!-- Basic Tab -->
      <n-tab-pane name="basic" :tab="t('forwarder.basicSettings')">
        <n-form-item :label="t('common.name')" path="name">
          <n-input v-model:value="formData.name" :placeholder="t('forwarder.namePlaceholder')" />
        </n-form-item>

        <n-form-item :label="t('common.description')" path="description">
          <n-input
            v-model:value="formData.description"
            type="textarea"
            :placeholder="t('forwarder.descriptionPlaceholder')"
            :rows="2"
          />
        </n-form-item>

        <n-form-item :label="t('forwarder.direction')" path="direction">
          <n-radio-group v-model:value="formData.direction">
            <n-space>
              <n-radio value="inbound">{{ t('forwarder.directionInbound') }}</n-radio>
              <n-radio value="outbound">{{ t('forwarder.directionOutbound') }}</n-radio>
              <n-radio value="bidirectional">{{ t('forwarder.directionBidirectional') }}</n-radio>
            </n-space>
          </n-radio-group>
        </n-form-item>

        <n-form-item :label="t('forwarder.priority')" path="priority">
          <n-input-number v-model:value="formData.priority" :min="0" :max="100" />
        </n-form-item>

        <n-form-item :label="t('forwarder.enabled')" path="enabled">
          <n-switch v-model:value="formData.enabled" />
        </n-form-item>

        <n-form-item :label="t('forwarder.matchLabels')" path="match_labels">
          <n-dynamic-tags v-model:value="matchLabelsList" @update:value="updateMatchLabels" />
        </n-form-item>
      </n-tab-pane>

      <!-- Inbound Tab -->
      <n-tab-pane
        v-if="formData.direction === 'inbound' || formData.direction === 'bidirectional'"
        name="inbound"
        :tab="t('forwarder.inboundConfig')"
      >
        <n-form-item :label="t('forwarder.sourceFormat')" path="inbound_config.source_format">
          <n-select
            v-model:value="formData.inbound_config.source_format"
            :options="sourceFormatOptions"
          />
        </n-form-item>

        <n-form-item :label="t('forwarder.inboundMode')" path="inbound_config.mode">
          <n-radio-group v-model:value="formData.inbound_config.mode">
            <n-space>
              <n-radio value="integrate">
                <n-tooltip trigger="hover">
                  <template #trigger>{{ t('forwarder.modeIntegrate') }}</template>
                  {{ t('forwarder.modeIntegrateDesc') }}
                </n-tooltip>
              </n-radio>
              <n-radio value="proxy">
                <n-tooltip trigger="hover">
                  <template #trigger>{{ t('forwarder.modeProxy') }}</template>
                  {{ t('forwarder.modeProxyDesc') }}
                </n-tooltip>
              </n-radio>
            </n-space>
          </n-radio-group>
        </n-form-item>

        <n-form-item :label="t('forwarder.authType')" path="inbound_config.auth_type">
          <n-select
            v-model:value="formData.inbound_config.auth_type"
            :options="authTypeOptions"
          />
        </n-form-item>

        <!-- Bearer Auth -->
        <template v-if="formData.inbound_config.auth_type === 'bearer'">
          <n-form-item :label="t('forwarder.authToken')" path="inbound_config.auth_config.token">
            <n-input
              v-model:value="formData.inbound_config.auth_config.token"
              type="password"
              show-password-on="click"
              :placeholder="t('forwarder.tokenPlaceholder')"
            />
          </n-form-item>
        </template>

        <!-- Basic Auth -->
        <template v-if="formData.inbound_config.auth_type === 'basic'">
          <n-form-item :label="t('forwarder.authUsername')" path="inbound_config.auth_config.username">
            <n-input
              v-model:value="formData.inbound_config.auth_config.username"
              :placeholder="t('forwarder.usernamePlaceholder')"
            />
          </n-form-item>
          <n-form-item :label="t('forwarder.authPassword')" path="inbound_config.auth_config.password">
            <n-input
              v-model:value="formData.inbound_config.auth_config.password"
              type="password"
              show-password-on="click"
              :placeholder="t('forwarder.passwordPlaceholder')"
            />
          </n-form-item>
        </template>

        <!-- HMAC Auth -->
        <template v-if="formData.inbound_config.auth_type === 'hmac'">
          <n-form-item :label="t('forwarder.hmacSecret')" path="inbound_config.auth_config.hmac_secret">
            <n-input
              v-model:value="formData.inbound_config.auth_config.hmac_secret"
              type="password"
              show-password-on="click"
              :placeholder="t('forwarder.hmacSecretPlaceholder')"
            />
          </n-form-item>
          <n-form-item :label="t('forwarder.hmacHeader')" path="inbound_config.auth_config.hmac_header">
            <n-input
              v-model:value="formData.inbound_config.auth_config.hmac_header"
              :placeholder="t('forwarder.hmacHeaderPlaceholder')"
            />
          </n-form-item>
          <n-form-item :label="t('forwarder.hmacAlgorithm')" path="inbound_config.auth_config.hmac_algorithm">
            <n-select
              v-model:value="formData.inbound_config.auth_config.hmac_algorithm"
              :options="hmacAlgorithmOptions"
            />
          </n-form-item>
        </template>

        <!-- Inbound Severity Mapping -->
        <n-divider>{{ t('forwarder.inboundSeverityMapping') }}</n-divider>
        <n-form-item :label="t('forwarder.enableSeverityMapping')">
          <n-switch v-model:value="formData.inbound_severity_mapping.enabled" />
        </n-form-item>
        <template v-if="formData.inbound_severity_mapping.enabled">
          <n-form-item :label="t('forwarder.severityMap')">
            <n-dynamic-input
              v-model:value="inboundSeverityMappingList"
              :on-create="createSeverityMapping"
            >
              <template #default="{ value }">
                <n-space>
                  <n-input v-model:value="value.source" :placeholder="t('forwarder.sourceSeverity')" style="width: 150px" />
                  <n-text>→</n-text>
                  <n-input v-model:value="value.target" :placeholder="t('forwarder.targetSeverity')" style="width: 150px" />
                </n-space>
              </template>
            </n-dynamic-input>
          </n-form-item>
          <n-form-item :label="t('forwarder.defaultSeverity')">
            <n-input v-model:value="formData.inbound_severity_mapping.default_severity" :placeholder="t('forwarder.defaultSeverityPlaceholder')" />
          </n-form-item>
        </template>

        <!-- Proxy Target (only for proxy mode) -->
        <template v-if="formData.inbound_config.mode === 'proxy'">
          <n-divider>{{ t('forwarder.proxyTarget') }}</n-divider>
          <n-form-item :label="t('forwarder.targetType')">
            <n-radio-group v-model:value="proxyTargetType">
              <n-space>
                <n-radio value="media">{{ t('forwarder.targetMedia') }}</n-radio>
                <n-radio value="url">{{ t('forwarder.targetURL') }}</n-radio>
              </n-space>
            </n-radio-group>
          </n-form-item>
          <n-form-item v-if="proxyTargetType === 'media'" :label="t('forwarder.targetMedia')">
            <n-select
              v-model:value="formData.inbound_config.proxy_target.target_media_id"
              :options="mediaOptions"
              :loading="loadingMedia"
              :placeholder="t('forwarder.selectMedia')"
              filterable
            />
          </n-form-item>
          <n-form-item v-if="proxyTargetType === 'url'" :label="t('forwarder.targetURL')">
            <n-input v-model:value="formData.inbound_config.proxy_target.target_url" :placeholder="t('forwarder.urlPlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('forwarder.httpMethod')">
            <n-select v-model:value="formData.inbound_config.proxy_target.method" :options="httpMethodOptions" />
          </n-form-item>
          <n-form-item :label="t('forwarder.bodyTemplate')">
            <n-input v-model:value="formData.inbound_config.proxy_target.body_template" type="textarea" :rows="4" :placeholder="t('forwarder.bodyTemplatePlaceholder')" />
          </n-form-item>
        </template>

        <!-- Inbound Endpoint Info -->
        <n-alert v-if="id" type="info" style="margin-top: 16px">
          <template #header>{{ t('forwarder.inboundEndpoint') }}</template>
          <n-text code>{{ inboundEndpoint }}</n-text>
        </n-alert>
      </n-tab-pane>

      <!-- Outbound Tab -->
      <n-tab-pane
        v-if="formData.direction === 'outbound' || formData.direction === 'bidirectional'"
        name="outbound"
        :tab="t('forwarder.outboundConfig')"
      >
        <n-form-item :label="t('forwarder.targetType')" path="outbound_target_type">
          <n-radio-group v-model:value="outboundTargetType">
            <n-space>
              <n-radio value="media">{{ t('forwarder.targetMedia') }}</n-radio>
              <n-radio value="url">{{ t('forwarder.targetURL') }}</n-radio>
            </n-space>
          </n-radio-group>
        </n-form-item>

        <n-form-item v-if="outboundTargetType === 'media'" :label="t('forwarder.targetMedia')" path="outbound_config.target_media_id">
          <n-select
            v-model:value="formData.outbound_config.target_media_id"
            :options="mediaOptions"
            :loading="loadingMedia"
            :placeholder="t('forwarder.selectMedia')"
            filterable
          />
        </n-form-item>

        <n-form-item v-if="outboundTargetType === 'url'" :label="t('forwarder.targetURL')" path="outbound_config.target_url">
          <n-input
            v-model:value="formData.outbound_config.target_url"
            :placeholder="t('forwarder.urlPlaceholder')"
          />
        </n-form-item>

        <n-form-item :label="t('forwarder.httpMethod')" path="outbound_config.method">
          <n-select
            v-model:value="formData.outbound_config.method"
            :options="httpMethodOptions"
          />
        </n-form-item>

        <n-form-item :label="t('forwarder.bodyTemplate')" path="outbound_config.body_template">
          <n-input
            v-model:value="formData.outbound_config.body_template"
            type="textarea"
            :rows="6"
            :placeholder="t('forwarder.bodyTemplatePlaceholder')"
          />
        </n-form-item>

        <n-form-item :label="t('forwarder.timeout')" path="outbound_config.timeout">
          <n-input-number v-model:value="formData.outbound_config.timeout" :min="1000" :max="60000" :step="1000">
            <template #suffix>ms</template>
          </n-input-number>
        </n-form-item>

        <n-form-item :label="t('forwarder.retryTimes')" path="outbound_config.retry_times">
          <n-input-number v-model:value="formData.outbound_config.retry_times" :min="0" :max="10" />
        </n-form-item>

        <!-- Outbound Severity Mapping -->
        <n-divider>{{ t('forwarder.outboundSeverityMapping') }}</n-divider>
        <n-form-item :label="t('forwarder.enableSeverityMapping')">
          <n-switch v-model:value="formData.outbound_severity_mapping.enabled" />
        </n-form-item>
        <template v-if="formData.outbound_severity_mapping.enabled">
          <n-form-item :label="t('forwarder.severityMap')">
            <n-dynamic-input
              v-model:value="outboundSeverityMappingList"
              :on-create="createSeverityMapping"
            >
              <template #default="{ value }">
                <n-space>
                  <n-input v-model:value="value.source" :placeholder="t('forwarder.sourceSeverity')" style="width: 150px" />
                  <n-text>→</n-text>
                  <n-input v-model:value="value.target" :placeholder="t('forwarder.targetSeverity')" style="width: 150px" />
                </n-space>
              </template>
            </n-dynamic-input>
          </n-form-item>
          <n-form-item :label="t('forwarder.defaultSeverity')">
            <n-input v-model:value="formData.outbound_severity_mapping.default_severity" :placeholder="t('forwarder.defaultSeverityPlaceholder')" />
          </n-form-item>
        </template>
      </n-tab-pane>

      <!-- Platform Capabilities Tab (only for integrate mode) -->
      <n-tab-pane
        v-if="showPlatformCapabilities"
        name="capabilities"
        :tab="t('forwarder.platformCapabilities')"
      >
        <n-alert type="info" style="margin-bottom: 16px">
          {{ t('forwarder.capabilitiesIntegrateOnly') }}
        </n-alert>
        <n-form-item :label="t('forwarder.capNotification')" path="platform_capabilities.enable_notification">
          <n-switch v-model:value="formData.platform_capabilities.enable_notification" />
          <n-text depth="3" style="margin-left: 12px">{{ t('forwarder.capNotificationDesc') }}</n-text>
        </n-form-item>

        <n-form-item :label="t('forwarder.capEscalation')" path="platform_capabilities.enable_escalation">
          <n-switch v-model:value="formData.platform_capabilities.enable_escalation" />
          <n-text depth="3" style="margin-left: 12px">{{ t('forwarder.capEscalationDesc') }}</n-text>
        </n-form-item>

        <n-form-item :label="t('forwarder.capMute')" path="platform_capabilities.enable_mute">
          <n-switch v-model:value="formData.platform_capabilities.enable_mute" />
          <n-text depth="3" style="margin-left: 12px">{{ t('forwarder.capMuteDesc') }}</n-text>
        </n-form-item>

        <n-form-item :label="t('forwarder.capInhibition')" path="platform_capabilities.enable_inhibition">
          <n-switch v-model:value="formData.platform_capabilities.enable_inhibition" />
          <n-text depth="3" style="margin-left: 12px">{{ t('forwarder.capInhibitionDesc') }}</n-text>
        </n-form-item>

        <n-form-item :label="t('forwarder.capAI')" path="platform_capabilities.enable_ai_analysis">
          <n-switch v-model:value="formData.platform_capabilities.enable_ai_analysis" />
          <n-text depth="3" style="margin-left: 12px">{{ t('forwarder.capAIDesc') }}</n-text>
        </n-form-item>

        <n-form-item :label="t('forwarder.pipelineID')" path="platform_capabilities.pipeline_id">
          <n-input-number
            v-model:value="formData.platform_capabilities.pipeline_id"
            :min="0"
            :placeholder="t('forwarder.pipelineIDPlaceholder')"
            clearable
          />
        </n-form-item>
      </n-tab-pane>
    </n-tabs>

    <n-space justify="end" style="margin-top: 24px">
      <n-button @click="$emit('cancel')">{{ t('common.cancel') }}</n-button>
      <n-button type="primary" :loading="submitting" @click="handleSubmit">
        {{ t('common.save') }}
      </n-button>
    </n-space>
  </n-form>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NForm, NFormItem, NInput, NInputNumber, NSelect, NSwitch, NRadioGroup, NRadio,
  NSpace, NButton, NText, NAlert, NTabs, NTabPane, NDynamicTags, NDynamicInput,
  NDivider, NTooltip, useMessage
} from 'naive-ui'
import type { FormInst, FormRules } from 'naive-ui'
import {
  getAlertForwarder, createAlertForwarder, updateAlertForwarder
} from '@/api/alert-forwarder'
import type {
  AlertForwarder, InboundConfig, OutboundConfig,
  SeverityMappingConfig, PlatformCapabilitiesConfig
} from '@/api/alert-forwarder'
import { notifyMediaApi } from '@/api/notify'

// Form-specific type where nested objects are always defined
interface ForwarderFormData extends Omit<Required<Pick<AlertForwarder,
  'name' | 'description' | 'enabled' | 'direction' | 'priority' | 'match_labels'
>>, 'match_labels'> {
  match_labels: Record<string, string>
  inbound_config: InboundConfig & { auth_config: NonNullable<InboundConfig['auth_config']>; proxy_target: OutboundConfig }
  outbound_config: OutboundConfig
  inbound_severity_mapping: SeverityMappingConfig
  outbound_severity_mapping: SeverityMappingConfig
  platform_capabilities: PlatformCapabilitiesConfig
}

const props = defineProps<{
  id?: number | null
}>()

const emit = defineEmits<{
  success: []
  cancel: []
}>()

const { t } = useI18n()
const message = useMessage()
const formRef = ref<FormInst | null>(null)
const submitting = ref(false)
const loadingMedia = ref(false)
const activeTab = ref('basic')
const mediaOptions = ref<{ label: string; value: number }[]>([])

// Outbound target type
const outboundTargetType = ref<'media' | 'url'>('media')
const proxyTargetType = ref<'media' | 'url'>('url')

// Match labels as list for dynamic tags
const matchLabelsList = ref<string[]>([])

// Severity mapping lists for dynamic input
const inboundSeverityMappingList = ref<{ source: string; target: string }[]>([])
const outboundSeverityMappingList = ref<{ source: string; target: string }[]>([])

// Form data with defaults
const formData = reactive<ForwarderFormData>({
  name: '',
  description: '',
  enabled: true,
  direction: 'inbound',
  priority: 0,
  match_labels: {},
  inbound_config: {
    source_format: 'alertmanager',
    mode: 'integrate',
    auth_type: 'none',
    auth_config: {
      token: '',
      username: '',
      password: '',
      hmac_secret: '',
      hmac_header: 'X-Signature',
      hmac_algorithm: 'sha256'
    },
    proxy_target: {
      target_url: '',
      method: 'POST',
      headers: {},
      body_template: '',
      timeout: 30000,
      retry_times: 3,
      retry_interval: 100
    }
  },
  outbound_config: {
    target_media_id: undefined,
    target_url: '',
    method: 'POST',
    headers: {},
    body_template: '',
    timeout: 30000,
    retry_times: 3,
    retry_interval: 100
  },
  inbound_severity_mapping: {
    enabled: false,
    mapping: {},
    default_severity: ''
  },
  outbound_severity_mapping: {
    enabled: false,
    mapping: {},
    default_severity: ''
  },
  platform_capabilities: {
    enable_escalation: false,
    enable_mute: false,
    enable_inhibition: false,
    enable_notification: true,
    enable_ai_analysis: false,
    pipeline_id: undefined
  }
})

// Computed: show platform capabilities tab only for integrate mode
const showPlatformCapabilities = computed(() => {
  if (formData.direction === 'outbound') return false
  return formData.inbound_config?.mode === 'integrate'
})

// Options
const sourceFormatOptions = computed(() => [
  { label: 'Alertmanager', value: 'alertmanager' },
  { label: 'Grafana', value: 'grafana' },
  { label: 'Prometheus', value: 'prometheus' },
  { label: t('forwarder.generic'), value: 'generic' }
])

const authTypeOptions = computed(() => [
  { label: t('forwarder.authNone'), value: 'none' },
  { label: 'Bearer Token', value: 'bearer' },
  { label: 'Basic Auth', value: 'basic' },
  { label: 'HMAC Signature', value: 'hmac' }
])

const hmacAlgorithmOptions = computed(() => [
  { label: 'SHA-256', value: 'sha256' },
  { label: 'SHA-1', value: 'sha1' }
])

const httpMethodOptions = computed(() => [
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'PATCH', value: 'PATCH' }
])

// Rules
const rules: FormRules = {
  name: {
    required: true,
    message: () => t('forwarder.nameRequired'),
    trigger: 'blur'
  },
  direction: {
    required: true,
    message: () => t('forwarder.directionRequired'),
    trigger: 'change'
  }
}

// Computed
const inboundEndpoint = computed(() => {
  if (props.id) {
    return `${window.location.origin}/api/v1/alert-forwarders/${props.id}/inbound`
  }
  return ''
})

// Methods
function createSeverityMapping() {
  return { source: '', target: '' }
}

function updateMatchLabels(labels: string[]) {
  const result: Record<string, string> = {}
  labels.forEach(label => {
    const [key, value] = label.split('=')
    if (key && value) {
      result[key] = value
    }
  })
  formData.match_labels = result
}

// Convert severity mapping map to list and back
function mappingToList(mapping?: Record<string, string>): { source: string; target: string }[] {
  if (!mapping) return []
  return Object.entries(mapping).map(([source, target]) => ({ source, target }))
}

function listToMapping(list: { source: string; target: string }[]): Record<string, string> {
  const result: Record<string, string> = {}
  list.forEach(({ source, target }) => {
    if (source && target) result[source] = target
  })
  return result
}

async function loadMediaOptions() {
  loadingMedia.value = true
  try {
    const res = await notifyMediaApi.list({ page: 1, page_size: 100 })
    mediaOptions.value = (res.data.data?.list || []).map((m: any) => ({
      label: `${m.name} (${m.type})`,
      value: m.id
    }))
  } catch (error) {
    // Ignore
  } finally {
    loadingMedia.value = false
  }
}

async function loadForwarder() {
  if (!props.id) return

  try {
    const res = await getAlertForwarder(props.id)
    const data = res.data.data
    if (data) {
      Object.assign(formData, data)

      // Convert match_labels to list
      if (data.match_labels) {
        matchLabelsList.value = Object.entries(data.match_labels).map(
          ([k, v]) => `${k}=${v}`
        )
      }

      // Convert severity mappings to lists
      inboundSeverityMappingList.value = mappingToList(data.inbound_severity_mapping?.mapping)
      outboundSeverityMappingList.value = mappingToList(data.outbound_severity_mapping?.mapping)

      // Set outbound target type
      if (data.outbound_config?.target_media_id) {
        outboundTargetType.value = 'media'
      } else if (data.outbound_config?.target_url) {
        outboundTargetType.value = 'url'
      }

      // Set proxy target type
      if (data.inbound_config?.proxy_target?.target_media_id) {
        proxyTargetType.value = 'media'
      } else {
        proxyTargetType.value = 'url'
      }
    }
  } catch (error: any) {
    message.error(error.message || t('common.error'))
  }
}

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  // Convert severity mapping lists to maps
  if (formData.inbound_severity_mapping?.enabled) {
    formData.inbound_severity_mapping.mapping = listToMapping(inboundSeverityMappingList.value)
  }
  if (formData.outbound_severity_mapping?.enabled) {
    formData.outbound_severity_mapping.mapping = listToMapping(outboundSeverityMappingList.value)
  }

  // Clear unused outbound config fields
  if (outboundTargetType.value === 'media') {
    formData.outbound_config!.target_url = ''
  } else {
    formData.outbound_config!.target_media_id = undefined
  }

  // Clear proxy target if not proxy mode
  if (formData.inbound_config?.mode !== 'proxy') {
    delete (formData.inbound_config as any).proxy_target
  }

  // Clear platform capabilities if not integrate mode
  if (formData.inbound_config?.mode !== 'integrate' && formData.direction === 'inbound') {
    delete (formData as any).platform_capabilities
  }

  submitting.value = true
  try {
    const payload = { ...formData } as Partial<AlertForwarder>
    if (props.id) {
      await updateAlertForwarder(props.id, payload)
      message.success(t('common.updateSuccess'))
    } else {
      await createAlertForwarder(payload)
      message.success(t('common.createSuccess'))
    }
    emit('success')
  } catch (error: any) {
    message.error(error.message || t('common.error'))
  } finally {
    submitting.value = false
  }
}

// Watch direction changes
watch(() => formData.direction, (val) => {
  if (val === 'inbound') {
    activeTab.value = 'inbound'
  } else if (val === 'outbound') {
    activeTab.value = 'outbound'
  }
})

// Init
onMounted(() => {
  loadMediaOptions()
  if (props.id) {
    loadForwarder()
  }
})
</script>
