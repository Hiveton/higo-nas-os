/*
 * Generated contract bridge for HiGoOS web-pc.
 *
 * Source: ../../../../server-go/api/openapi.yaml
 * Generated: 2026-05-06
 *
 * This first checked-in generated surface keeps frontend imports stable until
 * a full OpenAPI generator is wired into the build. Runtime behavior is
 * delegated to the typed hand-written client in ../client.
 */
export { apiClient as generatedApiClient } from '../client';
export type { ApiClient } from '../client';

export const generatedContract = {
  source: 'server-go/api/openapi.yaml',
  basePath: '/api/v1',
  operationGroups: [
    'system',
    'desktop',
    'events',
    'files',
    'storage',
    'steward',
    'agents',
    'workflows',
    'assistant',
    'media',
    'downloads',
    'docker',
    'security',
    'monitoring',
    'settings',
    'remote',
  ],
} as const;
