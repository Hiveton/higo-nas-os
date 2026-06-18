/**
 * Simplified Chinese — the source-of-truth locale.
 * Keys mirror the component tree: `common.*`, `windows.<app>.*`.
 * Migrate hardcoded strings here incrementally as files are touched.
 */
export default {
  common: {
    actions: {
      confirm: '确定',
      cancel: '取消',
      delete: '删除',
      save: '保存',
      add: '添加',
      close: '关闭',
      clear: '清空',
      retry: '重试',
      refresh: '刷新',
    },
    states: {
      loading: '加载中',
      empty: '暂无数据',
      error: '出错了',
    },
  },
  windows: {
    download: {
      title: '下载中心',
      deleteTitle: '删除下载任务',
      deleteHint: '可以只删除队列记录，也可以同时删除已经下载的文件和未完成的断点文件。',
      deleteTaskOnly: '仅删除任务',
      deleteTaskAndFiles: '删除任务和文件',
      emptyQueue: '下载队列为空',
      emptyQueueHint: '添加 HTTP、BT、磁力或订阅链接后会显示在这里。',
      emptyCategory: '该分类暂无任务',
      emptyCategoryHint: '切换分类查看其它任务。',
      emptyDetail: '暂无下载任务',
      emptyDetailHint: '从左侧添加真实下载链接。',
    },
  },
} as const;
