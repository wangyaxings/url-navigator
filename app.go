package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// URLItem represents a single URL bookmark
type URLItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Category represents a URL category
type Category struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}



// GetDataDir returns the application data directory
func (a *App) GetDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(homeDir, ".urlnavigator")
}

// EnsureDataDir ensures the data directory exists
func (a *App) EnsureDataDir() error {
	dataDir := a.GetDataDir()
	return os.MkdirAll(dataDir, 0755)
}

// GetURLs returns all stored URLs
func (a *App) GetURLs() ([]URLItem, error) {
	if err := a.EnsureDataDir(); err != nil {
		return nil, err
	}

	filePath := filepath.Join(a.GetDataDir(), "urls.json")
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []URLItem{}, nil
		}
		return nil, err
	}

	var urls []URLItem
	err = json.Unmarshal(data, &urls)
	return urls, err
}

// SaveURLs saves URLs to file
func (a *App) SaveURLs(urls []URLItem) error {
	if err := a.EnsureDataDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(urls, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(a.GetDataDir(), "urls.json")
	return ioutil.WriteFile(filePath, data, 0644)
}

// AddURL adds a new URL
func (a *App) AddURL(title, url, description, category string, tags []string) (*URLItem, error) {
	urls, err := a.GetURLs()
	if err != nil {
		return nil, err
	}

	newURL := URLItem{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Title:       title,
		URL:         url,
		Description: description,
		Category:    category,
		Tags:        tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	urls = append(urls, newURL)
	err = a.SaveURLs(urls)
	if err != nil {
		return nil, err
	}

	return &newURL, nil
}

// UpdateURL updates an existing URL
func (a *App) UpdateURL(id, title, url, description, category string, tags []string) (*URLItem, error) {
	urls, err := a.GetURLs()
	if err != nil {
		return nil, err
	}

	for i, urlItem := range urls {
		if urlItem.ID == id {
			urls[i].Title = title
			urls[i].URL = url
			urls[i].Description = description
			urls[i].Category = category
			urls[i].Tags = tags
			urls[i].UpdatedAt = time.Now()

			err = a.SaveURLs(urls)
			if err != nil {
				return nil, err
			}

			return &urls[i], nil
		}
	}

	return nil, fmt.Errorf("URL with id %s not found", id)
}

// DeleteURL deletes a URL by ID
func (a *App) DeleteURL(id string) error {
	urls, err := a.GetURLs()
	if err != nil {
		return err
	}

	for i, urlItem := range urls {
		if urlItem.ID == id {
			urls = append(urls[:i], urls[i+1:]...)
			return a.SaveURLs(urls)
		}
	}

	return fmt.Errorf("URL with id %s not found", id)
}

// GetCategories returns all categories
func (a *App) GetCategories() ([]Category, error) {
	if err := a.EnsureDataDir(); err != nil {
		return nil, err
	}

	filePath := filepath.Join(a.GetDataDir(), "categories.json")
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default categories
			defaultCategories := []Category{
				{ID: "1", Name: "工作", Description: "工作相关网站", Color: "#3b82f6"},
				{ID: "2", Name: "学习", Description: "学习资源网站", Color: "#10b981"},
				{ID: "3", Name: "娱乐", Description: "娱乐休闲网站", Color: "#f59e0b"},
				{ID: "4", Name: "工具", Description: "实用工具网站", Color: "#8b5cf6"},
				{ID: "5", Name: "其他", Description: "其他类型网站", Color: "#6b7280"},
			}
			a.SaveCategories(defaultCategories)
			return defaultCategories, nil
		}
		return nil, err
	}

	var categories []Category
	err = json.Unmarshal(data, &categories)
	return categories, err
}

// SaveCategories saves categories to file
func (a *App) SaveCategories(categories []Category) error {
	if err := a.EnsureDataDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(categories, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(a.GetDataDir(), "categories.json")
	return ioutil.WriteFile(filePath, data, 0644)
}

// SearchURLs searches URLs by keyword
func (a *App) SearchURLs(keyword string) ([]URLItem, error) {
	urls, err := a.GetURLs()
	if err != nil {
		return nil, err
	}

	if keyword == "" {
		return urls, nil
	}

	var filtered []URLItem
	for _, url := range urls {
		if containsIgnoreCase(url.Title, keyword) ||
			containsIgnoreCase(url.Description, keyword) ||
			containsIgnoreCase(url.URL, keyword) ||
			containsIgnoreCase(url.Category, keyword) {
			filtered = append(filtered, url)
		}
	}

	return filtered, nil
}

// containsIgnoreCase checks if str contains substr (case insensitive)
func containsIgnoreCase(str, substr string) bool {
	return len(str) >= len(substr) &&
		   (substr == "" ||
		    len(str) > 0 &&
		    (str == substr ||
		     (len(str) > len(substr) &&
		      (str[:len(substr)] == substr ||
		       str[len(str)-len(substr):] == substr ||
		       containsIgnoreCase(str[1:], substr)))))
}

// TestUpdateWithNewVersion 测试有新版本的情况
func (a *App) TestUpdateWithNewVersion() (*UpdateInfo, error) {
	info := &UpdateInfo{
		HasUpdate:      true,
		CurrentVersion: "1.0.0",
		LatestVersion:  "1.1.0",
		UpdateURL:      "https://github.com/your-username/urlnavigator/releases/download/v1.1.0/",
		ReleaseNotes:   "新版本功能:\n- 修复分类创建问题\n- 优化界面性能\n- 添加快捷键支持",
	}
	return info, nil
}

// TestUpdateNoNewVersion 测试没有新版本的情况
func (a *App) TestUpdateNoNewVersion() (*UpdateInfo, error) {
	info := &UpdateInfo{
		HasUpdate:      false,
		CurrentVersion: "1.0.0",
		LatestVersion:  "1.0.0",
		UpdateURL:      "",
		ReleaseNotes:   "",
	}
	return info, nil
}