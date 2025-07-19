import React, { useState, useEffect, useRef } from 'react';
import { Download, CheckCircle, AlertCircle, RefreshCw, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { UpdateInfo, UpdateProgress } from '@/types';
import * as AppService from '../../wailsjs/go/main/App';

const UpdateChecker: React.FC = () => {
  const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null);
  const [isCheckingUpdate, setIsCheckingUpdate] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  const [showUpdateDialog, setShowUpdateDialog] = useState(false);
  const [showProgressDialog, setShowProgressDialog] = useState(false);
  const [currentVersion, setCurrentVersion] = useState<string>('');
  const [updateProgress, setUpdateProgress] = useState<UpdateProgress | null>(null);
  const progressIntervalRef = useRef<number | null>(null);

    // 检查更新
  const checkForUpdates = async () => {
    try {
      setIsCheckingUpdate(true);
      const [updateData, version] = await Promise.all([
        AppService.CheckForUpdates(),
        AppService.GetCurrentVersion()
      ]);
      setUpdateInfo(updateData);
      setCurrentVersion(version);

      if (updateData.hasUpdate) {
        setShowUpdateDialog(true);
      }
    } catch (error) {
      console.error('Failed to check for updates:', error);
    } finally {
      setIsCheckingUpdate(false);
    }
  };

  // 开始进度监控
  const startProgressMonitoring = () => {
    if (progressIntervalRef.current) {
      clearInterval(progressIntervalRef.current);
    }

    progressIntervalRef.current = setInterval(async () => {
      try {
        const progress = await AppService.GetUpdateProgress();
        setUpdateProgress(progress);

        // 如果更新完成或出错，停止监控
        if (progress?.phase === 'completed' || progress?.phase === 'error') {
          if (progressIntervalRef.current) {
            clearInterval(progressIntervalRef.current);
            progressIntervalRef.current = null;
          }

          // 如果出错，显示错误信息
          if (progress.phase === 'error') {
            setIsUpdating(false);
            alert(`更新失败: ${progress.error || progress.message}`);
            setShowProgressDialog(false);
          }
        }
      } catch (error) {
        console.error('Failed to get update progress:', error);
      }
    }, 500); // 每500ms检查一次进度
  };

  // 停止进度监控
  const stopProgressMonitoring = () => {
    if (progressIntervalRef.current) {
      clearInterval(progressIntervalRef.current);
      progressIntervalRef.current = null;
    }
  };

  // 下载并应用更新
  const downloadAndApplyUpdate = async () => {
    if (!updateInfo?.updateUrl) return;

    try {
      setIsUpdating(true);
      setShowUpdateDialog(false);
      setShowProgressDialog(true);

      // 开始监控进度
      startProgressMonitoring();

      // 开始下载更新
      await AppService.DownloadAndApplyUpdate(updateInfo.updateUrl);
      // 如果更新成功，应用会重启，这里的代码可能不会执行
    } catch (error) {
      console.error('Failed to update:', error);
      alert('更新失败，请稍后重试');
      setIsUpdating(false);
      setShowProgressDialog(false);
      stopProgressMonitoring();
    }
  };

  // 组件挂载时检查更新
  useEffect(() => {
    checkForUpdates();
  }, []);

  // 组件卸载时清理定时器
  useEffect(() => {
    return () => {
      stopProgressMonitoring();
    };
  }, []);



  return (
    <>
      <div className="flex justify-center">
        <Button
          variant="outline"
          onClick={checkForUpdates}
          disabled={isCheckingUpdate}
        >
          {isCheckingUpdate ? (
            <>
              <AlertCircle className="h-4 w-4 mr-2 animate-spin" />
              检查中...
            </>
          ) : (
            <>
              <Download className="h-4 w-4 mr-2" />
              检查更新
            </>
          )}
        </Button>
      </div>

      {/* 更新对话框 */}
      <Dialog open={showUpdateDialog} onOpenChange={setShowUpdateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center">
              <Download className="h-5 w-5 mr-2" />
              发现新版本
            </DialogTitle>
            <DialogDescription>
              有新版本可用，是否现在更新？
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="bg-gray-50 p-4 rounded-lg">
              <div className="flex justify-between items-center mb-2">
                <span className="text-sm font-medium">当前版本:</span>
                <span className="text-sm text-gray-600">{currentVersion}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm font-medium">最新版本:</span>
                <span className="text-sm text-green-600 font-medium">
                  {updateInfo?.latestVersion}
                </span>
              </div>
            </div>

            {updateInfo?.releaseNotes && (
              <div>
                <h4 className="text-sm font-medium mb-2">更新内容:</h4>
                <div className="bg-gray-50 p-3 rounded text-sm text-gray-700 max-h-32 overflow-y-auto">
                  <pre className="whitespace-pre-wrap">{updateInfo.releaseNotes}</pre>
                </div>
              </div>
            )}

            <div className="text-sm text-gray-500">
              更新后应用将自动重启
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowUpdateDialog(false)}
              disabled={isUpdating}
            >
              稍后更新
            </Button>
            <Button
              onClick={downloadAndApplyUpdate}
              disabled={isUpdating}
              className="bg-green-600 hover:bg-green-700"
            >
              <Download className="h-4 w-4 mr-2" />
              立即更新
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 更新进度对话框 */}
      <Dialog open={showProgressDialog} onOpenChange={() => {}}>
        <DialogContent className="sm:max-w-md" onPointerDownOutside={(e) => e.preventDefault()}>
          <DialogHeader>
            <DialogTitle className="flex items-center">
              <Download className="h-5 w-5 mr-2" />
              正在更新
            </DialogTitle>
            <DialogDescription>
              请不要关闭应用程序，更新完成后将自动重启
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            {/* 进度条 */}
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="font-medium">
                  {updateProgress?.message || '准备中...'}
                </span>
                <span className="text-gray-500">
                  {updateProgress?.progress || 0}%
                </span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div
                  className="bg-green-600 h-2 rounded-full transition-all duration-300 ease-out"
                  style={{ width: `${updateProgress?.progress || 0}%` }}
                />
              </div>
            </div>

            {/* 详细信息 */}
            {updateProgress?.phase === 'downloading' && (
              <div className="grid grid-cols-2 gap-4 text-sm">
                {updateProgress.speed && (
                  <div>
                    <span className="text-gray-500">下载速度:</span>
                    <div className="font-medium">{updateProgress.speed}</div>
                  </div>
                )}
                {updateProgress.eta && (
                  <div>
                    <span className="text-gray-500">剩余时间:</span>
                    <div className="font-medium">{updateProgress.eta}</div>
                  </div>
                )}
              </div>
            )}

            {/* 当前阶段状态 */}
            <div className="flex items-center space-x-2 text-sm">
              {updateProgress?.phase === 'downloading' && (
                <>
                  <Download className="h-4 w-4 text-blue-500 animate-pulse" />
                  <span>正在下载更新文件...</span>
                </>
              )}
              {updateProgress?.phase === 'installing' && (
                <>
                  <RefreshCw className="h-4 w-4 text-orange-500 animate-spin" />
                  <span>正在安装更新...</span>
                </>
              )}
              {updateProgress?.phase === 'completed' && (
                <>
                  <CheckCircle className="h-4 w-4 text-green-500" />
                  <span>更新完成，即将重启...</span>
                </>
              )}
              {updateProgress?.phase === 'error' && (
                <>
                  <X className="h-4 w-4 text-red-500" />
                  <span>更新失败: {updateProgress.error}</span>
                </>
              )}
            </div>
          </div>

          {/* 只有出错时才显示关闭按钮 */}
          {updateProgress?.phase === 'error' && (
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => {
                  setShowProgressDialog(false);
                  setIsUpdating(false);
                  stopProgressMonitoring();
                }}
              >
                关闭
              </Button>
            </DialogFooter>
          )}
        </DialogContent>
      </Dialog>

      {/* 无更新提示 */}
      {updateInfo && !updateInfo.hasUpdate && !isCheckingUpdate && (
        <div className="fixed bottom-4 right-4 bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded shadow-lg">
          <div className="flex items-center">
            <CheckCircle className="h-4 w-4 mr-2" />
            <span className="text-sm">当前已是最新版本</span>
          </div>
        </div>
      )}
    </>
  );
};

export default UpdateChecker;