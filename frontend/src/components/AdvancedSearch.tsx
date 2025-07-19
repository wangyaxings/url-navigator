import { useState } from 'react';
import { Search, Filter, X, Tag } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Category } from '@/types';

interface AdvancedSearchOptions {
  query: string;
  category: string;
  tags: string[];
  startDate: string;
  endDate: string;
  sortBy: string;
  searchIn: string[];
}

interface AdvancedSearchProps {
  categories: Category[];
  onSearch: (options: AdvancedSearchOptions) => void;
  onReset: () => void;
}

export function AdvancedSearch({ categories, onSearch, onReset }: AdvancedSearchProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [searchOptions, setSearchOptions] = useState<AdvancedSearchOptions>({
    query: '',
    category: '',
    tags: [],
    startDate: '',
    endDate: '',
    sortBy: 'date',
    searchIn: ['title', 'description', 'url']
  });

  const handleSearch = () => {
    onSearch(searchOptions);
    setIsOpen(false);
  };

  const handleReset = () => {
    const defaultOptions: AdvancedSearchOptions = {
      query: '',
      category: '',
      tags: [],
      startDate: '',
      endDate: '',
      sortBy: 'date',
      searchIn: ['title', 'description', 'url']
    };
    setSearchOptions(defaultOptions);
    onReset();
    setIsOpen(false);
  };

  const addTag = (tag: string) => {
    if (tag.trim() && !searchOptions.tags.includes(tag.trim())) {
      setSearchOptions(prev => ({
        ...prev,
        tags: [...prev.tags, tag.trim()]
      }));
    }
  };

  const removeTag = (tagToRemove: string) => {
    setSearchOptions(prev => ({
      ...prev,
      tags: prev.tags.filter(tag => tag !== tagToRemove)
    }));
  };

  const toggleSearchField = (field: string) => {
    setSearchOptions(prev => ({
      ...prev,
      searchIn: prev.searchIn.includes(field)
        ? prev.searchIn.filter(f => f !== field)
        : [...prev.searchIn, field]
    }));
  };

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <Filter className="h-4 w-4 mr-2" />
          高级搜索
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center">
            <Search className="h-5 w-5 mr-2" />
            高级搜索
          </DialogTitle>
          <DialogDescription>
            使用多种条件精确搜索您的书签
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* 搜索查询 */}
          <div>
            <label className="text-sm font-medium mb-2 block">搜索关键词</label>
            <Input
              placeholder="输入搜索关键词..."
              value={searchOptions.query}
              onChange={(e) => setSearchOptions(prev => ({ ...prev, query: e.target.value }))}
            />
          </div>

          {/* 搜索范围 */}
          <div>
            <label className="text-sm font-medium mb-2 block">搜索范围</label>
            <div className="flex flex-wrap gap-2">
              {[
                { key: 'title', label: '标题' },
                { key: 'description', label: '描述' },
                { key: 'url', label: 'URL' }
              ].map(({ key, label }) => (
                <Button
                  key={key}
                  type="button"
                  variant={searchOptions.searchIn.includes(key) ? "default" : "outline"}
                  size="sm"
                  onClick={() => toggleSearchField(key)}
                >
                  {label}
                </Button>
              ))}
            </div>
          </div>

          {/* 分类选择 */}
          <div>
            <label className="text-sm font-medium mb-2 block">分类</label>
            <select
              value={searchOptions.category}
              onChange={(e) => setSearchOptions(prev => ({ ...prev, category: e.target.value }))}
              className="w-full px-3 py-2 border border-input bg-background text-foreground rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
            >
              <option value="">所有分类</option>
              {categories.map((category) => (
                <option key={category.id} value={category.name}>
                  {category.name}
                </option>
              ))}
            </select>
          </div>

          {/* 标签 */}
          <div>
            <label className="text-sm font-medium mb-2 block">标签</label>
            <div className="space-y-2">
              <Input
                placeholder="输入标签后按回车添加..."
                onKeyPress={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    addTag((e.target as HTMLInputElement).value);
                    (e.target as HTMLInputElement).value = '';
                  }
                }}
              />
              <div className="flex flex-wrap gap-2">
                {searchOptions.tags.map((tag, index) => (
                  <span
                    key={index}
                    className="inline-flex items-center px-2 py-1 bg-primary/10 text-primary rounded-md text-sm"
                  >
                    <Tag className="h-3 w-3 mr-1" />
                    {tag}
                    <button
                      onClick={() => removeTag(tag)}
                      className="ml-1 hover:text-destructive"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </span>
                ))}
              </div>
            </div>
          </div>

          {/* 日期范围 */}
          <div>
            <label className="text-sm font-medium mb-2 block">创建日期范围</label>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-xs text-muted-foreground mb-1 block">开始日期</label>
                <Input
                  type="date"
                  value={searchOptions.startDate}
                  onChange={(e) => setSearchOptions(prev => ({ ...prev, startDate: e.target.value }))}
                />
              </div>
              <div>
                <label className="text-xs text-muted-foreground mb-1 block">结束日期</label>
                <Input
                  type="date"
                  value={searchOptions.endDate}
                  onChange={(e) => setSearchOptions(prev => ({ ...prev, endDate: e.target.value }))}
                />
              </div>
            </div>
          </div>

          {/* 排序 */}
          <div>
            <label className="text-sm font-medium mb-2 block">排序方式</label>
            <select
              value={searchOptions.sortBy}
              onChange={(e) => setSearchOptions(prev => ({ ...prev, sortBy: e.target.value }))}
              className="w-full px-3 py-2 border border-input bg-background text-foreground rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
            >
              <option value="date">按创建日期</option>
              <option value="title">按标题</option>
              <option value="category">按分类</option>
            </select>
          </div>
        </div>

        {/* 操作按钮 */}
        <div className="flex justify-end space-x-2 pt-4 border-t">
          <Button variant="outline" onClick={handleReset}>
            重置
          </Button>
          <Button onClick={handleSearch}>
            <Search className="h-4 w-4 mr-2" />
            搜索
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}