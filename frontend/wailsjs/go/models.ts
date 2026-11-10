export namespace main {
	
	export class CPUDetail {
	    cores: number;
	    threads: number;
	    l2KB: number;
	    l3KB: number;
	    baseMHz: number;
	    hybrid: boolean;
	
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
	        this.hybrid = source["hybrid"];
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
	export class DiscordUser {
	    loggedIn: boolean;
	    configured: boolean;
	    id: string;
	    name: string;
	    avatar: string;
	    tier: string;
	    owner: boolean;
	    effective: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new DiscordUser(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loggedIn = source["loggedIn"];
	        this.configured = source["configured"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.avatar = source["avatar"];
	        this.tier = source["tier"];
	        this.owner = source["owner"];
	        this.effective = source["effective"];
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
	export class FixCheckResult {
	    problems: string[];
	    healthy: string;
	    always: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new FixCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.problems = source["problems"];
	        this.healthy = source["healthy"];
	        this.always = source["always"];
	        this.error = source["error"];
	    }
	}
	export class FixOutcome {
	    message: string;
	    error: string;
	    lines: string[];
	    noop: boolean;
	    reboot: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FixOutcome(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.error = source["error"];
	        this.lines = source["lines"];
	        this.noop = source["noop"];
	        this.reboot = source["reboot"];
	    }
	}
	export class FixView {
	    id: string;
	    title: string;
	    description: string;
	    icon: string;
	    admin: boolean;
	    reboot: boolean;
	    risk: string;
	    lastRun?: string;
	    canUndo: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FixView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.icon = source["icon"];
	        this.admin = source["admin"];
	        this.reboot = source["reboot"];
	        this.risk = source["risk"];
	        this.lastRun = source["lastRun"];
	        this.canUndo = source["canUndo"];
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
	export class GameEntry {
	    game: string;
	    exe: string;
	    path: string;
	    lastPlayed: string;
	    sessions: number;
	    totalSec: number;
	    avgFps: number;
	    low1Fps: number;
	    low01Fps: number;
	    measured: boolean;
	    sessionAt: string;
	
	    static createFrom(source: any = {}) {
	        return new GameEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game = source["game"];
	        this.exe = source["exe"];
	        this.path = source["path"];
	        this.lastPlayed = source["lastPlayed"];
	        this.sessions = source["sessions"];
	        this.totalSec = source["totalSec"];
	        this.avgFps = source["avgFps"];
	        this.low1Fps = source["low1Fps"];
	        this.low01Fps = source["low01Fps"];
	        this.measured = source["measured"];
	        this.sessionAt = source["sessionAt"];
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
	    path: string;
	    avgFps: number;
	    low1Fps: number;
	    low01Fps: number;
	
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
	        this.path = source["path"];
	        this.avgFps = source["avgFps"];
	        this.low1Fps = source["low1Fps"];
	        this.low01Fps = source["low01Fps"];
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
	export class OnceOutcome {
	    message: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new OnceOutcome(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.error = source["error"];
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
	    exe: string;
	
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
	        this.exe = source["exe"];
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
	export class TweakOutcome {
	    message: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new TweakOutcome(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.error = source["error"];
	    }
	}
	export class NvSetting {
	    id: number;
	    value: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new NvSetting(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.value = source["value"];
	        this.name = source["name"];
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
	    hex?: string;
	    kv?: string;
	    ifExists?: boolean;
	    date?: string;
	    delete?: boolean;
	    ifCurrent?: number;
	    optional?: boolean;
	
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
	        this.hex = source["hex"];
	        this.kv = source["kv"];
	        this.ifExists = source["ifExists"];
	        this.date = source["date"];
	        this.delete = source["delete"];
	        this.ifCurrent = source["ifCurrent"];
	        this.optional = source["optional"];
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
	    planId?: string;
	    nvidia?: NvSetting[];
	    once?: boolean;
	    fix?: boolean;
	    icon?: string;
	    warn?: string;
	    active: boolean;
	    canUndo: boolean;
	    blocked: boolean;
	    disabled: boolean;
	    lastRun?: string;
	    locked: boolean;
	    status: string;
	    detail: string;
	
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
	        this.planId = source["planId"];
	        this.nvidia = this.convertValues(source["nvidia"], NvSetting);
	        this.once = source["once"];
	        this.fix = source["fix"];
	        this.icon = source["icon"];
	        this.warn = source["warn"];
	        this.active = source["active"];
	        this.canUndo = source["canUndo"];
	        this.blocked = source["blocked"];
	        this.disabled = source["disabled"];
	        this.lastRun = source["lastRun"];
	        this.locked = source["locked"];
	        this.status = source["status"];
	        this.detail = source["detail"];
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

