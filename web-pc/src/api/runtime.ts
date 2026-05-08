export type ApiQueryValue = string | number | boolean | null | undefined;
export type ApiQuery = Record<string, ApiQueryValue | ApiQueryValue[]>;

export type ApiEnvelope<T> = {
  data?: T;
  success?: boolean;
  ok?: boolean;
  code?: string;
  message?: string;
  error?: string | { code?: string; message?: string; details?: unknown };
  requestId?: string;
};

export type ApiRequestOptions = Omit<RequestInit, 'body' | 'method'> & {
  body?: unknown;
  query?: ApiQuery;
  requestId?: string;
  parseAs?: 'json' | 'text' | 'blob' | 'void';
};

type InternalRequestOptions = ApiRequestOptions & {
  method?: string;
};

export type EventStreamOptions = {
  query?: ApiQuery;
  requestId?: string;
  withCredentials?: boolean;
};

export const API_BASE_URL = normalizeBaseUrl(import.meta.env.VITE_HIGOOS_API_BASE_URL ?? '');
export const API_CREDENTIALS = normalizeCredentials(import.meta.env.VITE_HIGOOS_API_CREDENTIALS);

export class ApiError extends Error {
  readonly status: number;
  readonly statusText: string;
  readonly code?: string;
  readonly details?: unknown;
  readonly requestId?: string;
  readonly url: string;
  readonly method: string;

  constructor(message: string, options: {
    status: number;
    statusText: string;
    code?: string;
    details?: unknown;
    requestId?: string;
    url: string;
    method: string;
  }) {
    super(message);
    this.name = 'ApiError';
    this.status = options.status;
    this.statusText = options.statusText;
    this.code = options.code;
    this.details = options.details;
    this.requestId = options.requestId;
    this.url = options.url;
    this.method = options.method;
  }
}

export function GET<T>(path: string, options?: ApiRequestOptions) {
  return request<T>(path, { ...options, method: 'GET' });
}

export function POST<T>(path: string, body?: unknown, options?: ApiRequestOptions) {
  return request<T>(path, { ...options, method: 'POST', body });
}

export function PUT<T>(path: string, body?: unknown, options?: ApiRequestOptions) {
  return request<T>(path, { ...options, method: 'PUT', body });
}

export function DELETE<T>(path: string, options?: ApiRequestOptions) {
  return request<T>(path, { ...options, method: 'DELETE' });
}

export async function request<T>(path: string, options: InternalRequestOptions = {}): Promise<T> {
  const method = options.method ?? 'GET';
  const { body, query, requestId: providedRequestId, parseAs, ...requestOptions } = options;
  const requestId = providedRequestId ?? createRequestId();
  const url = buildApiUrl(path, query);
  const headers = new Headers(options.headers);

  if (!headers.has('Accept')) headers.set('Accept', 'application/json');
  if (!headers.has('X-Request-Id')) headers.set('X-Request-Id', requestId);

  const init: RequestInit = {
    ...requestOptions,
    method,
    headers,
    credentials: options.credentials ?? API_CREDENTIALS,
  };

  if (body !== undefined) {
    if (isBodyInit(body)) {
      init.body = body;
    } else {
      if (!headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
      init.body = JSON.stringify(body);
    }
  }

  const response = await fetch(url, init);
  return parseResponse<T>(response, { method, requestId, url, parseAs });
}

export function createEventStream(path = '/api/v1/events/stream', options: EventStreamOptions = {}) {
  if (typeof EventSource === 'undefined') {
    throw new Error('EventSource is not available in this environment.');
  }

  const requestId = options.requestId ?? createRequestId();
  const url = buildApiUrl(path, {
    ...options.query,
    request_id: requestId,
  });

  return new EventSource(url, {
    withCredentials: options.withCredentials ?? API_CREDENTIALS === 'include',
  });
}

export function buildApiUrl(path: string, query?: ApiQuery) {
  const url = path.startsWith('http://') || path.startsWith('https://')
    ? path
    : `${API_BASE_URL}${path.startsWith('/') ? path : `/${path}`}`;

  const queryString = serializeQuery(query);
  if (!queryString) return url;
  return `${url}${url.includes('?') ? '&' : '?'}${queryString}`;
}

export function createRequestId() {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID();
  }
  return `req_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`;
}

function normalizeBaseUrl(value: string) {
  return value.trim().replace(/\/+$/, '');
}

function normalizeCredentials(value?: string): RequestCredentials {
  if (value === 'omit' || value === 'same-origin' || value === 'include') return value;
  return 'include';
}

function serializeQuery(query?: ApiQuery) {
  if (!query) return '';

  const params = new URLSearchParams();
  Object.entries(query).forEach(([key, value]) => {
    const values = Array.isArray(value) ? value : [value];
    values.forEach((item) => {
      if (item !== undefined && item !== null && item !== '') {
        params.append(key, String(item));
      }
    });
  });
  return params.toString();
}

function isBodyInit(body: unknown): body is BodyInit {
  return (
    typeof body === 'string' ||
    body instanceof FormData ||
    body instanceof Blob ||
    body instanceof ArrayBuffer ||
    body instanceof URLSearchParams ||
    body instanceof ReadableStream
  );
}

async function parseResponse<T>(
  response: Response,
  context: { method: string; requestId: string; url: string; parseAs?: ApiRequestOptions['parseAs'] },
): Promise<T> {
  if (context.parseAs === 'void' || response.status === 204) {
    if (!response.ok) throw buildApiError(response, context, undefined);
    return undefined as T;
  }

  if (context.parseAs === 'blob') {
    if (!response.ok) throw buildApiError(response, context, undefined);
    return response.blob() as Promise<T>;
  }

  const body = context.parseAs === 'text' ? await response.text() : await readJsonOrText(response);
  const envelope = asEnvelope<T>(body);

  if (!response.ok || envelope?.success === false || envelope?.ok === false || envelope?.error) {
    throw buildApiError(response, context, body);
  }

  if (envelope && Object.prototype.hasOwnProperty.call(envelope, 'data')) {
    return envelope.data as T;
  }
  return body as T;
}

async function readJsonOrText(response: Response) {
  const text = await response.text();
  if (!text) return undefined;

  const contentType = response.headers.get('Content-Type') ?? '';
  if (!contentType.includes('application/json')) return text;

  try {
    return JSON.parse(text) as unknown;
  } catch {
    return text;
  }
}

function asEnvelope<T>(body: unknown): ApiEnvelope<T> | null {
  if (!body || typeof body !== 'object' || Array.isArray(body)) return null;
  const candidate = body as ApiEnvelope<T>;
  const envelopeKeys = ['data', 'success', 'ok', 'code', 'message', 'error', 'requestId'];
  return envelopeKeys.some((key) => Object.prototype.hasOwnProperty.call(candidate, key)) ? candidate : null;
}

function buildApiError(
  response: Response,
  context: { method: string; requestId: string; url: string },
  body: unknown,
) {
  const envelope = asEnvelope<unknown>(body);
  const error = envelope?.error;
  const message =
    (typeof error === 'string' ? error : error?.message) ||
    envelope?.message ||
    response.statusText ||
    'API request failed';

  return new ApiError(message, {
    status: response.status,
    statusText: response.statusText,
    code: typeof error === 'string' ? envelope?.code : error?.code ?? envelope?.code,
    details: typeof error === 'string' ? body : error?.details ?? body,
    requestId: envelope?.requestId ?? response.headers.get('X-Request-Id') ?? context.requestId,
    url: context.url,
    method: context.method,
  });
}
