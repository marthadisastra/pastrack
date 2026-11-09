<script setup>
import { computed, ref } from 'vue'
import AppIcon from '../ui/AppIcon.vue'
import { makeWebManifest, downloadJson } from '../../modules/branding'
const draft = defineModel({ required: true })
const error = ref('')
const manifest = computed(() => makeWebManifest(draft.value.branding))
async function asset(event, key) {
  const file = event.target.files[0]
  event.target.value = ''
  error.value = ''
  if (!file) return
  if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) {
    error.value = 'Use a PNG, JPEG or WebP image.'
    return
  }
  if (file.size > 350 * 1024) {
    error.value = 'Image is too large for local preview storage. Maximum size is 350 KB.'
    return
  }
  try {
    const data = await new Promise((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => resolve(reader.result)
      reader.onerror = () => reject(new Error('Unable to read image.'))
      reader.readAsDataURL(file)
    })
    const image = new Image()
    image.src = data
    await image.decode()
    if (image.naturalWidth > 4096 || image.naturalHeight > 4096)
      throw new Error('Image dimensions must be 4096px or smaller.')
    draft.value.branding[key] = data
  } catch (cause) {
    error.value = cause.message || 'The selected image could not be decoded.'
  }
}
</script>
<template>
  <div class="space-y-3">
    <section class="panel p-3">
      <h2 class="flex items-center gap-2 text-sm font-semibold">
        <AppIcon name="verified_user" class="text-blue-600" />Application Identity & White-label
        Customization
      </h2>
      <p class="mt-1 text-[10px] text-slate-400">
        Configure fleet portal metadata and graphic brand assets. Domain routing requires server
        configuration.
      </p>
    </section>
    <p
      v-if="error"
      role="alert"
      class="rounded-md border border-rose-200 bg-rose-50 p-3 text-xs text-rose-700"
    >
      {{ error }}
    </p>
    <section class="panel">
      <div class="panel-title">
        <h3 class="flex items-center gap-2 text-xs font-semibold">
          <AppIcon name="settings_suggest" class="!text-base text-blue-600" />Application Metadata
        </h3>
        <span class="text-[9px] text-slate-400">Global Runtime Scope</span>
      </div>
      <div class="grid gap-3 p-3 sm:grid-cols-2 xl:grid-cols-3">
        <label class="field"
          >Application Name<input v-model="draft.branding.name" maxlength="40" required /></label
        ><label class="field"
          >Portal Subtitle / Tagline<input v-model="draft.branding.tagline" maxlength="60" /></label
        ><label class="field"
          >Company / Organization Name<input
            v-model="draft.branding.organization"
            maxlength="120" /></label
        ><label class="field"
          >System Version / Release Code<input
            value="v4.8.2-STABLE (Core)"
            readonly
            class="!bg-slate-50 text-slate-400" /></label
        ><label class="field"
          >Primary Brand Color
          <div class="flex items-center gap-2">
            <input v-model="draft.branding.color" type="color" class="h-8 w-10 !p-1" /><input
              v-model="draft.branding.color"
              maxlength="7"
              class="min-w-0 flex-1 font-mono"
            /></div></label
        ><label class="field"
          >Support / NOC Contact<input v-model="draft.branding.support" maxlength="120" /></label
        ><label class="field sm:col-span-2 xl:col-span-3"
          >Custom Domain FQDN<input
            v-model="draft.branding.domain"
            placeholder="telematics.example.com"
            maxlength="253"
          /><span class="text-[9px] text-slate-400"
            >Metadata only. Saving this field does not change DNS, TLS or hosting.</span
          ></label
        >
      </div>
    </section>
    <div class="grid gap-3 lg:grid-cols-3">
      <article
        v-for="assetInfo in [
          {
            key: 'logo',
            title: 'Primary Enterprise Logo',
            icon: 'image',
            description: 'Horizontal wordmark for dispatch consoles.',
          },
          {
            key: 'favicon',
            title: 'Favicon & PWA Icon',
            icon: 'tab',
            description: 'Browser tab icon; square images recommended.',
          },
          {
            key: 'touchIcon',
            title: 'Apple Touch Icon',
            icon: 'phone_iphone',
            description: 'Home screen tile preview for iOS / iPadOS.',
          },
        ]"
        :key="assetInfo.key"
        class="panel p-3"
      >
        <h3 class="flex items-center gap-2 text-xs font-semibold">
          <AppIcon :name="assetInfo.icon" class="!text-base text-blue-600" />{{ assetInfo.title }}
        </h3>
        <p class="mt-1 min-h-8 text-[10px] leading-4 text-slate-400">{{ assetInfo.description }}</p>
        <div
          class="my-3 flex h-20 items-center justify-center overflow-hidden rounded-lg border border-dashed border-slate-200 bg-slate-50"
        >
          <img
            v-if="draft.branding[assetInfo.key]"
            :src="draft.branding[assetInfo.key]"
            :alt="assetInfo.title"
            class="max-h-16 max-w-[90%] object-contain"
          /><span v-else class="font-bold" :style="{ color: draft.branding.color }">{{
            draft.branding.name
          }}</span>
        </div>
        <p class="mb-2 text-[9px] text-slate-400">PNG / JPEG / WebP · Maximum 350 KB</p>
        <label class="btn cursor-pointer"
          ><AppIcon name="upload" />Replace File<input
            type="file"
            accept="image/png,image/jpeg,image/webp"
            class="sr-only"
            :aria-label="`Replace ${assetInfo.title}`"
            @change="asset($event, assetInfo.key)" /></label
        ><button
          v-if="draft.branding[assetInfo.key]"
          class="btn ml-1 !text-rose-600"
          :aria-label="`Remove ${assetInfo.title}`"
          @click="draft.branding[assetInfo.key] = ''"
        >
          <AppIcon name="delete" />
        </button>
      </article>
    </div>
    <section class="panel">
      <div class="panel-title">
        <h3 class="flex items-center gap-2 text-xs font-semibold">
          <AppIcon name="data_object" class="!text-base text-blue-600" />Web App Manifest
        </h3>
        <button class="btn" @click="downloadJson('manifest.webmanifest', manifest)">
          <AppIcon name="download" />Download JSON
        </button>
      </div>
      <pre
        class="max-h-44 overflow-auto bg-slate-950 p-3 font-mono text-[10px] leading-5 text-emerald-300"
        >{{
          JSON.stringify(
            {
              ...manifest,
              icons: manifest.icons.map((icon) => ({ ...icon, src: '[selected local icon]' })),
            },
            null,
            2,
          )
        }}</pre>
      <p class="p-3 text-[10px] text-slate-400">
        Download generates a manifest preview. PWA hosting, service worker, icon resizing and cache
        re-indexing require deployment support.
      </p>
    </section>
  </div>
</template>
