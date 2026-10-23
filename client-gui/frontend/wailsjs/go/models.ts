export namespace core {
	
	export class Expert {
	    expert_id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Expert(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expert_id = source["expert_id"];
	        this.name = source["name"];
	    }
	}
	export class FileOp {
	    op: string;
	    path: string;
	    to: string;
	    body: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new FileOp(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.op = source["op"];
	        this.path = source["path"];
	        this.to = source["to"];
	        this.body = source["body"];
	        this.reason = source["reason"];
	    }
	}

}

export namespace main {
	
	export class OpResult2 {
	    op: string;
	    path: string;
	    status: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new OpResult2(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.op = source["op"];
	        this.path = source["path"];
	        this.status = source["status"];
	        this.detail = source["detail"];
	    }
	}
	export class ApplyResult {
	    ok: boolean;
	    results: OpResult2[];
	    feedback: string;
	
	    static createFrom(source: any = {}) {
	        return new ApplyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.results = this.convertValues(source["results"], OpResult2);
	        this.feedback = source["feedback"];
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
	export class AskResult {
	    ok: boolean;
	    msg: string;
	    answer: string;
	    thread: string;
	    ops: core.FileOp[];
	
	    static createFrom(source: any = {}) {
	        return new AskResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.msg = source["msg"];
	        this.answer = source["answer"];
	        this.thread = source["thread"];
	        this.ops = this.convertValues(source["ops"], core.FileOp);
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
	export class LoginResult {
	    ok: boolean;
	    role: string;
	    msg: string;
	    experts: core.Expert[];
	    usage: string;
	
	    static createFrom(source: any = {}) {
	        return new LoginResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.role = source["role"];
	        this.msg = source["msg"];
	        this.experts = this.convertValues(source["experts"], core.Expert);
	        this.usage = source["usage"];
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

