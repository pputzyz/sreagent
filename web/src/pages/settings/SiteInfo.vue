<script setup lang="ts">
import { onMounted } from 'vue'
import {
  NButton,
  NIcon,
  NInput,
  NFormItem,
  NSpin,
  NImage,
} from 'naive-ui'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { SaveOutline } from '@vicons/ionicons5'
import { siteInfoApi } from '@/api'
import { useConfigForm } from '@/composables'

const message = useMessage()
const { t } = useI18n()

const { form, loading, saving, isDirty, load, save } = useConfigForm({
  load: () => siteInfoApi.get().then(r => r.data.data),
  save: (f) => siteInfoApi.save({ ...f }),
})

onMounted(() => load())
</script>

<template>
  <NSpin :show="loading">
    <div class="sre-config-page">
      <header class="sre-config-header">
        <div>
          <h2 class="sre-config-header-title">{{ t('settings.siteInfo') }}</h2>
          <p class="sre-config-header-sub">{{ t('settings.siteInfoDesc') }}</p>
        </div>
      </header>

      <div class="config-sections sre-stagger">
        <!-- Branding Section -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.siteName') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.siteInfoDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.siteName')" class="full-row">
              <NInput
                v-model:value="form.site_name"
                :placeholder="t('settings.siteNamePlaceholder')"
              />
            </NFormItem>
          </div>
        </section>

        <!-- Logo & Favicon Section -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.logoUrl') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.preview') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.logoUrl')" class="full-row">
              <div class="url-preview-row">
                <NInput
                  v-model:value="form.logo_url"
                  :placeholder="t('settings.logoUrlPlaceholder')"
                  style="flex: 1"
                />
                <div v-if="form.logo_url" class="preview-box">
                  <NImage
                    :src="form.logo_url"
                    :width="40"
                    :height="40"
                    object-fit="contain"
                    :fallback-src="''"
                    :preview-disabled="true"
                  >
                    <template #error>
                      <span class="preview-error">-</span>
                    </template>
                  </NImage>
                </div>
              </div>
            </NFormItem>
            <NFormItem :label="t('settings.faviconUrl')" class="full-row">
              <div class="url-preview-row">
                <NInput
                  v-model:value="form.favicon_url"
                  :placeholder="t('settings.faviconUrlPlaceholder')"
                  style="flex: 1"
                />
                <div v-if="form.favicon_url" class="preview-box favicon-preview">
                  <NImage
                    :src="form.favicon_url"
                    :width="16"
                    :height="16"
                    object-fit="contain"
                    :fallback-src="''"
                    :preview-disabled="true"
                  >
                    <template #error>
                      <span class="preview-error">-</span>
                    </template>
                  </NImage>
                </div>
              </div>
            </NFormItem>
          </div>
        </section>

        <!-- Login Page Section -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.loginTitle') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.siteInfoDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.loginTitle')">
              <NInput
                v-model:value="form.login_title"
                :placeholder="t('settings.loginTitlePlaceholder')"
              />
            </NFormItem>
            <NFormItem :label="t('settings.loginSubtitle')">
              <NInput
                v-model:value="form.login_subtitle"
                :placeholder="t('settings.loginSubtitlePlaceholder')"
              />
            </NFormItem>
          </div>
        </section>

        <!-- Footer & CSS Section -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.footerText') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.siteInfoDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.footerText')" class="full-row">
              <NInput
                v-model:value="form.footer_text"
                :placeholder="t('settings.footerTextPlaceholder')"
              />
            </NFormItem>
            <NFormItem :label="t('settings.customCss')" class="full-row">
              <NInput
                v-model:value="form.custom_css"
                type="textarea"
                :rows="8"
                :placeholder="t('settings.customCssPlaceholder')"
                font-family="monospace"
              />
            </NFormItem>
          </div>
        </section>
      </div>

      <!-- Global Save -->
      <div class="global-footer">
        <NButton
          type="primary"
          :loading="saving"
          :disabled="!isDirty"
          @click="save"
        >
          <template #icon><NIcon :component="SaveOutline" /></template>
          {{ t('common.save') }}
        </NButton>
      </div>
    </div>
  </NSpin>
</template>

<style scoped>
.url-preview-row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.preview-box {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--sre-bg-elevated);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-sm);
  flex-shrink: 0;
}

.favicon-preview {
  width: 32px;
  height: 32px;
}

.preview-error {
  font-size: 12px;
  color: var(--sre-text-tertiary);
}

.global-footer {
  display: flex;
  justify-content: flex-end;
  padding: 20px 0;
  margin-top: 8px;
  border-top: var(--sre-hairline);
}
</style>
