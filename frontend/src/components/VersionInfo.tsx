import { useState, useEffect } from 'react';
import { Info, Github, ExternalLink } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import * as AppService from '../../wailsjs/go/main/App';

interface VersionData {
  version: string;
  github_owner: string;
  github_repo: string;
  app_name: string;
}

export function VersionInfo() {
  const [isOpen, setIsOpen] = useState(false);
  const [versionData, setVersionData] = useState<VersionData>({
    version: '1.2.1',
    github_owner: 'wangyaxings',
    github_repo: 'url-navigator',
    app_name: 'URLNavigator'
  });
  const [buildDate] = useState(new Date().toLocaleDateString('zh-CN'));

  useEffect(() => {
    const loadVersionInfo = async () => {
      try {
        const data = await AppService.GetVersionInfo();
        if (data && data.version) {
          setVersionData(data);
        }
      } catch (error) {
        console.warn('Failed to get version info from backend:', error);
        // 使用默认值，已在useState中设置
      }
    };

    loadVersionInfo();
  }, []);

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          className="text-xs text-muted-foreground hover:text-foreground"
        >
          <Info className="h-3 w-3 mr-1" />
          v{versionData.version}
        </Button>
      </DialogTrigger>

      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center">
            <Info className="h-5 w-5 mr-2" />
            关于 {versionData.app_name}
          </DialogTitle>
          <DialogDescription>
            智能书签管理工具
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-lg">版本信息</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">当前版本</span>
                <Badge variant="secondary">v{versionData.version}</Badge>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">构建日期</span>
                <span className="text-sm text-muted-foreground">{buildDate}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">技术栈</span>
                <span className="text-sm text-muted-foreground">Wails + React</span>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-lg">更新内容 v{versionData.version}</CardTitle>
            </CardHeader>
            <CardContent>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• ✨ 导入导出功能 - 支持Chrome/Firefox/Edge书签</li>
                <li>• ⚡ 缓存优化 - 提升应用性能和响应速度</li>
                <li>• 🔍 高级搜索功能增强</li>
                <li>• 🎨 深色模式主题支持</li>
                <li>• 🖱️ 右键菜单操作</li>
                <li>• ⌨️ 快捷键支持</li>
                <li>• 📱 响应式布局优化</li>
              </ul>
            </CardContent>
          </Card>

          <div className="flex items-center justify-between pt-2 border-t">
            <div className="flex items-center space-x-2">
              <Button variant="outline" size="sm" asChild>
                <a href={`https://github.com/${versionData.github_owner}/${versionData.github_repo}`} target="_blank" rel="noopener noreferrer">
                  <Github className="h-4 w-4 mr-1" />
                  GitHub
                  <ExternalLink className="h-3 w-3 ml-1" />
                </a>
              </Button>
            </div>
            <Button variant="outline" onClick={() => setIsOpen(false)}>
              关闭
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// 简单版本号显示组件
export function SimpleVersionInfo() {
  const [version, setVersion] = useState('1.2.1');

  useEffect(() => {
    const loadVersion = async () => {
      try {
        const versionStr = await AppService.GetCurrentVersion();
        if (versionStr) {
          setVersion(versionStr);
        }
      } catch (error) {
        console.warn('Failed to get version:', error);
      }
    };

    loadVersion();
  }, []);

  return (
    <div className="text-xs text-muted-foreground">
      v{version}
    </div>
  );
}