import { useEffect, useCallback } from 'react';

export interface KeyboardShortcuts {
  onNewBookmark: () => void;
  onSearch: () => void;
  onAdvancedSearch: () => void;
  onToggleViewMode: () => void;
  onToggleFullscreen: () => void;
  onRefresh: () => void;
  onShowHelp: () => void;
}

interface KeyMap {
  [key: string]: () => void;
}

export function useKeyboardShortcuts(shortcuts: KeyboardShortcuts) {
  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    const { ctrlKey, shiftKey, altKey, key } = event;

    // 如果在输入框中，忽略快捷键（除了ESC）
    const target = event.target as HTMLElement;
    const isInInput = target.tagName === 'INPUT' ||
                      target.tagName === 'TEXTAREA' ||
                      target.contentEditable === 'true';

    if (isInInput && key !== 'Escape') {
      return;
    }

    const keyMap: KeyMap = {
      // Ctrl + N: 新建书签
      'ctrl+n': () => {
        event.preventDefault();
        shortcuts.onNewBookmark();
      },

      // Ctrl + F: 搜索
      'ctrl+f': () => {
        event.preventDefault();
        shortcuts.onSearch();
      },

      // Ctrl + Shift + F: 高级搜索
      'ctrl+shift+f': () => {
        event.preventDefault();
        shortcuts.onAdvancedSearch();
      },

      // Ctrl + M: 切换视图模式
      'ctrl+m': () => {
        event.preventDefault();
        shortcuts.onToggleViewMode();
      },

      // F11: 全屏切换
      'f11': () => {
        event.preventDefault();
        shortcuts.onToggleFullscreen();
      },

      // F5 或 Ctrl + R: 刷新
      'f5': () => {
        event.preventDefault();
        shortcuts.onRefresh();
      },
      'ctrl+r': () => {
        event.preventDefault();
        shortcuts.onRefresh();
      },

      // F1 或 Ctrl + /: 显示帮助
      'f1': () => {
        event.preventDefault();
        shortcuts.onShowHelp();
      },
      'ctrl+/': () => {
        event.preventDefault();
        shortcuts.onShowHelp();
      },

      // ESC: 取消当前操作（通用）
      'escape': () => {
        // 可以用于关闭对话框等
        const dialogs = document.querySelectorAll('[role="dialog"]');
        if (dialogs.length > 0) {
          const lastDialog = dialogs[dialogs.length - 1];
          const closeButton = lastDialog.querySelector('[aria-label="Close"]') as HTMLElement;
          if (closeButton) {
            closeButton.click();
          }
        }
      }
    };

    // 构建键组合字符串
    const keyString = [
      ctrlKey && 'ctrl',
      shiftKey && 'shift',
      altKey && 'alt',
      key.toLowerCase()
    ].filter(Boolean).join('+');

    // 执行对应的快捷键操作
    if (keyMap[keyString]) {
      keyMap[keyString]();
    }
  }, [shortcuts]);

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [handleKeyDown]);
}

// 快捷键帮助信息
export const KEYBOARD_SHORTCUTS_HELP = [
  { keys: ['Ctrl', 'N'], description: '新建书签' },
  { keys: ['Ctrl', 'F'], description: '搜索书签' },
  { keys: ['Ctrl', 'Shift', 'F'], description: '高级搜索' },
  { keys: ['Ctrl', 'M'], description: '切换视图模式' },
  { keys: ['F11'], description: '全屏切换' },
  { keys: ['F5'], description: '刷新数据' },
  { keys: ['Ctrl', 'R'], description: '刷新数据' },
  { keys: ['F1'], description: '显示帮助' },
  { keys: ['Ctrl', '/'], description: '显示帮助' },
  { keys: ['ESC'], description: '取消操作' }
];