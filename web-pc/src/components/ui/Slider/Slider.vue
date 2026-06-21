<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(
  defineProps<{
    min?: number;
    max?: number;
    step?: number;
    disabled?: boolean;
    showValue?: boolean;
    /** Optional unit suffix shown next to the value, e.g. '%'. */
    unit?: string;
    id?: string;
  }>(),
  { min: 0, max: 100, step: 1, disabled: false, showValue: false, unit: '', id: undefined },
);

const model = defineModel<number>({ default: 0 });

const emit = defineEmits<{ change: [value: number] }>();

const percent = computed(() => {
  const span = props.max - props.min;
  if (span <= 0) return 0;
  return ((model.value - props.min) / span) * 100;
});

function onInput(event: Event) {
  model.value = Number((event.target as HTMLInputElement).value);
}
</script>

<template>
  <div class="ui-slider" :class="{ 'ui-slider--disabled': disabled }">
    <input
      :id="id"
      class="ui-slider__input"
      type="range"
      :min="min"
      :max="max"
      :step="step"
      :value="model"
      :disabled="disabled"
      :style="{ '--ui-slider-fill': `${percent}%` }"
      @input="onInput"
      @change="emit('change', model)"
    />
    <span v-if="showValue" class="ui-slider__value">{{ model }}{{ unit }}</span>
  </div>
</template>

<style scoped>
.ui-slider {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
}
.ui-slider--disabled {
  opacity: 0.55;
}

.ui-slider__input {
  flex: 1;
  min-width: 0;
  height: 6px;
  appearance: none;
  -webkit-appearance: none;
  background: linear-gradient(
    90deg,
    var(--accent) 0 var(--ui-slider-fill, 0%),
    rgba(120, 150, 180, 0.25) var(--ui-slider-fill, 0%) 100%
  );
  border-radius: 999px;
  cursor: pointer;
}
.ui-slider__input:disabled {
  cursor: not-allowed;
}

.ui-slider__input::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 16px;
  height: 16px;
  background: var(--text-inverse);
  border: 1px solid var(--accent);
  border-radius: 50%;
  box-shadow: 0 1px 3px rgba(15, 35, 60, 0.3);
  cursor: inherit;
}
.ui-slider__input::-moz-range-thumb {
  width: 16px;
  height: 16px;
  background: var(--text-inverse);
  border: 1px solid var(--accent);
  border-radius: 50%;
}
.ui-slider__input:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 4px;
}

.ui-slider__value {
  min-width: 40px;
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  text-align: right;
}
</style>
