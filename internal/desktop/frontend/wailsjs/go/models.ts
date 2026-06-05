export namespace app {
	
	export class AddCompanyInput {
	    CNPJ: string;
	    Name: string;
	    CertPath: string;
	    Environment: string;
	
	    static createFrom(source: any = {}) {
	        return new AddCompanyInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CNPJ = source["CNPJ"];
	        this.Name = source["Name"];
	        this.CertPath = source["CertPath"];
	        this.Environment = source["Environment"];
	    }
	}
	export class ExportInput {
	    CNPJ: string;
	    Competence: string;
	    Direction: string;
	    OutPath: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CNPJ = source["CNPJ"];
	        this.Competence = source["Competence"];
	        this.Direction = source["Direction"];
	        this.OutPath = source["OutPath"];
	    }
	}
	export class ListInput {
	    CNPJ: string;
	    Competence: string;
	    Direction: string;
	
	    static createFrom(source: any = {}) {
	        return new ListInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CNPJ = source["CNPJ"];
	        this.Competence = source["Competence"];
	        this.Direction = source["Direction"];
	    }
	}
	export class PullInput {
	    CNPJ: string;
	
	    static createFrom(source: any = {}) {
	        return new PullInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CNPJ = source["CNPJ"];
	    }
	}
	export class PullResult {
	    CompanyName: string;
	    CNPJ: string;
	    DocumentsFound: number;
	    EventsFound: number;
	    Errors: number;
	    Duration: number;
	
	    static createFrom(source: any = {}) {
	        return new PullResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CompanyName = source["CompanyName"];
	        this.CNPJ = source["CNPJ"];
	        this.DocumentsFound = source["DocumentsFound"];
	        this.EventsFound = source["EventsFound"];
	        this.Errors = source["Errors"];
	        this.Duration = source["Duration"];
	    }
	}
	export class StatusResult {
	    CompanyName: string;
	    CNPJ: string;
	    Environment: string;
	    LastNSU: number;
	
	    static createFrom(source: any = {}) {
	        return new StatusResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CompanyName = source["CompanyName"];
	        this.CNPJ = source["CNPJ"];
	        this.Environment = source["Environment"];
	        this.LastNSU = source["LastNSU"];
	    }
	}

}

export namespace nfse {
	
	export class Company {
	    ID: string;
	    CNPJ: string;
	    CNPJRoot: string;
	    Name: string;
	    CertPath: string;
	    Environment: string;
	    LastNSU: number;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Company(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.CNPJ = source["CNPJ"];
	        this.CNPJRoot = source["CNPJRoot"];
	        this.Name = source["Name"];
	        this.CertPath = source["CertPath"];
	        this.Environment = source["Environment"];
	        this.LastNSU = source["LastNSU"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
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
	export class Document {
	    ID: string;
	    ChaveAcesso: string;
	    // Go type: time
	    IssueDate: any;
	    Competence: string;
	    PrestadorCNPJ: string;
	    PrestadorName: string;
	    TomadorCNPJ: string;
	    TomadorName: string;
	    IntermediarioCNPJ: string;
	    IntermediarioName: string;
	    ServiceValue: number;
	    ISSValue: number;
	    IRRFValue: number;
	    INSSValue: number;
	    PISValue: number;
	    COFINSValue: number;
	    CSLLValue: number;
	    Status: string;
	    XMLPath: string;
	    RawHash: string;
	    ParseError: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Document(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.ChaveAcesso = source["ChaveAcesso"];
	        this.IssueDate = this.convertValues(source["IssueDate"], null);
	        this.Competence = source["Competence"];
	        this.PrestadorCNPJ = source["PrestadorCNPJ"];
	        this.PrestadorName = source["PrestadorName"];
	        this.TomadorCNPJ = source["TomadorCNPJ"];
	        this.TomadorName = source["TomadorName"];
	        this.IntermediarioCNPJ = source["IntermediarioCNPJ"];
	        this.IntermediarioName = source["IntermediarioName"];
	        this.ServiceValue = source["ServiceValue"];
	        this.ISSValue = source["ISSValue"];
	        this.IRRFValue = source["IRRFValue"];
	        this.INSSValue = source["INSSValue"];
	        this.PISValue = source["PISValue"];
	        this.COFINSValue = source["COFINSValue"];
	        this.CSLLValue = source["CSLLValue"];
	        this.Status = source["Status"];
	        this.XMLPath = source["XMLPath"];
	        this.RawHash = source["RawHash"];
	        this.ParseError = source["ParseError"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
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
	export class CompanyDocument {
	    ID: string;
	    ChaveAcesso: string;
	    // Go type: time
	    IssueDate: any;
	    Competence: string;
	    PrestadorCNPJ: string;
	    PrestadorName: string;
	    TomadorCNPJ: string;
	    TomadorName: string;
	    IntermediarioCNPJ: string;
	    IntermediarioName: string;
	    ServiceValue: number;
	    ISSValue: number;
	    IRRFValue: number;
	    INSSValue: number;
	    PISValue: number;
	    COFINSValue: number;
	    CSLLValue: number;
	    Status: string;
	    XMLPath: string;
	    RawHash: string;
	    ParseError: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	    RelationID: string;
	    CompanyID: string;
	    DocumentID: string;
	    CompanyRole: string;
	    VisibilityReason: string;
	    FirstSeenNSU: number;
	    LastSeenNSU: number;
	    FirstSeenNSUValid: boolean;
	    LastSeenNSUValid: boolean;
	    // Go type: time
	    FirstSyncedAt: any;
	    // Go type: time
	    LastSyncedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new CompanyDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.ChaveAcesso = source["ChaveAcesso"];
	        this.IssueDate = this.convertValues(source["IssueDate"], null);
	        this.Competence = source["Competence"];
	        this.PrestadorCNPJ = source["PrestadorCNPJ"];
	        this.PrestadorName = source["PrestadorName"];
	        this.TomadorCNPJ = source["TomadorCNPJ"];
	        this.TomadorName = source["TomadorName"];
	        this.IntermediarioCNPJ = source["IntermediarioCNPJ"];
	        this.IntermediarioName = source["IntermediarioName"];
	        this.ServiceValue = source["ServiceValue"];
	        this.ISSValue = source["ISSValue"];
	        this.IRRFValue = source["IRRFValue"];
	        this.INSSValue = source["INSSValue"];
	        this.PISValue = source["PISValue"];
	        this.COFINSValue = source["COFINSValue"];
	        this.CSLLValue = source["CSLLValue"];
	        this.Status = source["Status"];
	        this.XMLPath = source["XMLPath"];
	        this.RawHash = source["RawHash"];
	        this.ParseError = source["ParseError"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	        this.RelationID = source["RelationID"];
	        this.CompanyID = source["CompanyID"];
	        this.DocumentID = source["DocumentID"];
	        this.CompanyRole = source["CompanyRole"];
	        this.VisibilityReason = source["VisibilityReason"];
	        this.FirstSeenNSU = source["FirstSeenNSU"];
	        this.LastSeenNSU = source["LastSeenNSU"];
	        this.FirstSeenNSUValid = source["FirstSeenNSUValid"];
	        this.LastSeenNSUValid = source["LastSeenNSUValid"];
	        this.FirstSyncedAt = this.convertValues(source["FirstSyncedAt"], null);
	        this.LastSyncedAt = this.convertValues(source["LastSyncedAt"], null);
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
