import React, { useState, useEffect } from 'react';
import { Download, CheckCircle, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { UpdateInfo } from '@/types';
import * as AppService from '../../wailsjs/go/main/App';

const UpdateChecker: React.FC = () => {
  const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null);
  const [isCheckingUpdate, setIsCheckingUpdate] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  const [showUpdateDialog, setShowUpdateDialog] = useState(false);
  const [currentVersion, setCurrentVersion] = useState<string>('');

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

  // 下载并应用更新
  const downloadAndApplyUpdate = async () => {
    if (!updateInfo?.updateUrl) return;

    try {
      setIsUpdating(true);
      await AppService.DownloadAndApplyUpdate(updateInfo.updateUrl);
      // 如果更新成功，应用会重启，这里的代码可能不会执行
    } catch (error) {
      console.error('Failed to update:', error);
      alert('更新失败，请稍后重试');
    } finally {
      setIsUpdating(false);
    }
  };

  // 组件挂载时检查更新
  useEffect(() => {
    checkForUpdates();
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
              {isUpdating ? (
                <>
                  <AlertCircle className="h-4 w-4 mr-2 animate-spin" />
                  更新中...
                </>
              ) : (
                <>
                  <Download className="h-4 w-4 mr-2" />
                  立即更新
                </>
              )}
            </Button>
          </DialogFooter>
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