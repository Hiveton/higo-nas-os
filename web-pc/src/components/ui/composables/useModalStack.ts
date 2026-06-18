import { onBeforeUnmount, ref } from 'vue';

/**
 * Tracks the global stack of open overlays so nested modals layer correctly
 * and only the topmost one reacts to Escape / backdrop clicks.
 *
 * Each opened overlay gets a z-index of `--z-modal` + (depth * 2), with the
 * backdrop one below the panel.
 */

const BASE_Z = 1210; // mirrors --z-modal in tokens.css
const stack = ref<symbol[]>([]);

export function useModalStack() {
  const id = Symbol('modal');

  function push() {
    if (!stack.value.includes(id)) {
      stack.value = [...stack.value, id];
    }
  }

  function pop() {
    stack.value = stack.value.filter((entry) => entry !== id);
  }

  function depth(): number {
    return Math.max(0, stack.value.indexOf(id));
  }

  function isTop(): boolean {
    return stack.value[stack.value.length - 1] === id;
  }

  function panelZIndex(): number {
    return BASE_Z + depth() * 2;
  }

  function backdropZIndex(): number {
    return BASE_Z - 1 + depth() * 2;
  }

  onBeforeUnmount(pop);

  return { push, pop, depth, isTop, panelZIndex, backdropZIndex };
}
