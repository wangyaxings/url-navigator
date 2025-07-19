import { useState, useRef } from 'react';
import { Upload, Download, FileText, Chrome, Globe, AlertCircle, CheckCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import * as AppService from '../../wailsjs/go/main/App';

interface ImportExportProps {
  onImportComplete: () => void;
}

export function ImportExport({ onImportComplete }: ImportExportProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [isImporting, setIsImporting] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const [importResult, setImportResult] = useState<{
    success: boolean;
    count: number;
    message: string;
  } | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleExport = async () => {
    try {
      setIsExporting(true);
      const exportData = await AppService.ExportBookmarks();

      const blob = new Blob([exportData], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `url-navigator-bookmarks-${new Date().toISOString().split('T')[0]}.json`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);

    } catch (error) {
      console.error('Export failed:', error);
      alert('导出失败，请重试');
    } finally {
      setIsExporting(false);
    }
  };

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      importBookmarks(file);
    }
  };

  const importBookmarks = async (file: File) => {
    try {
      setIsImporting(true);
      setImportResult(null);

      const fileContent = await file.text();
      let importCount = 0;

      if (file.name.endsWith('.json')) {
        importCount = await AppService.ImportChromeBookmarks(fileContent);
      } else if (file.name.endsWith('.html') || file.name.endsWith('.htm')) {
        importCount = await AppService.ImportNetscapeBookmarks(fileContent);
      } else {
        throw new Error('不支持的文件格式');
      }

      setImportResult({
        success: true,
        count: importCount,
        message: `成功导入 ${importCount} 个书签`
      });

      onImportComplete();

    } catch (error) {
      console.error('Import failed:', error);
      setImportResult({
        success: false,
        count: 0,
        message: `导入失败: ${error instanceof Error ? error.message : '未知错误'}`
      });
    } finally {
      setIsImporting(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const triggerFileInput = () => {
    fileInputRef.current?.click();
  };

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <Upload className="h-4 w-4 mr-2" />
          导入/导出
        </Button>
      </DialogTrigger>

      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center">
            <FileText className="h-5 w-5 mr-2" />
            书签导入导出
          </DialogTitle>
          <DialogDescription>
            导入浏览器书签或导出当前书签数据
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center">
                <Download className="h-5 w-5 mr-2" />
                导出书签
              </CardTitle>
              <CardDescription>
                将当前所有书签和分类导出为JSON文件
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Button
                onClick={handleExport}
                disabled={isExporting}
                className="w-full"
              >
                {isExporting ? (
                  <>
                    <AlertCircle className="h-4 w-4 mr-2 animate-spin" />
                    导出中...
                  </>
                ) : (
                  <>
                    <Download className="h-4 w-4 mr-2" />
                    导出书签
                  </>
                )}
              </Button>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center">
                <Upload className="h-5 w-5 mr-2" />
                导入书签
              </CardTitle>
              <CardDescription>
                从浏览器书签文件导入数据
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="flex items-center p-3 border border-border rounded-lg">
                  <Chrome className="h-8 w-8 mr-3 text-blue-500" />
                  <div>
                    <p className="font-medium">Chrome 书签</p>
                    <p className="text-sm text-muted-foreground">JSON 格式文件</p>
                  </div>
                </div>
                <div className="flex items-center p-3 border border-border rounded-lg">
                  <Globe className="h-8 w-8 mr-3 text-orange-500" />
                  <div>
                    <p className="font-medium">Firefox/Edge 书签</p>
                    <p className="text-sm text-muted-foreground">HTML 格式文件</p>
                  </div>
                </div>
              </div>

              <Button
                onClick={triggerFileInput}
                disabled={isImporting}
                className="w-full"
              >
                {isImporting ? (
                  <>
                    <AlertCircle className="h-4 w-4 mr-2 animate-spin" />
                    导入中...
                  </>
                ) : (
                  <>
                    <Upload className="h-4 w-4 mr-2" />
                    选择书签文件
                  </>
                )}
              </Button>

              <input
                ref={fileInputRef}
                type="file"
                accept=".json,.html,.htm"
                onChange={handleFileSelect}
                className="hidden"
              />

              {importResult && (
                <div className={`p-4 rounded-lg border ${
                  importResult.success
                    ? 'bg-green-50 border-green-200 text-green-800 dark:bg-green-900/20 dark:border-green-800 dark:text-green-200'
                    : 'bg-red-50 border-red-200 text-red-800 dark:bg-red-900/20 dark:border-red-800 dark:text-red-200'
                }`}>
                  <div className="flex items-center">
                    {importResult.success ? (
                      <CheckCircle className="h-5 w-5 mr-2" />
                    ) : (
                      <AlertCircle className="h-5 w-5 mr-2" />
                    )}
                    <span className="font-medium">{importResult.message}</span>
                  </div>
                </div>
              )}

              <div className="text-sm text-muted-foreground bg-muted p-3 rounded-lg">
                <p className="font-medium mb-2">导入说明：</p>
                <ul className="space-y-1">
                  <li>• Chrome: 设置 → 书签 → 书签管理器 → 导出书签</li>
                  <li>• Firefox: 书签 → 管理所有书签 → 导入和备份 → 导出书签为HTML</li>
                  <li>• Edge: 设置 → 导入浏览器数据 → 导出收藏夹</li>
                  <li>• 导入的书签将添加到"导入"分类中</li>
                </ul>
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="flex justify-end pt-4 border-t">
          <Button variant="outline" onClick={() => setIsOpen(false)}>
            关闭
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}