// appService.ts - 更新的应用服务，支持新的版本管理功能

import {
    GetCurrentVersion,
    GetVersionInfo,
    UpdateVersionConfig,
    ForceReloadVersion,
    GetVersionFromWails
  } from '../../wailsjs/go/main/App';

  export interface VersionInfo {
    version: string;
    github_owner: string;
    github_repo: string;
    app_name: string;
  }

  export class AppService {
    /**
     * 获取当前版本号（确保有v前缀）
     * @returns Promise<string> 版本号，如 "v1.3.0"
     */
    static async GetCurrentVersion(): Promise<string> {
      try {
        const version = await GetCurrentVersion();
        return version || 'unknown';
      } catch (error) {
        console.error('Failed to get current version:', error);
        return 'unknown';
      }
    }

    /**
     * 获取完整版本信息
     * @returns Promise<VersionInfo> 完整的版本信息对象
     */
    static async GetVersionInfo(): Promise<VersionInfo> {
      try {
        const versionInfo = await GetVersionInfo();
        return versionInfo || {
          version: 'unknown',
          github_owner: 'wangyaxings',
          github_repo: 'url-navigator',
          app_name: 'URLNavigator'
        };
      } catch (error) {
        console.error('Failed to get version info:', error);
        return {
          version: 'unknown',
          github_owner: 'wangyaxings',
          github_repo: 'url-navigator',
          app_name: 'URLNavigator'
        };
      }
    }

    /**
     * 更新版本配置中的GitHub信息
     * @param owner GitHub用户名
     * @param repo GitHub仓库名
     * @returns Promise<boolean> 是否更新成功
     */
    static async UpdateVersionConfig(owner: string, repo: string): Promise<boolean> {
      try {
        await UpdateVersionConfig(owner, repo);
        return true;
      } catch (error) {
        console.error('Failed to update version config:', error);
        return false;
      }
    }

    /**
     * 强制重新加载版本信息（用于调试或刷新）
     * @returns Promise<boolean> 是否重新加载成功
     */
    static async ForceReloadVersion(): Promise<boolean> {
      try {
        await ForceReloadVersion();
        return true;
      } catch (error) {
        console.error('Failed to force reload version:', error);
        return false;
      }
    }

    /**
     * 直接从wails.json获取版本信息（调试用）
     * @returns Promise<string> 从wails.json读取的版本号
     */
    static async GetVersionFromWails(): Promise<string> {
      try {
        const version = await GetVersionFromWails();
        return version || 'unknown';
      } catch (error) {
        console.error('Failed to get version from wails.json:', error);
        return 'unknown';
      }
    }

    /**
     * 检查版本信息是否可用
     * @returns Promise<boolean> 版本信息是否有效
     */
    static async IsVersionAvailable(): Promise<boolean> {
      try {
        const version = await this.GetCurrentVersion();
        return version !== 'unknown' && version !== '';
      } catch (error) {
        return false;
      }
    }

    /**
     * 获取格式化的版本显示文本
     * @returns Promise<string> 格式化的版本文本
     */
    static async GetFormattedVersion(): Promise<string> {
      try {
        const version = await this.GetCurrentVersion();
        if (version === 'unknown' || version === '') {
          return '版本未知';
        }

        // 确保版本号有v前缀
        const formattedVersion = version.startsWith('v') ? version : `v${version}`;
        return formattedVersion;
      } catch (error) {
        console.error('Failed to get formatted version:', error);
        return '版本获取失败';
      }
    }

    /**
     * 获取GitHub链接
     * @returns Promise<string> GitHub仓库链接
     */
    static async GetGitHubUrl(): Promise<string> {
      try {
        const versionInfo = await this.GetVersionInfo();
        return `https://github.com/${versionInfo.github_owner}/${versionInfo.github_repo}`;
      } catch (error) {
        console.error('Failed to get GitHub URL:', error);
        return 'https://github.com/wangyaxings/url-navigator';
      }
    }

    /**
     * 获取GitHub Releases链接
     * @returns Promise<string> GitHub Releases页面链接
     */
    static async GetReleasesUrl(): Promise<string> {
      try {
        const versionInfo = await this.GetVersionInfo();
        return `https://github.com/${versionInfo.github_owner}/${versionInfo.github_repo}/releases`;
      } catch (error) {
        console.error('Failed to get releases URL:', error);
        return 'https://github.com/wangyaxings/url-navigator/releases';
      }
    }

    /**
     * 检查是否为开发版本
     * @returns Promise<boolean> 是否为开发版本
     */
    static async IsDevelopmentVersion(): Promise<boolean> {
      try {
        const version = await this.GetCurrentVersion();
        return version === 'unknown' || version.includes('dev') || version.includes('development');
      } catch (error) {
        return true; // 出错时假设为开发版本
      }
    }

    /**
     * 获取版本调试信息（仅在开发模式下使用）
     * @returns Promise<object> 版本调试信息
     */
    static async GetVersionDebugInfo(): Promise<{
      currentVersion: string;
      versionInfo: VersionInfo;
      wailsVersion: string;
      isAvailable: boolean;
      isDevelopment: boolean;
    }> {
      try {
        const [currentVersion, versionInfo, wailsVersion, isAvailable, isDevelopment] = await Promise.all([
          this.GetCurrentVersion(),
          this.GetVersionInfo(),
          this.GetVersionFromWails(),
          this.IsVersionAvailable(),
          this.IsDevelopmentVersion()
        ]);

        return {
          currentVersion,
          versionInfo,
          wailsVersion,
          isAvailable,
          isDevelopment
        };
      } catch (error) {
        console.error('Failed to get version debug info:', error);
        throw error;
      }
    }
  }

  // 导出默认实例
  export default AppService;