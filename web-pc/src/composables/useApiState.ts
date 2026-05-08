import { computed, shallowRef, ref } from 'vue';
import { ApiError } from '../api/runtime';

export type ApiStateOptions<T> = {
  initialData?: T;
  keepPreviousData?: boolean;
};

export function useApiState<T, Args extends unknown[] = []>(
  loader: (...args: Args) => Promise<T>,
  options: ApiStateOptions<T> = {},
) {
  const data = shallowRef<T | null>(options.initialData ?? null);
  const loading = ref(false);
  const error = shallowRef<Error | ApiError | null>(null);

  const hasData = computed(() => data.value !== null);
  const errorMessage = computed(() => error.value?.message ?? '');

  async function execute(...args: Args) {
    loading.value = true;
    error.value = null;
    if (!options.keepPreviousData) {
      data.value = null;
    }

    try {
      const result = await loader(...args);
      data.value = result;
      return result;
    } catch (reason) {
      error.value = normalizeError(reason);
      throw error.value;
    } finally {
      loading.value = false;
    }
  }

  function reset(nextData: T | null = options.initialData ?? null) {
    data.value = nextData;
    error.value = null;
    loading.value = false;
  }

  return {
    data,
    loading,
    error,
    hasData,
    errorMessage,
    execute,
    reset,
  };
}

function normalizeError(reason: unknown) {
  if (reason instanceof ApiError || reason instanceof Error) return reason;
  if (typeof reason === 'string') return new Error(reason);
  return new Error('API request failed');
}
