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