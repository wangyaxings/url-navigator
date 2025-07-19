import { useState, useEffect, useRef } from 'react';
import { Plus, Search, Settings, ExternalLink } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { URLItem, Category, AdvancedSearchOptions } from '@/types';
import URLFormDialog from '@/components/URLFormDialog';
import CategoryManager from '@/components/CategoryManager';
import UpdateChecker from '@/components/UpdateChecker';
import { ThemeToggle } from '@/components/ThemeToggle';
import { AdvancedSearch } from '@/components/AdvancedSearch';
import { DraggableURLCard } from '@/components/DraggableURLCard';
import { URLListView } from '@/components/URLListView';
import { LayoutControls, ViewMode, GridColumns } from '@/components/LayoutControls';
import { KeyboardShortcutsHelp, KeyboardShortcutTooltip } from '@/components/KeyboardShortcutsHelp';
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts';
import { ContextMenu } from '@/components/ContextMenu';
import { DndContext, closestCenter, DragEndEvent } from '@dnd-kit/core';
import { SortableContext, rectSortingStrategy } from '@dnd-kit/sortable';

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
  const [viewMode, setViewMode] = useState<ViewMode>('grid');
  const [gridColumns, setGridColumns] = useState<GridColumns>(4);
  const [isFullscreen, setIsFullscreen] = useState(false);
    const [isHelpDialogOpen, setIsHelpDialogOpen] = useState(false);
  const [showShortcutTooltip, setShowShortcutTooltip] = useState(true);

  // Context menu state
  const [contextMenu, setContextMenu] = useState<{
    isOpen: boolean;
    position: { x: number; y: number };
    url: URLItem | null;
  }>({
    isOpen: false,
    position: { x: 0, y: 0 },
    url: null
  });

  // Refs for keyboard shortcuts
  const searchInputRef = useRef<HTMLInputElement>(null);
  const advancedSearchTriggerRef = useRef<HTMLButtonElement>(null);

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

   // 处理全屏切换
   const handleFullscreenToggle = () => {
     setIsFullscreen(!isFullscreen);
   };

   // 获取网格CSS类
   const getGridClass = (columns: GridColumns) => {
     const gridClasses = {
       1: 'grid-cols-1',
       2: 'grid-cols-1 md:grid-cols-2',
       3: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3',
       4: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4',
       5: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5',
       6: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 3xl:grid-cols-6'
     };
         return gridClasses[columns];
  };

  // 键盘快捷键处理函数
  const handleToggleViewMode = () => {
    setViewMode(current => current === 'grid' ? 'list' : 'grid');
  };

  const handleFocusSearch = () => {
    if (searchInputRef.current) {
      searchInputRef.current.focus();
      searchInputRef.current.select();
    }
  };

  const handleTriggerAdvancedSearch = () => {
    if (advancedSearchTriggerRef.current) {
      advancedSearchTriggerRef.current.click();
    }
  };

  // 右键菜单处理函数
  const handleContextMenu = (event: React.MouseEvent, url: URLItem) => {
    setContextMenu({
      isOpen: true,
      position: { x: event.clientX, y: event.clientY },
      url
    });
  };

  const handleCloseContextMenu = () => {
    setContextMenu({
      isOpen: false,
      position: { x: 0, y: 0 },
      url: null
    });
  };

  const handleCopyUrl = async (url: string) => {
    try {
      await navigator.clipboard.writeText(url);
      // 可以在这里添加提示
    } catch (error) {
      console.error('Failed to copy URL:', error);
    }
  };

  const handleCopyTitle = async (title: string) => {
    try {
      await navigator.clipboard.writeText(title);
      // 可以在这里添加提示
    } catch (error) {
      console.error('Failed to copy title:', error);
    }
  };

  // 使用键盘快捷键
  useKeyboardShortcuts({
    onNewBookmark: () => setIsAddDialogOpen(true),
    onSearch: handleFocusSearch,
    onAdvancedSearch: handleTriggerAdvancedSearch,
    onToggleViewMode: handleToggleViewMode,
    onToggleFullscreen: handleFullscreenToggle,
    onRefresh: loadData,
    onShowHelp: () => setIsHelpDialogOpen(true)
  });

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
              ref={searchInputRef}
              placeholder="搜索网址、标题或描述... (Ctrl+F)"
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
            triggerRef={advancedSearchTriggerRef}
          />
          {isAdvancedSearchActive && (
            <Button variant="outline" size="sm" onClick={handleResetSearch}>
              重置搜索
            </Button>
          )}
          <LayoutControls
            viewMode={viewMode}
            gridColumns={gridColumns}
            isFullscreen={isFullscreen}
            onViewModeChange={setViewMode}
            onGridColumnsChange={setGridColumns}
            onFullscreenToggle={handleFullscreenToggle}
          />
        </div>

        {/* URL显示区域 */}
        <div className={`${isFullscreen ? 'fixed inset-0 z-50 bg-background p-6 overflow-auto' : ''}`}>
          {isFullscreen && (
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-2xl font-bold">书签浏览</h2>
              <Button onClick={handleFullscreenToggle} variant="outline">
                <ExternalLink className="h-4 w-4 mr-2" />
                退出全屏
              </Button>
            </div>
          )}

          {viewMode === 'list' ? (
            <URLListView
              urls={filteredUrls}
              onEdit={setEditingURL}
              onDelete={setDeleteDialogURL}
              onOpen={openURL}
              getCategoryColor={getCategoryColor}
              isDragEnabled={true}
              onContextMenu={handleContextMenu}
            />
          ) : (
            <DndContext
              collisionDetection={closestCenter}
              onDragEnd={handleDragEnd}
            >
              <SortableContext
                items={filteredUrls.map(url => url.id)}
                strategy={rectSortingStrategy}
              >
                <div className={`grid ${getGridClass(gridColumns)} gap-6`}>
                  {filteredUrls
                    .sort((a, b) => a.order - b.order)
                    .map((url) => (
                      <DraggableURLCard
                        key={url.id}
                        url={url}
                        onEdit={setEditingURL}
                        onDelete={setDeleteDialogURL}
                        onOpen={openURL}
                        getCategoryColor={getCategoryColor}
                        onContextMenu={handleContextMenu}
                      />
                    ))}
                </div>
              </SortableContext>
            </DndContext>
          )}
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

        {/* 键盘快捷键帮助 */}
        <KeyboardShortcutsHelp
          isOpen={isHelpDialogOpen}
          onOpenChange={setIsHelpDialogOpen}
        />

        {/* 快捷键提示 */}
        {showShortcutTooltip && (
          <div className="relative">
            <KeyboardShortcutTooltip />
            <Button
              onClick={() => setShowShortcutTooltip(false)}
              variant="ghost"
              size="sm"
              className="absolute -top-2 -right-2 h-6 w-6 p-0 rounded-full bg-background border"
            >
              ×
                         </Button>
           </div>
         )}

         {/* 右键菜单 */}
         <ContextMenu
           isOpen={contextMenu.isOpen}
           position={contextMenu.position}
           url={contextMenu.url}
           onClose={handleCloseContextMenu}
           onEdit={setEditingURL}
           onDelete={setDeleteDialogURL}
           onOpen={openURL}
           onCopyUrl={handleCopyUrl}
           onCopyTitle={handleCopyTitle}
         />
       </div>
     </div>
   );
}

export default App;