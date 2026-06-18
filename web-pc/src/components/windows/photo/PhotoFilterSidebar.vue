<script setup lang="ts">
import { Album, Sparkles } from 'lucide-vue-next';

type DimensionKey = 'timeline' | 'people' | 'places' | 'devices' | 'albums';

defineProps<{
  loading: boolean;
  dimensionOptions: Array<{ key: DimensionKey; label: string; icon: typeof Album }>;
  activeDimension: DimensionKey;
  facets: string[];
  selectedFacet: string;
}>();

const emit = defineEmits<{
  (e: 'select-dimension', key: DimensionKey): void;
  (e: 'select-facet', facet: string): void;
}>();
</script>

<template>
  <aside class="photo-media__sidebar" aria-label="媒体筛选">
    <div class="photo-media__section-title">
      <Sparkles :size="15" />
      {{ loading ? '同步媒体' : '相册媒体' }}
    </div>

    <nav class="photo-media__dimensions" aria-label="筛选维度">
      <button
        v-for="dimension in dimensionOptions"
        :key="dimension.key"
        class="photo-media__dimension"
        :class="{ 'photo-media__dimension--active': activeDimension === dimension.key }"
        type="button"
        @click="emit('select-dimension', dimension.key)"
      >
        <component :is="dimension.icon" :size="15" />
        <span>{{ dimension.label }}</span>
      </button>
    </nav>

    <div class="photo-media__facets" aria-label="筛选值">
      <button
        v-for="facet in facets"
        :key="facet"
        class="photo-media__facet"
        :class="{ 'photo-media__facet--active': selectedFacet === facet }"
        type="button"
        @click="emit('select-facet', facet)"
      >
        {{ facet }}
      </button>
    </div>
  </aside>
</template>
