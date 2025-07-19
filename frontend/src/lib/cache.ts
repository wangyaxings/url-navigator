/**
 * 缓存键名常量和辅助函数
 */

// 缓存键名前缀
export const CACHE_KEYS = {
  API: {
    URLS: 'api:urls',
    CATEGORIES: 'api:categories',
    URL_DETAIL: 'api:url-detail',
  },
  SEARCH: {
    BASIC: 'search:basic',
    ADVANCED: 'search:advanced',
    CATEGORIES: 'search:categories',
  },
  FAVICON: {
    PREFIX: 'favicon:',
  },
  USER_SETTINGS: {
    THEME: 'settings:theme',
    VIEW_MODE: 'settings:view-mode',
    GRID_COLUMNS: 'settings:grid-columns',
    SHORTCUTS_HELP: 'settings:shortcuts-help',
  }
} as const;

// 缓存策略枚举
export enum CacheStrategy {
  // 短期缓存 - 1分钟
  SHORT = 60 * 1000,
  // 中期缓存 - 5分钟
  MEDIUM = 5 * 60 * 1000,
  // 长期缓存 - 30分钟
  LONG = 30 * 60 * 1000,
  // 超长期缓存 - 24小时（如favicon）
  VERY_LONG = 24 * 60 * 60 * 1000,
}

/**
 * 生成搜索缓存键
 */
export function generateSearchCacheKey(query: string, category?: string): string {
  const normalizedQuery = query.toLowerCase().trim();
  return category && category !== 'all'
    ? `${CACHE_KEYS.SEARCH.BASIC}:${normalizedQuery}:${category}`
    : `${CACHE_KEYS.SEARCH.BASIC}:${normalizedQuery}`;
}

/**
 * 生成高级搜索缓存键
 */
export function generateAdvancedSearchCacheKey(options: any): string {
  const keyParts = [
    options.query || '',
    options.category || 'all',
    options.tags?.join(',') || '',
    options.dateRange || '',
    options.sortBy || '',
    options.sortOrder || ''
  ];
  return `${CACHE_KEYS.SEARCH.ADVANCED}:${keyParts.join(':')}`;
}

/**
 * 生成favicon缓存键
 */
export function generateFaviconCacheKey(url: string): string {
  try {
    const domain = new URL(url).hostname;
    return `${CACHE_KEYS.FAVICON.PREFIX}${domain}`;
  } catch {
    return `${CACHE_KEYS.FAVICON.PREFIX}unknown`;
  }
}

/**
 * 生成URL详情缓存键
 */
export function generateURLDetailCacheKey(urlId: string): string {
  return `${CACHE_KEYS.API.URL_DETAIL}:${urlId}`;
}

/**
 * 清除相关缓存
 */
export function clearRelatedCache(cacheManager: any, pattern: string): void {
  // 由于Map没有模式匹配，我们需要遍历所有键
  const stats = cacheManager.getStats();
  console.log(`Clearing cache pattern: ${pattern}, current items: ${stats.totalItems}`);

  // 根据模式清除特定缓存
  if (pattern === 'search') {
    cacheManager.clear(); // 如果是搜索相关，清除整个搜索缓存
  }
}

/**
 * 缓存大小估算器
 */
export function estimateCacheSize(data: any): number {
  try {
    return JSON.stringify(data).length * 2; // UTF-16 字符，每个字符2字节
  } catch {
    return 0;
  }
}

/**
 * 缓存健康检查
 */
export function performCacheHealthCheck(cacheManager: any): {
  isHealthy: boolean;
  stats: any;
  warnings: string[];
} {
  const stats = cacheManager.getStats();
  const warnings: string[] = [];
  let isHealthy = true;

  // 检查过期项目比例
  const expiredRatio = stats.expiredItems / stats.totalItems;
  if (expiredRatio > 0.3) {
    warnings.push(`High expired items ratio: ${(expiredRatio * 100).toFixed(1)}%`);
    isHealthy = false;
  }

  // 检查内存使用
  const memoryMB = stats.memoryUsage / (1024 * 1024);
  if (memoryMB > 10) {
    warnings.push(`High memory usage: ${memoryMB.toFixed(1)}MB`);
    isHealthy = false;
  }

  return {
    isHealthy,
    stats: {
      ...stats,
      memoryMB: memoryMB.toFixed(2),
      expiredRatio: (expiredRatio * 100).toFixed(1)
    },
    warnings
  };
}

/**
 * 调试缓存信息
 */
export function debugCacheInfo(cacheManager: any, prefix: string = 'Cache'): void {
  const healthCheck = performCacheHealthCheck(cacheManager);
  console.group(`${prefix} Debug Info`);
  console.log('Stats:', healthCheck.stats);
  if (healthCheck.warnings.length > 0) {
    console.warn('Warnings:', healthCheck.warnings);
  }
  console.groupEnd();
}