export namespace main {
	
	export class AdvancedSearchOptions {
	    query: string;
	    category: string;
	    tags: string[];
	    startDate: string;
	    endDate: string;
	    sortBy: string;
	    searchIn: string[];
	
	    static createFrom(source: any = {}) {
	        return new AdvancedSearchOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.category = source["category"];
	        this.tags = source["tags"];
	        this.startDate = source["startDate"];
	        this.endDate = source["endDate"];
	        this.sortBy = source["sortBy"];
	        this.searchIn = source["searchIn"];
	    }
	}
	export class Category {
	    id: string;
	    name: string;
	    description: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.color = source["color"];
	    }
	}
	export class URLItem {
	    id: string;
	    title: string;
	    url: string;
	    description: string;
	    category: string;
	    tags: string[];
	    order: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new URLItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.tags = source["tags"];
	        this.order = source["order"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateInfo {
	    hasUpdate: boolean;
	    currentVersion: string;
	    latestVersion: string;
	    updateUrl: string;
	    releaseNotes: string;
	    errorMessage?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasUpdate = source["hasUpdate"];
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.updateUrl = source["updateUrl"];
	        this.releaseNotes = source["releaseNotes"];
	        this.errorMessage = source["errorMessage"];
	    }
	}
	export class UpdateProgress {
	    phase: string;
	    progress: number;
	    speed: string;
	    eta: string;
	    downloaded: number;
	    total: number;
	    message: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.progress = source["progress"];
	        this.speed = source["speed"];
	        this.eta = source["eta"];
	        this.downloaded = source["downloaded"];
	        this.total = source["total"];
	        this.message = source["message"];
	        this.error = source["error"];
	    }
	}
	export class VersionInfo {
	    version: string;
	    github_owner: string;
	    github_repo: string;
	    app_name: string;
	    source: string;
	    is_default: boolean;
	
	    static createFrom(source: any = {}) {
	        return new VersionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.github_owner = source["github_owner"];
	        this.github_repo = source["github_repo"];
	        this.app_name = source["app_name"];
	        this.source = source["source"];
	        this.is_default = source["is_default"];
	    }
	}

}

