import { useState, useEffect } from 'react';
import { Plus, Search, Settings, ExternalLink, Edit2, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { URLItem, Category, AdvancedSearchOptions } from '@/types';
import URLFormDialog from '@/components/URLFormDialog';
import CategoryManager from '@/components/CategoryManager';
import UpdateChecker from '@/components/UpdateChecker';
import { ThemeToggle } from '@/components/ThemeToggle';
import { AdvancedSearch } from '@/components/AdvancedSearch';
import { DraggableURLCard } from '@/components/DraggableURLCard';
import { DndContext, closestCenter, DragEndEvent } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy, rectSortingStrategy } from '@dnd-kit/sortable';

// 导入 Wails 生成的绑定
import * as AppService from '../wailsjs/go/main/App';

function App() {
  const [urls, setUrls] = useState<URLItem[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [editingURL, setEditingURL] = useState<URLItem | null>(null);
  const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
  const [isCategoryManagerOpen, setIsCategoryManagerOpen] = useState(false);
  const [deleteDialogURL, setDeleteDialogURL] = useState<URLItem | null>(null);
  const [isAdvancedSearchActive, setIsAdvancedSearchActive] = useState(false);

  // 加载数据
  const loadData = async () => {
    try {
      const [urlsData, categoriesData] = await Promise.all([
        AppService.GetURLs(),
        AppService.GetCategories()
      ]);
      setUrls(urlsData || []);
      setCategories(categoriesData || []);
    } catch (error) {
      console.error('Failed to load data:', error);
    }
  };

  // 搜索URLs
  const searchURLs = async (keyword: string) => {
    try {
      const results = await AppService.SearchURLs(keyword);
      setUrls(results || []);
    } catch (error) {
      console.error('Failed to search URLs:', error);
    }
  };

  // 高级搜索
  const handleAdvancedSearch = async (options: AdvancedSearchOptions) => {
    try {
      const results = await AppService.AdvancedSearchURLs(options);
      setUrls(results || []);
      setIsAdvancedSearchActive(true);
    } catch (error) {
      console.error('Failed to perform advanced search:', error);
    }
  };

  // 重置搜索
  const handleResetSearch = async () => {
    setSearchTerm('');
    setSelectedCategory('all');
    setIsAdvancedSearchActive(false);
    await loadData();
  };

  // 处理拖拽结束
  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;

    if (active.id !== over?.id) {
      const oldIndex = urls.findIndex((url) => url.id === active.id);
      const newIndex = urls.findIndex((url) => url.id === over?.id);

      if (oldIndex !== -1 && newIndex !== -1) {
        // 重新排列本地数组
        const newUrls = [...urls];
        const [reorderedItem] = newUrls.splice(oldIndex, 1);
        newUrls.splice(newIndex, 0, reorderedItem);

        setUrls(newUrls);

        // 创建新的顺序数组
        const urlIDs = newUrls.map(url => url.id);

        try {
          await AppService.ReorderURLs(urlIDs);
        } catch (error) {
          console.error('Failed to reorder URLs:', error);
          // 如果失败，恢复原来的顺序
          setUrls(urls);
        }
      }
    }
  };

  // 删除URL
  const handleDeleteURL = async (url: URLItem) => {
    try {
      await AppService.DeleteURL(url.id);
      setDeleteDialogURL(null);
      if (searchTerm) {
        searchURLs(searchTerm);
      } else {
        loadData();
      }
    } catch (error) {
      console.error('Failed to delete URL:', error);
    }
  };

  // 打开URL
  const openURL = (url: string) => {
    window.open(url, '_blank');
  };

  // 过滤URLs
  const filteredUrls = selectedCategory === 'all'
    ? urls
    : urls.filter(url => url.category === selectedCategory);

  // 获取分类颜色
  const getCategoryColor = (categoryName: string) => {
    const category = categories.find(c => c.name === categoryName);
    return category?.color || '#6b7280';
  };

  useEffect(() => {
    loadData();
  }, []);

  useEffect(() => {
    if (searchTerm) {
      searchURLs(searchTerm);
    } else {
      loadData();
    }
  }, [searchTerm]);

  return (
    <div className="min-h-screen bg-background text-foreground p-6 transition-colors duration-300">
      <div className="max-w-7xl mx-auto">
        {/* 顶部工具栏 */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold text-foreground">网址导航</h1>
            <p className="text-muted-foreground mt-1">管理您的网址收藏</p>
          </div>
          <div className="flex items-center space-x-3">
            <ThemeToggle />
            <UpdateChecker />
            <Button
              variant="outline"
              onClick={() => setIsCategoryManagerOpen(true)}
            >
              <Settings className="h-4 w-4 mr-2" />
              分类管理
            </Button>
            <Button onClick={() => setIsAddDialogOpen(true)}>
              <Plus className="h-4 w-4 mr-2" />
              添加网址
            </Button>
          </div>
        </div>

        {/* 搜索和过滤 */}
        <div className="flex items-center space-x-4 mb-6">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground h-4 w-4" />
            <Input
              placeholder="搜索网址、标题或描述..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="pl-10"
            />
          </div>
          <select
            value={selectedCategory}
            onChange={(e) => setSelectedCategory(e.target.value)}
            className="px-4 py-2 border border-input bg-background text-foreground rounded-md focus:outline-none focus:ring-2 focus:ring-ring transition-colors"
          >
            <option value="all">所有分类</option>
            {categories.map((category) => (
              <option key={category.id} value={category.name}>
                {category.name}
              </option>
            ))}
          </select>
          <AdvancedSearch
            categories={categories}
            onSearch={handleAdvancedSearch}
            onReset={handleResetSearch}
          />
          {isAdvancedSearchActive && (
            <Button variant="outline" size="sm" onClick={handleResetSearch}>
              重置搜索
            </Button>
          )}
        </div>

        {/* URL卡片网格 */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {filteredUrls.map((url) => (
            <Card key={url.id} className="hover:shadow-lg transition-shadow">
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center space-x-2">
                      {url.favicon ? (
                        <img
                          src={url.favicon}
                          alt={`${url.title} favicon`}
                          className="w-5 h-5 flex-shrink-0 rounded-sm"
                          onError={(e) => {
                            (e.target as HTMLImageElement).style.display = 'none';
                          }}
                        />
                      ) : (
                        <div className="w-5 h-5 flex-shrink-0 rounded-sm bg-muted flex items-center justify-center">
                          <ExternalLink className="w-3 h-3 text-muted-foreground" />
                        </div>
                      )}
                      <CardTitle className="text-lg font-semibold text-foreground line-clamp-1">
                        {url.title}
                      </CardTitle>
                    </div>
                    <CardDescription className="mt-1 line-clamp-2">
                      {url.description}
                    </CardDescription>
                  </div>
                  <div className="flex items-center space-x-1 ml-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setEditingURL(url)}
                    >
                      <Edit2 className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setDeleteDialogURL(url)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="pt-0">
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <span
                      className="inline-block w-3 h-3 rounded-full"
                      style={{ backgroundColor: getCategoryColor(url.category) }}
                    />
                    <span className="text-sm text-muted-foreground">{url.category}</span>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => openURL(url.url)}
                  >
                    <ExternalLink className="h-4 w-4 mr-1" />
                    访问
                  </Button>
                </div>
                {url.tags && url.tags.length > 0 && (
                  <div className="flex flex-wrap gap-1 mt-3">
                    {url.tags.map((tag, index) => (
                      <span
                        key={index}
                        className="inline-block px-2 py-1 text-xs bg-gray-100 text-gray-700 rounded"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>

        {filteredUrls.length === 0 && (
          <div className="text-center py-12">
            <p className="text-gray-500 text-lg">没有找到相关网址</p>
            <p className="text-gray-400 mt-1">尝试调整搜索条件或添加新的网址</p>
          </div>
        )}

        {/* 添加/编辑URL对话框 */}
        <URLFormDialog
          isOpen={isAddDialogOpen || !!editingURL}
          onClose={() => {
            setIsAddDialogOpen(false);
            setEditingURL(null);
          }}
          onSave={() => {
            setIsAddDialogOpen(false);
            setEditingURL(null);
            loadData();
          }}
          editingURL={editingURL}
          categories={categories}
        />

        {/* 分类管理对话框 */}
        <CategoryManager
          isOpen={isCategoryManagerOpen}
          onClose={() => setIsCategoryManagerOpen(false)}
          onSave={() => {
            setIsCategoryManagerOpen(false);
            loadData();
          }}
        />

        {/* 删除确认对话框 */}
        <Dialog open={!!deleteDialogURL} onOpenChange={() => setDeleteDialogURL(null)}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>确认删除</DialogTitle>
              <DialogDescription>
                您确定要删除网址 "{deleteDialogURL?.title}" 吗？此操作无法撤销。
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button variant="outline" onClick={() => setDeleteDialogURL(null)}>
                取消
              </Button>
              <Button
                variant="destructive"
                onClick={() => deleteDialogURL && handleDeleteURL(deleteDialogURL)}
              >
                删除
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  );
}

export default App;