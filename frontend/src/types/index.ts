export interface URLItem {
  id: string;
  title: string;
  url: string;
  description: string;
  category: string;
  tags: string[];
  favicon?: string;
  order: number;
  createdAt: string;
  updatedAt: string;
}

export interface Category {
  id: string;
  name: string;
  description: string;
  color: string;
}

export interface UpdateInfo {
  hasUpdate: boolean;
  currentVersion: string;
  latestVersion: string;
  updateUrl: string;
  releaseNotes: string;
  errorMessage?: string;
}

export interface UpdateProgress {
  phase: string;          // "downloading", "installing", "completed", "error"
  progress: number;       // 0-100
  speed?: string;         // Download speed (e.g. "1.2 MB/s")
  eta?: string;           // Estimated time (e.g. "2m 30s")
  downloaded?: number;    // Bytes downloaded
  total?: number;         // Total bytes
  message: string;        // Status message
  error?: string;         // Error message if any
}

export interface AdvancedSearchOptions {
  query: string;
  category: string;
  tags: string[];
  startDate: string;
  endDate: string;
  sortBy: string;
  searchIn: string[];
}