export namespace main {
	
	export class CPUDetail {
	    cores: number;
	    threads: number;
	    l2KB: number;
	    l3KB: number;
	    baseMHz: number;
	
	    static createFrom(source: any = {}) {
	        return new CPUDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cores = source["cores"];
	        this.threads = source["threads"];
	        this.l2KB = source["l2KB"];
	        this.l3KB = source["l3KB"];
	        this.baseMHz = source["baseMHz"];
	    }
	}
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
	export class DiskInfo {
	    letter: string;
	    totalGB: number;
	    freeGB: number;
	    usedPct: number;
	
	    static createFrom(source: any = {}) {
	        return new DiskInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.letter = source["letter"];
	        this.totalGB = source["totalGB"];
	        this.freeGB = source["freeGB"];
	        this.usedPct = source["usedPct"];
	    }
	}
	export class GPULive {
	    name: string;
	    temp: number;
	    util: number;
	    vramUsed: number;
	    vramTotal: number;
	    clock: number;
	    clockMax: number;
	    power: number;
	    fan: number;
	    driver: string;
	
	    static createFrom(source: any = {}) {
	        return new GPULive(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.temp = source["temp"];
	        this.util = source["util"];
	        this.vramUsed = source["vramUsed"];
	        this.vramTotal = source["vramTotal"];
	        this.clock = source["clock"];
	        this.clockMax = source["clockMax"];
	        this.power = source["power"];
	        this.fan = source["fan"];
	        this.driver = source["driver"];
	    }
	}
	export class GameSession {
	    game: string;
	    exe: string;
	    start: string;
	    durationSec: number;
	    avgCpu: number;
	    maxCpu: number;
	    avgRam: number;
	    avgGpu: number;
	    avgGpuTemp: number;
	    maxGpuTemp: number;
	    samples: number;
	
	    static createFrom(source: any = {}) {
	        return new GameSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game = source["game"];
	        this.exe = source["exe"];
	        this.start = source["start"];
	        this.durationSec = source["durationSec"];
	        this.avgCpu = source["avgCpu"];
	        this.maxCpu = source["maxCpu"];
	        this.avgRam = source["avgRam"];
	        this.avgGpu = source["avgGpu"];
	        this.avgGpuTemp = source["avgGpuTemp"];
	        this.maxGpuTemp = source["maxGpuTemp"];
	        this.samples = source["samples"];
	    }
	}
	export class Live {
	    cpu: number;
	    ram: number;
	    ramUsedGB: number;
	    ramTotalGB: number;
	    cpuMHz: number;
	    diskBusy: number;
	    disks: DiskInfo[];
	    gpu?: GPULive;
	
	    static createFrom(source: any = {}) {
	        return new Live(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpu = source["cpu"];
	        this.ram = source["ram"];
	        this.ramUsedGB = source["ramUsedGB"];
	        this.ramTotalGB = source["ramTotalGB"];
	        this.cpuMHz = source["cpuMHz"];
	        this.diskBusy = source["diskBusy"];
	        this.disks = this.convertValues(source["disks"], DiskInfo);
	        this.gpu = this.convertValues(source["gpu"], GPULive);
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
	    user: string;
	    cpuInfo: CPUDetail;
	
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
	        this.user = source["user"];
	        this.cpuInfo = this.convertValues(source["cpuInfo"], CPUDetail);
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

