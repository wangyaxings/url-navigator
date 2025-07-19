import { useState } from 'react';
import { Grid, List, Maximize2, Minimize2, LayoutGrid } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';

export type ViewMode = 'grid' | 'list';
export type GridColumns = 1 | 2 | 3 | 4 | 5 | 6;

interface LayoutControlsProps {
  viewMode: ViewMode;
  gridColumns: GridColumns;
  isFullscreen: boolean;
  onViewModeChange: (mode: ViewMode) => void;
  onGridColumnsChange: (columns: GridColumns) => void;
  onFullscreenToggle: () => void;
}

export function LayoutControls({
  viewMode,
  gridColumns,
  isFullscreen,
  onViewModeChange,
  onGridColumnsChange,
  onFullscreenToggle
}: LayoutControlsProps) {
  const [isLayoutDialogOpen, setIsLayoutDialogOpen] = useState(false);

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

  return (
    <div className="flex items-center space-x-2">
      {/* View Mode Toggle */}
      <div className="flex border rounded-md">
        <Button
          variant={viewMode === 'grid' ? 'default' : 'ghost'}
          size="sm"
          onClick={() => onViewModeChange('grid')}
          className="rounded-r-none"
        >
          <Grid className="h-4 w-4" />
        </Button>
        <Button
          variant={viewMode === 'list' ? 'default' : 'ghost'}
          size="sm"
          onClick={() => onViewModeChange('list')}
          className="rounded-l-none border-l"
        >
          <List className="h-4 w-4" />
        </Button>
      </div>

      {/* Grid Columns Selector */}
      {viewMode === 'grid' && (
        <Dialog open={isLayoutDialogOpen} onOpenChange={setIsLayoutDialogOpen}>
          <DialogTrigger asChild>
            <Button variant="outline" size="sm">
              <LayoutGrid className="h-4 w-4 mr-2" />
              {gridColumns}列
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle>网格布局设置</DialogTitle>
              <DialogDescription>
                选择网格列数以适应您的屏幕和偏好
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium mb-3 block">列数选择</label>
                <div className="grid grid-cols-3 gap-2">
                  {([1, 2, 3, 4, 5, 6] as GridColumns[]).map((columns) => (
                    <Button
                      key={columns}
                      variant={gridColumns === columns ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => {
                        onGridColumnsChange(columns);
                        setIsLayoutDialogOpen(false);
                      }}
                      className="flex flex-col items-center p-3 h-auto"
                    >
                      <div className={`grid ${getGridClass(columns)} gap-1 w-8 h-6 mb-1`}>
                        {Array.from({ length: Math.min(columns, 6) }).map((_, i) => (
                          <div key={i} className="bg-current rounded-sm opacity-60" />
                        ))}
                      </div>
                      <span className="text-xs">{columns}列</span>
                    </Button>
                  ))}
                </div>
              </div>

              <div className="text-xs text-muted-foreground bg-muted p-3 rounded">
                <p><strong>响应式说明：</strong></p>
                <ul className="mt-1 space-y-1">
                  <li>• 移动设备始终显示1列</li>
                  <li>• 平板设备自动调整为合适的列数</li>
                  <li>• 大屏幕显示您选择的列数</li>
                </ul>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}

      {/* Fullscreen Toggle */}
      <Button
        variant="outline"
        size="sm"
        onClick={onFullscreenToggle}
        title={isFullscreen ? "退出全屏" : "进入全屏"}
      >
        {isFullscreen ? (
          <Minimize2 className="h-4 w-4" />
        ) : (
          <Maximize2 className="h-4 w-4" />
        )}
      </Button>
    </div>
  );
}