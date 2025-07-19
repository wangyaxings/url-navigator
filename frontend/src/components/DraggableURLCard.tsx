import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { ExternalLink, Edit2, Trash2, GripVertical } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { URLItem } from '@/types';

interface DraggableURLCardProps {
  url: URLItem;
  onEdit: (url: URLItem) => void;
  onDelete: (url: URLItem) => void;
  onOpen: (url: string) => void;
  getCategoryColor: (category: string) => string;
}

export function DraggableURLCard({
  url,
  onEdit,
  onDelete,
  onOpen,
  getCategoryColor
}: DraggableURLCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: url.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <Card
      ref={setNodeRef}
      style={style}
      className={`hover:shadow-lg transition-all duration-200 ${
        isDragging ? 'opacity-50 shadow-xl scale-105' : ''
      }`}
    >
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
              className="cursor-grab hover:cursor-grabbing"
              {...attributes}
              {...listeners}
            >
              <GripVertical className="h-4 w-4 text-muted-foreground" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onEdit(url)}
            >
              <Edit2 className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onDelete(url)}
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
            onClick={() => onOpen(url.url)}
          >
            <ExternalLink className="h-4 w-4 mr-1" />
            访问
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}