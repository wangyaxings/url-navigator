import React, { useState, useEffect } from 'react';
import { Plus, Edit2, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Category } from '@/types';
import * as AppService from '../../wailsjs/go/main/App';

interface CategoryManagerProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: () => void;
}

const CategoryManager: React.FC<CategoryManagerProps> = ({
  isOpen,
  onClose,
  onSave,
}) => {
  const [categories, setCategories] = useState<Category[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [editingCategory, setEditingCategory] = useState<Category | null>(null);
  const [showAddForm, setShowAddForm] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    color: '#3b82f6'
  });

  const colorOptions = [
    '#3b82f6', // blue
    '#10b981', // green
    '#f59e0b', // yellow
    '#8b5cf6', // purple
    '#ef4444', // red
    '#f97316', // orange
    '#06b6d4', // cyan
    '#84cc16', // lime
    '#ec4899', // pink
    '#6b7280', // gray
  ];

  // 加载分类数据
  const loadCategories = async () => {
    try {
      const data = await AppService.GetCategories();
      setCategories(data || []);
    } catch (error) {
      console.error('Failed to load categories:', error);
    }
  };

  // 保存分类
  const saveCategories = async (updatedCategories: Category[]) => {
    try {
      setIsLoading(true);
      await AppService.SaveCategories(updatedCategories);
      setCategories(updatedCategories);
    } catch (error) {
      console.error('Failed to save categories:', error);
    } finally {
      setIsLoading(false);
    }
  };

  // 添加分类
  const handleAddCategory = async () => {
    if (!formData.name.trim()) return;

    console.log('Adding category:', formData);

    const newCategory: Category = {
      id: Date.now().toString(),
      name: formData.name,
      description: formData.description,
      color: formData.color,
    };

    console.log('New category object:', newCategory);
    const updatedCategories = [...categories, newCategory];
    console.log('Updated categories array:', updatedCategories);

    try {
      await saveCategories(updatedCategories);
      resetForm();
      setShowAddForm(false);
    } catch (error) {
      console.error('Error saving category:', error);
      alert('保存分类失败: ' + error);
    }
  };

  // 更新分类
  const handleUpdateCategory = async () => {
    if (!editingCategory || !formData.name.trim()) return;

    const updatedCategories = categories.map(cat =>
      cat.id === editingCategory.id
        ? {
            ...cat,
            name: formData.name,
            description: formData.description,
            color: formData.color,
          }
        : cat
    );

    await saveCategories(updatedCategories);
    resetForm();
    setEditingCategory(null);
  };

  // 删除分类
  const handleDeleteCategory = async (categoryId: string) => {
    const updatedCategories = categories.filter(cat => cat.id !== categoryId);
    await saveCategories(updatedCategories);
  };

  // 重置表单
  const resetForm = () => {
    setFormData({
      name: '',
      description: '',
      color: '#3b82f6'
    });
  };

  // 开始编辑分类
  const startEditCategory = (category: Category) => {
    setFormData({
      name: category.name,
      description: category.description,
      color: category.color,
    });
    setEditingCategory(category);
  };

  // 处理关闭
  const handleClose = () => {
    resetForm();
    setEditingCategory(null);
    setShowAddForm(false);
    onClose();
  };

  // 处理保存并关闭
  const handleSaveAndClose = () => {
    onSave();
    handleClose();
  };

  useEffect(() => {
    if (isOpen) {
      loadCategories();
    }
  }, [isOpen]);

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[600px] max-h-[80vh] overflow-hidden">
        <DialogHeader>
          <DialogTitle>分类管理</DialogTitle>
          <DialogDescription>
            管理您的网址分类，添加、编辑或删除分类
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col space-y-4 overflow-hidden">
          {/* 添加分类按钮 */}
          <div className="flex justify-end">
            <Button
              onClick={() => setShowAddForm(true)}
              disabled={showAddForm || !!editingCategory}
            >
              <Plus className="h-4 w-4 mr-2" />
              添加分类
            </Button>
          </div>

          {/* 添加/编辑表单 */}
          {(showAddForm || editingCategory) && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">
                  {editingCategory ? '编辑分类' : '添加分类'}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <label className="text-sm font-medium mb-2 block">
                    分类名称 *
                  </label>
                  <Input
                    placeholder="分类名称"
                    value={formData.name}
                    onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                  />
                </div>

                <div>
                  <label className="text-sm font-medium mb-2 block">
                    分类描述
                  </label>
                  <Input
                    placeholder="分类描述（可选）"
                    value={formData.description}
                    onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                  />
                </div>

                <div>
                  <label className="text-sm font-medium mb-2 block">
                    分类颜色
                  </label>
                  <div className="flex space-x-2">
                    {colorOptions.map((color) => (
                      <button
                        key={color}
                        type="button"
                        className={`w-8 h-8 rounded-full border-2 ${
                          formData.color === color ? 'border-gray-800' : 'border-gray-300'
                        }`}
                        style={{ backgroundColor: color }}
                        onClick={() => setFormData(prev => ({ ...prev, color }))}
                      />
                    ))}
                  </div>
                </div>

                <div className="flex space-x-2 pt-2">
                  <Button
                    variant="outline"
                    onClick={() => {
                      resetForm();
                      setShowAddForm(false);
                      setEditingCategory(null);
                    }}
                    disabled={isLoading}
                  >
                    取消
                  </Button>
                  <Button
                    onClick={editingCategory ? handleUpdateCategory : handleAddCategory}
                    disabled={isLoading || !formData.name.trim()}
                  >
                    {isLoading ? '保存中...' : (editingCategory ? '更新' : '添加')}
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}

          {/* 分类列表 */}
          <div className="overflow-y-auto max-h-[400px] space-y-2">
            {categories.map((category) => (
              <Card key={category.id} className="border">
                <CardContent className="p-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-3 flex-1">
                      <div
                        className="w-4 h-4 rounded-full"
                        style={{ backgroundColor: category.color }}
                      />
                      <div className="flex-1">
                        <h4 className="font-medium">{category.name}</h4>
                        {category.description && (
                          <p className="text-sm text-gray-600">{category.description}</p>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center space-x-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => startEditCategory(category)}
                        disabled={showAddForm || !!editingCategory}
                      >
                        <Edit2 className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDeleteCategory(category.id)}
                        disabled={isLoading || showAddForm || !!editingCategory}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          {categories.length === 0 && (
            <div className="text-center py-8 text-gray-500">
              还没有分类，点击"添加分类"创建第一个分类
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            取消
          </Button>
          <Button onClick={handleSaveAndClose}>
            完成
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default CategoryManager;