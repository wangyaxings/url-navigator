import { ExternalLink, Edit2, Trash2, GripVertical } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { URLItem } from '@/types';

interface URLListViewProps {
  urls: URLItem[];
  onEdit: (url: URLItem) => void;
  onDelete: (url: URLItem) => void;
  onOpen: (url: string) => void;
  getCategoryColor: (category: string) => string;
  isDragEnabled?: boolean;
  onContextMenu?: (event: React.MouseEvent, url: URLItem) => void;
}

export function URLListView({
  urls,
  onEdit,
  onDelete,
  onOpen,
  getCategoryColor,
  isDragEnabled = false,
  onContextMenu
}: URLListViewProps) {
  return (
    <div className="space-y-2">
      {urls
        .sort((a, b) => a.order - b.order)
        .map((url) => (
          <div
            key={url.id}
            className="flex items-center space-x-3 p-3 bg-card border border-border rounded-lg hover:shadow-md transition-all duration-200 group"
            onContextMenu={(e) => {
              e.preventDefault();
              onContextMenu?.(e, url);
            }}
          >
            {/* Drag Handle */}
            {isDragEnabled && (
              <Button
                variant="ghost"
                size="sm"
                className="cursor-grab hover:cursor-grabbing opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <GripVertical className="h-4 w-4 text-muted-foreground" />
              </Button>
            )}

            {/* Favicon */}
            <div className="flex-shrink-0">
              {url.favicon ? (
                <img
                  src={url.favicon}
                  alt={`${url.title} favicon`}
                  className="w-5 h-5 rounded-sm"
                  onError={(e) => {
                    (e.target as HTMLImageElement).style.display = 'none';
                  }}
                />
              ) : (
                <div className="w-5 h-5 rounded-sm bg-muted flex items-center justify-center">
                  <ExternalLink className="w-3 h-3 text-muted-foreground" />
                </div>
              )}
            </div>

            {/* Category Indicator */}
            <div className="flex-shrink-0">
              <span
                className="inline-block w-3 h-3 rounded-full"
                style={{ backgroundColor: getCategoryColor(url.category) }}
                title={url.category}
              />
            </div>

            {/* Content */}
            <div className="flex-1 min-w-0">
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <h3 className="font-medium text-foreground truncate">
                    {url.title}
                  </h3>
                  <p className="text-sm text-muted-foreground truncate mt-1">
                    {url.description}
                  </p>
                  <div className="flex items-center space-x-2 mt-1">
                    <span className="text-xs text-muted-foreground">
                      {url.category}
                    </span>
                    {url.tags && url.tags.length > 0 && (
                      <>
                        <span className="text-xs text-muted-foreground">•</span>
                        <div className="flex flex-wrap gap-1">
                          {url.tags.slice(0, 3).map((tag, index) => (
                            <span
                              key={index}
                              className="inline-block px-1.5 py-0.5 text-xs bg-secondary text-secondary-foreground rounded"
                            >
                              {tag}
                            </span>
                          ))}
                          {url.tags.length > 3 && (
                            <span className="text-xs text-muted-foreground">
                              +{url.tags.length - 3}
                            </span>
                          )}
                        </div>
                      </>
                    )}
                  </div>
                </div>
              </div>
            </div>

            {/* URL Display */}
            <div className="hidden md:block flex-shrink-0 max-w-xs">
              <p className="text-xs text-muted-foreground truncate">
                {url.url}
              </p>
            </div>

            {/* Actions */}
            <div className="flex items-center space-x-1 opacity-0 group-hover:opacity-100 transition-opacity">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onEdit(url)}
                title="编辑"
              >
                <Edit2 className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onDelete(url)}
                title="删除"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => onOpen(url.url)}
                title="访问网站"
              >
                <ExternalLink className="h-4 w-4" />
              </Button>
            </div>
          </div>
        ))}
    </div>
  );
}