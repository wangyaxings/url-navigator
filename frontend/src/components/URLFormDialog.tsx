import React, { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { URLItem, Category } from '@/types';
import * as AppService from '../../wailsjs/go/main/App';

interface URLFormDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: () => void;
  editingURL: URLItem | null;
  categories: Category[];
}

const URLFormDialog: React.FC<URLFormDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  editingURL,
  categories
}) => {
  const [formData, setFormData] = useState({
    title: '',
    url: '',
    description: '',
    category: '',
    tags: ''
  });
  const [isLoading, setIsLoading] = useState(false);
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  // 重置表单
  const resetForm = () => {
    setFormData({
      title: '',
      url: '',
      description: '',
      category: categories.length > 0 ? categories[0].name : '',
      tags: ''
    });
    setErrors({});
  };

  // 验证表单
  const validateForm = () => {
    const newErrors: { [key: string]: string } = {};

    if (!formData.title.trim()) {
      newErrors.title = '标题不能为空';
    }

    if (!formData.url.trim()) {
      newErrors.url = 'URL不能为空';
    } else {
      // 简单的URL验证
      try {
        new URL(formData.url);
      } catch {
        newErrors.url = '请输入有效的URL';
      }
    }

    if (!formData.category) {
      newErrors.category = '请选择分类';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  // 处理保存
  const handleSave = async () => {
    if (!validateForm()) return;

    setIsLoading(true);
    try {
      const tags = formData.tags.split(',').map(tag => tag.trim()).filter(tag => tag.length > 0);

      if (editingURL) {
        // 更新现有URL
        await AppService.UpdateURL(
          editingURL.id,
          formData.title,
          formData.url,
          formData.description,
          formData.category,
          tags
        );
      } else {
        // 添加新URL
        await AppService.AddURL(
          formData.title,
          formData.url,
          formData.description,
          formData.category,
          tags
        );
      }

      onSave();
      resetForm();
    } catch (error) {
      console.error('Failed to save URL:', error);
    } finally {
      setIsLoading(false);
    }
  };

  // 处理关闭
  const handleClose = () => {
    resetForm();
    onClose();
  };

  // 处理输入变化
  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    // 清除相关错误
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  // 当编辑URL时，填充表单
  useEffect(() => {
    if (editingURL) {
      setFormData({
        title: editingURL.title,
        url: editingURL.url,
        description: editingURL.description,
        category: editingURL.category,
        tags: editingURL.tags.join(', ')
      });
    } else {
      resetForm();
    }
  }, [editingURL, categories]);

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>
            {editingURL ? '编辑网址' : '添加网址'}
          </DialogTitle>
          <DialogDescription>
            {editingURL ? '修改网址信息' : '添加一个新的网址到您的收藏'}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          {/* 标题 */}
          <div className="grid gap-2">
            <label htmlFor="title" className="text-sm font-medium">
              标题 *
            </label>
            <Input
              id="title"
              placeholder="网站标题"
              value={formData.title}
              onChange={(e) => handleInputChange('title', e.target.value)}
              className={errors.title ? 'border-red-500' : ''}
            />
            {errors.title && (
              <span className="text-sm text-red-500">{errors.title}</span>
            )}
          </div>

          {/* URL */}
          <div className="grid gap-2">
            <label htmlFor="url" className="text-sm font-medium">
              URL *
            </label>
            <Input
              id="url"
              placeholder="https://example.com"
              value={formData.url}
              onChange={(e) => handleInputChange('url', e.target.value)}
              className={errors.url ? 'border-red-500' : ''}
            />
            {errors.url && (
              <span className="text-sm text-red-500">{errors.url}</span>
            )}
          </div>

          {/* 描述 */}
          <div className="grid gap-2">
            <label htmlFor="description" className="text-sm font-medium">
              描述
            </label>
            <textarea
              id="description"
              placeholder="网站描述（可选）"
              value={formData.description}
              onChange={(e) => handleInputChange('description', e.target.value)}
              className="flex min-h-[60px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              rows={3}
            />
          </div>

          {/* 分类 */}
          <div className="grid gap-2">
            <label htmlFor="category" className="text-sm font-medium">
              分类 *
            </label>
            <select
              id="category"
              value={formData.category}
              onChange={(e) => handleInputChange('category', e.target.value)}
              className={`flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${
                errors.category ? 'border-red-500' : ''
              }`}
            >
              <option value="">选择分类</option>
              {categories.map((category) => (
                <option key={category.id} value={category.name}>
                  {category.name}
                </option>
              ))}
            </select>
            {errors.category && (
              <span className="text-sm text-red-500">{errors.category}</span>
            )}
          </div>

          {/* 标签 */}
          <div className="grid gap-2">
            <label htmlFor="tags" className="text-sm font-medium">
              标签
            </label>
            <Input
              id="tags"
              placeholder="标签1, 标签2, 标签3（用逗号分隔）"
              value={formData.tags}
              onChange={(e) => handleInputChange('tags', e.target.value)}
            />
            <span className="text-xs text-gray-500">
              用逗号分隔多个标签
            </span>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={isLoading}>
            取消
          </Button>
          <Button onClick={handleSave} disabled={isLoading}>
            {isLoading ? '保存中...' : (editingURL ? '更新' : '添加')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default URLFormDialog;