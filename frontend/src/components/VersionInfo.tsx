// frontend/src/components/VersionInfo.tsx
import { useState, useEffect } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Github, ExternalLink, RefreshCw, Bug, AlertTriangle, CheckCircle } from 'lucide-react';
import { AppService } from '@/services/appService';
import { VersionDebug } from './VersionDebug';

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

// 动态获取版本更新内容
const getVersionChangelogs = (version: string): string[] => {
  // 移除硬编码，根据版本动态返回更新内容
  // 这里可以从后端API获取，或者根据版本号映射
  const versionClean = version.replace(/^v/, '');

  // 示例：根据版本号返回对应的更新内容
  const changelogs: Record<string, string[]> = {
    '1.2.1': [
      '• ✨ 导入导出功能 - 支持Chrome/Firefox/Edge书签',
      '• ⚡ 缓存优化 - 提升应用性能和响应速度',
      '• 🔍 高级搜索功能增强',
      '• 🎨 深色模式主题支持',
      '• 🖱️ 右键菜单操作',
      '• ⌨️ 快捷键支持',
      '• 📱 响应式布局优化',
      '• 🔧 Go语言发布工具替换PowerShell脚本',
      '• 🐛 修复版本号显示问题'
    ],
    '1.2.0': [
      '• 🎯 新增高级搜索功能',
      '• 🔄 优化数据同步机制',
      '• 🐛 修复若干已知问题'
    ],
    '1.1.0': [
      '• 📱 响应式设计优化',
      '• ⚡ 性能提升',
      '• 🔧 修复导入功能问题'
    ]
  };

  return changelogs[versionClean] || [
    '• 📝 版本更新',
    '• 🔧 性能优化和bug修复'
  ];
};

export function VersionInfo({ isOpen, onClose }: VersionInfoProps) {
  const [versionData, setVersionData] = useState<VersionData>({
    version: 'unknown',
    github_owner: 'wangyaxings',
    github_repo: 'url-navigator',
    app_name: 'URLNavigator'
  });
  const [isLoading, setIsLoading] = useState(false);
  const [showDebug, setShowDebug] = useState(false);
  const [versionSource, setVersionSource] = useState<{
    source: string;
    is_default: boolean;
    reliable: boolean;
  } | null>(null);

  const setIsOpen = (open: boolean) => {
    if (!open) onClose();
  };

  const loadVersionInfo = async () => {
    setIsLoading(true);
    try {
      // 并行获取版本信息和来源信息
      const [fullVersionInfo, versionWithSource] = await Promise.all([
        AppService.GetVersionInfo(),
        AppService.GetCurrentVersionWithSource()
      ]);

      // 设置版本数据
      if (fullVersionInfo) {
        setVersionData(fullVersionInfo);
      }

      // 设置版本来源信息
      if (versionWithSource) {
        setVersionSource({
          source: versionWithSource.source,
          is_default: versionWithSource.is_default,
          reliable: versionWithSource.reliable
        });
      }

    } catch (error) {
      console.error('Failed to load version info:', error);
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
    return version.startsWith('v') ? version : `v${version}`;
  };

  const buildDate = new Date().toLocaleDateString('zh-CN');

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center">
            关于 {versionData.app_name}
            <div className="ml-auto flex gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowDebug(true)}
                className="h-6 w-6 p-0"
                title="调试信息"
              >
                <Bug className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleRefresh}
                className="h-6 w-6 p-0"
                disabled={isLoading}
                title="刷新版本"
              >
                <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
              </Button>
            </div>
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
                <div className="flex items-center gap-2">
                  <Badge
                    variant={versionSource?.reliable ? "default" : "secondary"}
                    className={isLoading ? 'animate-pulse' : ''}
                  >
                    {formatVersion(versionData.version)}
                  </Badge>
                  {versionSource && (
                    <div className="flex items-center gap-1">
                      {versionSource.reliable ? (
                        <CheckCircle className="h-3 w-3 text-green-600" />
                      ) : (
                        <AlertTriangle className="h-3 w-3 text-yellow-600" />
                      )}
                    </div>
                  )}
                </div>
              </div>
              {versionSource && (
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">版本来源</span>
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-muted-foreground">
                      {AppService.getSourceDescription(versionSource.source)}
                    </span>
                    {versionSource.is_default && (
                      <Badge variant="outline" className="text-xs">
                        {versionSource.reliable ? '默认' : '兜底'}
                      </Badge>
                    )}
                  </div>
                </div>
              )}
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
              {/* 修复：动态显示版本号，而不是硬编码 */}
              <CardTitle className="text-lg">
                {formatVersion(versionData.version)} 更新内容
              </CardTitle>
            </CardHeader>
            <CardContent>
              <ul className="text-sm text-muted-foreground space-y-1">
                {/* 修复：根据当前版本动态显示更新内容 */}
                {getVersionChangelogs(versionData.version).map((item, index) => (
                  <li key={index}>{item}</li>
                ))}
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

        {/* 版本调试对话框 */}
        <VersionDebug isOpen={showDebug} onClose={() => setShowDebug(false)} />
      </DialogContent>
    </Dialog>
  );
}

// 修复SimpleVersionInfo组件中的版本获取逻辑
export function SimpleVersionInfo() {
  const [version, setVersion] = useState('加载中...');
  const [isLoading, setIsLoading] = useState(true);
  const [isReliable, setIsReliable] = useState(true);

  useEffect(() => {
    const loadVersion = async () => {
      setIsLoading(true);
      try {
        // 获取版本和来源信息
        const versionWithSource = await AppService.GetCurrentVersionWithSource();
        if (versionWithSource) {
          setVersion(versionWithSource.version);
          setIsReliable(versionWithSource.reliable);
          return;
        }

        // 兜底方案：尝试获取当前版本
        const versionStr = await AppService.GetCurrentVersion();
        setVersion(versionStr || 'unknown');
        setIsReliable(false);

      } catch (error) {
        console.error('Failed to get version:', error);
        setVersion('unknown');
        setIsReliable(false);
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
    <div className="flex items-center gap-1">
      <div className="text-xs text-muted-foreground">
        {version}
      </div>
      {!isReliable && (
        <AlertTriangle className="h-3 w-3 text-yellow-600" title="版本可能不准确" />
      )}
    </div>
  );
}