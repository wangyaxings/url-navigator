package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	Order       int       `json:"order"`
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

	// Calculate next order (highest order + 1)
	order := 0
	for _, existingURL := range urls {
		if existingURL.Order >= order {
			order = existingURL.Order + 1
		}
	}

	newURL := URLItem{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Title:       title,
		URL:         url,
		Description: description,
		Category:    category,
		Tags:        tags,
		Order:       order,
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

// AddCategory adds a new category
func (a *App) AddCategory(name, description, color string) (*Category, error) {
	categories, err := a.GetCategories()
	if err != nil {
		return nil, err
	}

	newCategory := Category{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Name:        name,
		Description: description,
		Color:       color,
	}

	categories = append(categories, newCategory)
	err = a.SaveCategories(categories)
	if err != nil {
		return nil, err
	}

	return &newCategory, nil
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

// ReorderURLs updates the order of URLs based on new positions
func (a *App) ReorderURLs(urlIDs []string) error {
	urls, err := a.GetURLs()
	if err != nil {
		return err
	}

	// Create a map for quick lookup
	urlMap := make(map[string]*URLItem)
	for i := range urls {
		urlMap[urls[i].ID] = &urls[i]
	}

	// Update orders based on new positions
	for newOrder, urlID := range urlIDs {
		if url, exists := urlMap[urlID]; exists {
			url.Order = newOrder
			url.UpdatedAt = time.Now()
		}
	}

	return a.SaveURLs(urls)
}

// AdvancedSearchOptions represents advanced search parameters
type AdvancedSearchOptions struct {
	Query     string   `json:"query"`
	Category  string   `json:"category"`
	Tags      []string `json:"tags"`
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	SortBy    string   `json:"sortBy"`    // title, date, category, frequency
	SearchIn  []string `json:"searchIn"` // title, description, url
}

// AdvancedSearchURLs performs advanced search with multiple criteria
func (a *App) AdvancedSearchURLs(options AdvancedSearchOptions) ([]URLItem, error) {
	urls, err := a.GetURLs()
	if err != nil {
		return nil, err
	}

	var filteredUrls []URLItem

	for _, url := range urls {
		if a.matchesAdvancedCriteria(url, options) {
			filteredUrls = append(filteredUrls, url)
		}
	}

	// Sort results
	a.sortURLs(filteredUrls, options.SortBy)

	return filteredUrls, nil
}

// matchesAdvancedCriteria checks if a URL matches the advanced search criteria
func (a *App) matchesAdvancedCriteria(url URLItem, options AdvancedSearchOptions) bool {
	// Check category filter
	if options.Category != "" && options.Category != "all" && url.Category != options.Category {
		return false
	}

	// Check tags filter
	if len(options.Tags) > 0 {
		hasMatchingTag := false
		for _, searchTag := range options.Tags {
			for _, urlTag := range url.Tags {
				if strings.Contains(strings.ToLower(urlTag), strings.ToLower(searchTag)) {
					hasMatchingTag = true
					break
				}
			}
			if hasMatchingTag {
				break
			}
		}
		if !hasMatchingTag {
			return false
		}
	}

	// Check date range
	if options.StartDate != "" || options.EndDate != "" {
		urlDate := url.CreatedAt.Format("2006-01-02")

		if options.StartDate != "" && urlDate < options.StartDate {
			return false
		}

		if options.EndDate != "" && urlDate > options.EndDate {
			return false
		}
	}

	// Check search query in specified fields
	if options.Query != "" {
		searchFields := options.SearchIn
		if len(searchFields) == 0 {
			// Default to all fields if none specified
			searchFields = []string{"title", "description", "url"}
		}

		queryLower := strings.ToLower(options.Query)
		found := false

		for _, field := range searchFields {
			var searchText string
			switch field {
			case "title":
				searchText = strings.ToLower(url.Title)
			case "description":
				searchText = strings.ToLower(url.Description)
			case "url":
				searchText = strings.ToLower(url.URL)
			}

			if strings.Contains(searchText, queryLower) {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

// sortURLs sorts URLs based on the specified criteria
func (a *App) sortURLs(urls []URLItem, sortBy string) {
	switch sortBy {
	case "title":
		for i := 0; i < len(urls)-1; i++ {
			for j := i + 1; j < len(urls); j++ {
				if strings.ToLower(urls[i].Title) > strings.ToLower(urls[j].Title) {
					urls[i], urls[j] = urls[j], urls[i]
				}
			}
		}
	case "date":
		for i := 0; i < len(urls)-1; i++ {
			for j := i + 1; j < len(urls); j++ {
				if urls[i].CreatedAt.Before(urls[j].CreatedAt) {
					urls[i], urls[j] = urls[j], urls[i]
				}
			}
		}
	case "category":
		for i := 0; i < len(urls)-1; i++ {
			for j := i + 1; j < len(urls); j++ {
				if strings.ToLower(urls[i].Category) > strings.ToLower(urls[j].Category) {
					urls[i], urls[j] = urls[j], urls[i]
				}
			}
		}
	}
}

// ExportBookmarks exports all bookmarks to JSON format
func (a *App) ExportBookmarks() (string, error) {
	urls, err := a.GetURLs()
	if err != nil {
		return "", err
	}

	categories, err := a.GetCategories()
	if err != nil {
		return "", err
	}

	// 获取当前版本信息
	currentVersion := "unknown"
	if RuntimeVersion != nil && RuntimeVersion.Version != "" {
		currentVersion = RuntimeVersion.Version
	}

	exportData := map[string]interface{}{
		"bookmarks":  urls,
		"categories": categories,
		"exportedAt": time.Now(),
		"version":    currentVersion,
	}

	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// ChromeBookmark represents Chrome bookmark structure
type ChromeBookmark struct {
	DateAdded    string            `json:"date_added"`
	DateModified string            `json:"date_modified"`
	GUID         string            `json:"guid"`
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	URL          string            `json:"url"`
	Children     []ChromeBookmark  `json:"children,omitempty"`
}

// ChromeBookmarkRoot represents the root structure of Chrome bookmarks
type ChromeBookmarkRoot struct {
	Checksum string `json:"checksum"`
	Roots    struct {
		BookmarkBar ChromeBookmark `json:"bookmark_bar"`
		Other       ChromeBookmark `json:"other"`
		Synced      ChromeBookmark `json:"synced"`
	} `json:"roots"`
	Version int `json:"version"`
}

// ImportChromeBookmarks imports bookmarks from Chrome JSON format
func (a *App) ImportChromeBookmarks(jsonData string) (int, error) {
	var chromeData ChromeBookmarkRoot
	if err := json.Unmarshal([]byte(jsonData), &chromeData); err != nil {
		return 0, err
	}

	var importedCount int
	defaultCategory := "导入"

	// Ensure default category exists
	categories, _ := a.GetCategories()
	hasDefaultCategory := false
	for _, cat := range categories {
		if cat.Name == defaultCategory {
			hasDefaultCategory = true
			break
		}
	}

	if !hasDefaultCategory {
		a.AddCategory(defaultCategory, "从浏览器导入的书签", "#6366f1")
	}

	// Parse bookmark bar
	if err := a.parseChromeBookmarks(chromeData.Roots.BookmarkBar.Children, defaultCategory, &importedCount); err != nil {
		return importedCount, err
	}

	// Parse other bookmarks
	if err := a.parseChromeBookmarks(chromeData.Roots.Other.Children, defaultCategory, &importedCount); err != nil {
		return importedCount, err
	}

	return importedCount, nil
}

// parseChromeBookmarks recursively parses Chrome bookmark structure
func (a *App) parseChromeBookmarks(bookmarks []ChromeBookmark, category string, count *int) error {
	for _, bookmark := range bookmarks {
		if bookmark.Type == "url" && bookmark.URL != "" {
			_, err := a.AddURL(bookmark.Name, bookmark.URL, "", category, []string{})
			if err != nil {
				continue // Skip invalid bookmarks
			}
			*count++
		} else if bookmark.Type == "folder" && len(bookmark.Children) > 0 {
			// Use folder name as category or subcategory info
			folderCategory := category
			if bookmark.Name != "" {
				folderCategory = bookmark.Name
			}
			a.parseChromeBookmarks(bookmark.Children, folderCategory, count)
		}
	}
	return nil
}

// ImportNetscapeBookmarks imports bookmarks from Netscape HTML format
func (a *App) ImportNetscapeBookmarks(htmlData string) (int, error) {
	var importedCount int
	defaultCategory := "导入"

	// Ensure default category exists
	categories, _ := a.GetCategories()
	hasDefaultCategory := false
	for _, cat := range categories {
		if cat.Name == defaultCategory {
			hasDefaultCategory = true
			break
		}
	}

	if !hasDefaultCategory {
		a.AddCategory(defaultCategory, "从浏览器导入的书签", "#6366f1")
	}

	// Parse HTML bookmarks using regex
	linkRegex := regexp.MustCompile(`<A[^>]+HREF="([^"]+)"[^>]*>([^<]+)</A>`)
	matches := linkRegex.FindAllStringSubmatch(htmlData, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			url := match[1]
			title := match[2]

			if url != "" && title != "" {
				_, err := a.AddURL(title, url, "", defaultCategory, []string{})
				if err != nil {
					continue // Skip invalid bookmarks
				}
				importedCount++
			}
		}
	}

	return importedCount, nil
}

