// 缓存项接口
interface CacheItem<T> {
  data: T;
  timestamp: number;
  expiry: number;
  key: string;
}

// 缓存配置
interface CacheConfig {
  defaultTTL: number;
  maxSize: number;
  persistToDisk: boolean;
}

// 默认配置
const DEFAULT_CONFIG: CacheConfig = {
  defaultTTL: 5 * 60 * 1000, // 5分钟
  maxSize: 100,
  persistToDisk: true
};

class CacheManager<T = any> {
  private cache = new Map<string, CacheItem<T>>();
  private config: CacheConfig;
  private persistenceKey: string;

  constructor(persistenceKey: string = 'url-navigator-cache', config: Partial<CacheConfig> = {}) {
    this.persistenceKey = persistenceKey;
    this.config = { ...DEFAULT_CONFIG, ...config };

    if (this.config.persistToDisk) {
      this.loadFromDisk();
    }
    this.startCleanupTimer();
  }

  set<U extends T>(key: string, data: U, ttl?: number): void {
    const expiry = Date.now() + (ttl || this.config.defaultTTL);

    const item: CacheItem<U> = {
      data,
      timestamp: Date.now(),
      expiry,
      key
    };

    if (this.cache.size >= this.config.maxSize) {
      this.evictOldest();
    }

    this.cache.set(key, item as CacheItem<T>);

    if (this.config.persistToDisk) {
      this.saveToDisk();
    }
  }

  get<U extends T>(key: string): U | null {
    const item = this.cache.get(key);

    if (!item) {
      return null;
    }

    if (Date.now() > item.expiry) {
      this.delete(key);
      return null;
    }

    return item.data as U;
  }

  delete(key: string): boolean {
    const deleted = this.cache.delete(key);

    if (deleted && this.config.persistToDisk) {
      this.saveToDisk();
    }

    return deleted;
  }

  clear(): void {
    this.cache.clear();

    if (this.config.persistToDisk) {
      this.clearDisk();
    }
  }

  getStats() {
    const items = Array.from(this.cache.values());
    const now = Date.now();

    return {
      totalItems: items.length,
      expiredItems: items.filter(item => now > item.expiry).length,
      memoryUsage: this.estimateMemoryUsage()
    };
  }

  cleanup(): number {
    const now = Date.now();
    let cleanedCount = 0;

    for (const [key, item] of this.cache.entries()) {
      if (now > item.expiry) {
        this.cache.delete(key);
        cleanedCount++;
      }
    }

    if (cleanedCount > 0 && this.config.persistToDisk) {
      this.saveToDisk();
    }

    return cleanedCount;
  }

  private evictOldest(): void {
    let oldestKey: string | null = null;
    let oldestTimestamp = Date.now();

    for (const [key, item] of this.cache.entries()) {
      if (item.timestamp < oldestTimestamp) {
        oldestTimestamp = item.timestamp;
        oldestKey = key;
      }
    }

    if (oldestKey) {
      this.cache.delete(oldestKey);
    }
  }

  private estimateMemoryUsage(): number {
    let size = 0;

    for (const [key, item] of this.cache.entries()) {
      size += key.length * 2;
      size += JSON.stringify(item.data).length * 2;
      size += 24;
    }

    return size;
  }

  private loadFromDisk(): void {
    try {
      const stored = localStorage.getItem(this.persistenceKey);
      if (stored) {
        const items: CacheItem<T>[] = JSON.parse(stored);
        const now = Date.now();

        for (const item of items) {
          if (now <= item.expiry) {
            this.cache.set(item.key, item);
          }
        }
      }
    } catch (error) {
      console.warn('Failed to load cache from disk:', error);
    }
  }

  private saveToDisk(): void {
    try {
      const items = Array.from(this.cache.values());
      localStorage.setItem(this.persistenceKey, JSON.stringify(items));
    } catch (error) {
      console.warn('Failed to save cache to disk:', error);
    }
  }

  private clearDisk(): void {
    try {
      localStorage.removeItem(this.persistenceKey);
    } catch (error) {
      console.warn('Failed to clear disk cache:', error);
    }
  }

  private startCleanupTimer(): void {
    setInterval(() => {
      this.cleanup();
    }, 5 * 60 * 1000);
  }
}

// 创建全局缓存实例
export const apiCache = new CacheManager('api-cache', {
  defaultTTL: 2 * 60 * 1000,
  maxSize: 50,
  persistToDisk: false
});

export const faviconCache = new CacheManager('favicon-cache', {
  defaultTTL: 24 * 60 * 60 * 1000,
  maxSize: 200,
  persistToDisk: true
});

export const searchCache = new CacheManager('search-cache', {
  defaultTTL: 10 * 60 * 1000,
  maxSize: 30,
  persistToDisk: false
});

export function withCache<T extends (...args: any[]) => Promise<any>>(
  fn: T,
  cache: CacheManager,
  keyGenerator: (...args: Parameters<T>) => string,
  ttl?: number
): T {
  return (async (...args: Parameters<T>) => {
    const key = keyGenerator(...args);

    const cached = cache.get(key);
    if (cached !== null) {
      return cached;
    }

    const result = await fn(...args);
    cache.set(key, result, ttl);

    return result;
  }) as T;
}

export { CacheManager };
export default CacheManager;