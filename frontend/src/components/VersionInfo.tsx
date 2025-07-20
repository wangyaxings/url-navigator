import { useState, useEffect } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Github, ExternalLink, RefreshCw } from 'lucide-react';
import { AppService } from '@/services/appService';

interface VersionInfoProps {
  isOpen: boolean;
  onClose: () => void;
}

interface VersionData {
  version: string;
  github_owner: string;
  github_repo: string;
  app_name: string;
}

export function VersionInfo({ isOpen, onClose }: VersionInfoProps) {
  const [versionData, setVersionData] = useState<VersionData>({
    version: 'unknown',
    github_owner: 'wangyaxings',
    github_repo: 'url-navigator',
    app_name: 'URLNavigator'
  });
  const [isLoading, setIsLoading] = useState(false);

  const setIsOpen = (open: boolean) => {
    if (!open) onClose();
  };

  const loadVersionInfo = async () => {
    setIsLoading(true);
    try {
      // 首先尝试获取完整版本信息
      const fullVersionInfo = await AppService.GetVersionInfo();
      if (fullVersionInfo && fullVersionInfo.version && fullVersionInfo.version !== 'unknown') {
        setVersionData(fullVersionInfo);
        return;
      }

      // 如果获取完整信息失败，尝试获取当前版本
      const currentVersion = await AppService.GetCurrentVersion();
      if (currentVersion && currentVersion !== 'unknown') {
        setVersionData(prev => ({
          ...prev,
          version: currentVersion
        }));
        return;
      }

      // 最后尝试从wails.json获取版本（调试用）
      try {
        const wailsVersion = await AppService.GetVersionFromWails();
        if (wailsVersion && wailsVersion !== 'unknown') {
          setVersionData(prev => ({
            ...prev,
            version: wailsVersion
          }));
        }
      } catch (error) {
        console.warn('Failed to get version from wails.json:', error);
      }

    } catch (error) {
      console.error('Failed to load version info:', error);
      // 保持默认值
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen) {
      loadVersionInfo();
    }
  }, [isOpen]);

  const handleRefresh = () => {
    loadVersionInfo();
  };

  const formatVersion = (version: string) => {
    if (!version || version === 'unknown') {
      return 'unknown';
    }
    // 确保显示时有v前缀
    return version.startsWith('v') ? version : `v${version}`;
  };

  const buildDate = new Date().toLocaleDateString('zh-CN');

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center justify-between">
            关于 {versionData.app_name}
            <Button
              variant="ghost"
              size="sm"
              onClick={handleRefresh}
              disabled={isLoading}
              className="h-8 w-8 p-0"
            >
              <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
            </Button>
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-lg">应用信息</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">当前版本</span>
                <Badge variant="secondary" className={isLoading ? 'animate-pulse' : ''}>
                  {formatVersion(versionData.version)}
                </Badge>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">构建日期</span>
                <span className="text-sm text-muted-foreground">{buildDate}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">技术栈</span>
                <span className="text-sm text-muted-foreground">Wails + React</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">GitHub</span>
                <span className="text-sm text-muted-foreground">
                  {versionData.github_owner}/{versionData.github_repo}
                </span>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-lg">更新内容 {formatVersion(versionData.version)}</CardTitle>
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
                <li>• 🔧 Go语言发布工具替换PowerShell脚本</li>
                <li>• 🐛 修复版本号显示问题</li>
              </ul>
            </CardContent>
          </Card>

          <div className="flex items-center justify-between pt-2 border-t">
            <div className="flex items-center space-x-2">
              <Button variant="outline" size="sm" asChild>
                <a
                  href={`https://github.com/${versionData.github_owner}/${versionData.github_repo}`}
                  target="_blank"
                  rel="noopener noreferrer"
                >
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

// 简单版本号显示组件 - 修复硬编码问题
export function SimpleVersionInfo() {
  const [version, setVersion] = useState('加载中...');
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const loadVersion = async () => {
      setIsLoading(true);
      try {
        // 首先尝试获取当前版本
        const versionStr = await AppService.GetCurrentVersion();
        if (versionStr && versionStr !== 'unknown') {
          // 确保版本有v前缀用于显示
          const formattedVersion = versionStr.startsWith('v') ? versionStr : `v${versionStr}`;
          setVersion(formattedVersion);
          return;
        }

        // 如果失败，尝试获取完整版本信息
        const versionInfo = await AppService.GetVersionInfo();
        if (versionInfo && versionInfo.version && versionInfo.version !== 'unknown') {
          const formattedVersion = versionInfo.version.startsWith('v') ? versionInfo.version : `v${versionInfo.version}`;
          setVersion(formattedVersion);
          return;
        }

        // 最后尝试从wails.json获取
        try {
          const wailsVersion = await AppService.GetVersionFromWails();
          if (wailsVersion && wailsVersion !== 'unknown') {
            setVersion(wailsVersion);
            return;
          }
        } catch (error) {
          console.warn('Failed to get version from wails.json:', error);
        }

        // 如果所有方法都失败，显示unknown
        setVersion('unknown');

      } catch (error) {
        console.error('Failed to get version:', error);
        setVersion('unknown');
      } finally {
        setIsLoading(false);
      }
    };

    loadVersion();
  }, []);

  if (isLoading) {
    return (
      <div className="text-xs text-muted-foreground animate-pulse">
        加载中...
      </div>
    );
  }

  return (
    <div className="text-xs text-muted-foreground">
      {version}
    </div>
  );
}