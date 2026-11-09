export namespace main {
	
	export class CleanItem {
	    id: string;
	    title: string;
	    desc: string;
	    bytes: number;
	    files: number;
	    admin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CleanItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.desc = source["desc"];
	        this.bytes = source["bytes"];
	        this.files = source["files"];
	        this.admin = source["admin"];
	    }
	}
	export class CleanOutcome {
	    freed: number;
	    errors: string[];
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new CleanOutcome(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.freed = source["freed"];
	        this.errors = source["errors"];
	        this.error = source["error"];
	    }
	}
	export class Live {
	    cpu: number;
	    ram: number;
	
	    static createFrom(source: any = {}) {
	        return new Live(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpu = source["cpu"];
	        this.ram = source["ram"];
	    }
	}
	export class Profile {
	    id: string;
	    title: string;
	    description: string;
	    tweaks: string[];
	    needsAdmin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.tweaks = source["tweaks"];
	        this.needsAdmin = source["needsAdmin"];
	    }
	}
	export class ProfileOutcome {
	    applied: string[];
	    skipped: string[];
	    failed: string[];
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ProfileOutcome(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.applied = source["applied"];
	        this.skipped = source["skipped"];
	        this.failed = source["failed"];
	        this.error = source["error"];
	    }
	}
	export class StartupEntry {
	    id: string;
	    name: string;
	    command: string;
	    scope: string;
	    label: string;
	    enabled: boolean;
	    admin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StartupEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.command = source["command"];
	        this.scope = source["scope"];
	        this.label = source["label"];
	        this.enabled = source["enabled"];
	        this.admin = source["admin"];
	    }
	}
	export class SysInfo {
	    cpu: string;
	    cores: number;
	    gpus: string[];
	    ramGB: number;
	    windows: string;
	    admin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SysInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpu = source["cpu"];
	        this.cores = source["cores"];
	        this.gpus = source["gpus"];
	        this.ramGB = source["ramGB"];
	        this.windows = source["windows"];
	        this.admin = source["admin"];
	    }
	}
	export class RegChange {
	    root: string;
	    path: string;
	    name: string;
	    kind: string;
	    d?: number;
	    s?: string;
	    mustExist?: boolean;
	    ramKB?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RegChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.d = source["d"];
	        this.s = source["s"];
	        this.mustExist = source["mustExist"];
	        this.ramKB = source["ramKB"];
	    }
	}
	export class TweakView {
	    id: string;
	    title: string;
	    description: string;
	    category: string;
	    admin: boolean;
	    reboot: boolean;
	    power: boolean;
	    vendor?: string;
	    risk: string;
	    minBuild?: number;
	    maxBuild?: number;
	    tasks?: string[];
	    changes: RegChange[];
	    active: boolean;
	    canUndo: boolean;
	    blocked: boolean;
	    disabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TweakView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.admin = source["admin"];
	        this.reboot = source["reboot"];
	        this.power = source["power"];
	        this.vendor = source["vendor"];
	        this.risk = source["risk"];
	        this.minBuild = source["minBuild"];
	        this.maxBuild = source["maxBuild"];
	        this.tasks = source["tasks"];
	        this.changes = this.convertValues(source["changes"], RegChange);
	        this.active = source["active"];
	        this.canUndo = source["canUndo"];
	        this.blocked = source["blocked"];
	        this.disabled = source["disabled"];
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

}

