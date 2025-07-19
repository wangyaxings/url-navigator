import { useEffect, useRef } from 'react';
import { Edit2, Trash2, ExternalLink, Copy, FolderOpen } from 'lucide-react';
import { URLItem } from '@/types';

interface ContextMenuProps {
  isOpen: boolean;
  position: { x: number; y: number };
  url: URLItem | null;
  onClose: () => void;
  onEdit: (url: URLItem) => void;
  onDelete: (url: URLItem) => void;
  onOpen: (url: string) => void;
  onCopyUrl: (url: string) => void;
  onCopyTitle: (title: string) => void;
}

export function ContextMenu({
  isOpen,
  position,
  url,
  onClose,
  onEdit,
  onDelete,
  onOpen,
  onCopyUrl,
  onCopyTitle
}: ContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        onClose();
      }
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      document.addEventListener('keydown', handleEscape);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [isOpen, onClose]);

  // 调整菜单位置，防止超出屏幕
  const getMenuStyle = () => {
    if (!isOpen || !menuRef.current) return { display: 'none' };

    const menuWidth = 200;
    const menuHeight = 240;
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;

    let { x, y } = position;

    // 防止右侧溢出
    if (x + menuWidth > viewportWidth) {
      x = viewportWidth - menuWidth - 10;
    }

    // 防止底部溢出
    if (y + menuHeight > viewportHeight) {
      y = viewportHeight - menuHeight - 10;
    }

    // 防止左侧和顶部溢出
    x = Math.max(10, x);
    y = Math.max(10, y);

    return {
      position: 'fixed' as const,
      left: x,
      top: y,
      zIndex: 1000,
    };
  };

  if (!isOpen || !url) return null;

  return (
    <div
      ref={menuRef}
      style={getMenuStyle()}
      className="bg-background border border-border rounded-lg shadow-lg py-2 min-w-[200px] animate-in fade-in-0 zoom-in-95 duration-100"
    >
      <div className="px-3 py-2 border-b border-border">
        <p className="text-sm font-medium text-foreground truncate" title={url.title}>
          {url.title}
        </p>
        <p className="text-xs text-muted-foreground truncate" title={url.url}>
          {url.url}
        </p>
      </div>

      <div className="py-1">
        <button
          onClick={() => {
            onOpen(url.url);
            onClose();
          }}
          className="w-full flex items-center px-3 py-2 text-sm text-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
        >
          <ExternalLink className="h-4 w-4 mr-3" />
          打开链接
        </button>

        <button
          onClick={() => {
            onEdit(url);
            onClose();
          }}
          className="w-full flex items-center px-3 py-2 text-sm text-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
        >
          <Edit2 className="h-4 w-4 mr-3" />
          编辑书签
        </button>

        <div className="h-px bg-border my-1" />

        <button
          onClick={() => {
            onCopyUrl(url.url);
            onClose();
          }}
          className="w-full flex items-center px-3 py-2 text-sm text-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
        >
          <Copy className="h-4 w-4 mr-3" />
          复制链接
        </button>

        <button
          onClick={() => {
            onCopyTitle(url.title);
            onClose();
          }}
          className="w-full flex items-center px-3 py-2 text-sm text-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
        >
          <Copy className="h-4 w-4 mr-3" />
          复制标题
        </button>

        <button
          onClick={() => {
            const text = `${url.title} - ${url.url}`;
            navigator.clipboard.writeText(text);
            onClose();
          }}
          className="w-full flex items-center px-3 py-2 text-sm text-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
        >
          <FolderOpen className="h-4 w-4 mr-3" />
          复制为文本
        </button>

        <div className="h-px bg-border my-1" />

        <button
          onClick={() => {
            onDelete(url);
            onClose();
          }}
          className="w-full flex items-center px-3 py-2 text-sm text-destructive hover:bg-destructive hover:text-destructive-foreground transition-colors"
        >
          <Trash2 className="h-4 w-4 mr-3" />
          删除书签
        </button>
      </div>
    </div>
  );
}