import { Keyboard, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { KEYBOARD_SHORTCUTS_HELP } from '@/hooks/useKeyboardShortcuts';

interface KeyboardShortcutsHelpProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

export function KeyboardShortcutsHelp({ isOpen, onOpenChange }: KeyboardShortcutsHelpProps) {
  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center">
            <Keyboard className="h-5 w-5 mr-2" />
            键盘快捷键
          </DialogTitle>
          <DialogDescription>
            使用这些快捷键可以提高您的操作效率
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="grid gap-3">
            {KEYBOARD_SHORTCUTS_HELP.map((shortcut, index) => (
              <div key={index} className="flex items-center justify-between p-3 bg-muted rounded-lg">
                <span className="text-sm text-muted-foreground">
                  {shortcut.description}
                </span>
                <div className="flex items-center space-x-1">
                  {shortcut.keys.map((key, keyIndex) => (
                    <div key={keyIndex} className="flex items-center">
                      <kbd className="px-2 py-1 text-xs font-semibold text-foreground bg-background border border-border rounded shadow-sm">
                        {key}
                      </kbd>
                      {keyIndex < shortcut.keys.length - 1 && (
                        <span className="mx-1 text-muted-foreground">+</span>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>

          <div className="mt-6 p-4 bg-primary/5 border border-primary/20 rounded-lg">
            <h4 className="text-sm font-medium text-primary mb-2">小贴士</h4>
            <ul className="text-xs text-muted-foreground space-y-1">
              <li>• 在输入框中时，大部分快捷键会被禁用</li>
              <li>• ESC键可以关闭当前打开的对话框</li>
              <li>• 快捷键在所有页面中都可使用</li>
            </ul>
          </div>
        </div>

        <div className="flex justify-end pt-4 border-t">
          <Button onClick={() => onOpenChange(false)} variant="outline">
            <X className="h-4 w-4 mr-2" />
            关闭
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// 快捷键提示组件，可以在界面上显示
export function KeyboardShortcutTooltip() {
  return (
    <div className="fixed bottom-4 right-4 bg-card border border-border shadow-lg rounded-lg p-3 text-xs text-muted-foreground max-w-xs z-40">
      <div className="flex items-center mb-2">
        <Keyboard className="h-3 w-3 mr-1" />
        <span className="font-medium">快捷键提示</span>
      </div>
      <div className="space-y-1">
        <div>按 <kbd className="px-1 py-0.5 bg-muted rounded text-xs">F1</kbd> 查看所有快捷键</div>
        <div>按 <kbd className="px-1 py-0.5 bg-muted rounded text-xs">Ctrl+N</kbd> 新建书签</div>
        <div>按 <kbd className="px-1 py-0.5 bg-muted rounded text-xs">Ctrl+F</kbd> 搜索</div>
      </div>
    </div>
  );
}