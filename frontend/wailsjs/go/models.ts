export namespace main {
	
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

