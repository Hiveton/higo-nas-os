/**
 * English locale. Partial — missing keys fall back to zh-CN.
 * Fill in as the UI is internationalized.
 */
export default {
  common: {
    actions: {
      confirm: 'Confirm',
      cancel: 'Cancel',
      delete: 'Delete',
      save: 'Save',
      add: 'Add',
      close: 'Close',
      clear: 'Clear',
      retry: 'Retry',
      refresh: 'Refresh',
    },
    states: {
      loading: 'Loading',
      empty: 'No data',
      error: 'Something went wrong',
    },
  },
  windows: {
    download: {
      title: 'Downloads',
      deleteTitle: 'Delete download task',
      deleteHint:
        'You can remove just the queue record, or also delete the downloaded files and unfinished resume data.',
      deleteTaskOnly: 'Remove task only',
      deleteTaskAndFiles: 'Delete task and files',
      emptyQueue: 'Download queue is empty',
      emptyQueueHint: 'Add an HTTP, BT, magnet, or subscription link and it will show up here.',
      emptyCategory: 'No tasks in this category',
      emptyCategoryHint: 'Switch categories to view other tasks.',
      emptyDetail: 'No download tasks',
      emptyDetailHint: 'Add a real download link from the left.',
    },
  },
};
