export namespace app {
	
	export class AddCompanyInput {
	    CNPJ: string;
	    Name: string;
	    CredentialID: string;
	    CredentialLabel: string;
	    CertPath: string;
	    Environment: string;
	    SyncStartPolicy: string;
	    SyncStartDate: string;
	
	    static createFrom(source: any = {}) {
	        return new AddCompanyInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CNPJ = source["CNPJ"];
	        this.Name = source["Name"];
	        this.CredentialID = source["CredentialID"];
	        this.CredentialLabel = source["CredentialLabel"];
	        this.CertPath = source["CertPath"];
	        this.Environment = source["Environment"];
	        this.SyncStartPolicy = source["SyncStartPolicy"];
	        this.SyncStartDate = source["SyncStartDate"];
	    }
	}
	export class AddCredentialInput {
	    Label: string;
	    CertPath: string;
	
	    static createFrom(source: any = {}) {
	        return new AddCredentialInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Label = source["Label"];
	        this.CertPath = source["CertPath"];
	    }
	}
	export class AssignCredentialInput {
	    CompanyCNPJ: string;
	    CredentialID: string;
	
	    static createFrom(source: any = {}) {
	        return new AssignCredentialInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CompanyCNPJ = source["CompanyCNPJ"];
	        this.CredentialID = source["CredentialID"];
	    }
	}
	export class ListInput {
	    CNPJ: string;
	    Competence: string;
	    Direction: string;
	    OnlyUnread: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ListInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CNPJ = source["CNPJ"];
	        this.Competence = source["Competence"];
	        this.Direction = source["Direction"];
	        this.OnlyUnread = source["OnlyUnread"];
	    }
	}
	export class PullInput {
	    CNPJ: string;
	    Mode: string;
	
	    static createFrom(source: any = {}) {
	        return new PullInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CNPJ = source["CNPJ"];
	        this.Mode = source["Mode"];
	    }
	}
	export class PullResult {
	    CompanyName: string;
	    CNPJ: string;
	    CredentialLabel: string;
	    CredentialCNPJ: string;
	    ConsultationBasis: string;
	    Status: string;
	    StopReason: string;
	    LastProcessedNSU: number;
	    LastFoundNSU?: number;
	    EmptyStreak: number;
	    DocumentsFound: number;
	    EventsFound: number;
	    DocumentsSaved: number;
	    EventsSaved: number;
	    DocumentsSkippedByPolicy: number;
	    EventsSkippedByPolicy: number;
	    Errors: number;
	    Duration: number;
	
	    static createFrom(source: any = {}) {
	        return new PullResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CompanyName = source["CompanyName"];
	        this.CNPJ = source["CNPJ"];
	        this.CredentialLabel = source["CredentialLabel"];
	        this.CredentialCNPJ = source["CredentialCNPJ"];
	        this.ConsultationBasis = source["ConsultationBasis"];
	        this.Status = source["Status"];
	        this.StopReason = source["StopReason"];
	        this.LastProcessedNSU = source["LastProcessedNSU"];
	        this.LastFoundNSU = source["LastFoundNSU"];
	        this.EmptyStreak = source["EmptyStreak"];
	        this.DocumentsFound = source["DocumentsFound"];
	        this.EventsFound = source["EventsFound"];
	        this.DocumentsSaved = source["DocumentsSaved"];
	        this.EventsSaved = source["EventsSaved"];
	        this.DocumentsSkippedByPolicy = source["DocumentsSkippedByPolicy"];
	        this.EventsSkippedByPolicy = source["EventsSkippedByPolicy"];
	        this.Errors = source["Errors"];
	        this.Duration = source["Duration"];
	    }
	}
	export class QueryNFSeInput {
	    CNPJ: string;
	    ChaveAcesso: string;
	
	    static createFrom(source: any = {}) {
	        return new QueryNFSeInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CNPJ = source["CNPJ"];
	        this.ChaveAcesso = source["ChaveAcesso"];
	    }
	}
	export class ResetSyncInput {
	    CNPJ: string;
	
	    static createFrom(source: any = {}) {
	        return new ResetSyncInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CNPJ = source["CNPJ"];
	    }
	}
	export class StatusResult {
	    CompanyName: string;
	    CNPJ: string;
	    Environment: string;
	    ConsultationCNPJ: string;
	    CredentialCNPJ: string;
	    // Go type: time
	    CredentialNotAfter?: any;
	    LastProcessedNSU: number;
	    LastFoundNSU?: number;
	    // Go type: time
	    LastSyncAt?: any;
	    LastRunStatus: string;
	    LastRunStopReason: string;
	    TotalEmitidas: number;
	    TotalTomadas: number;
	
	    static createFrom(source: any = {}) {
	        return new StatusResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CompanyName = source["CompanyName"];
	        this.CNPJ = source["CNPJ"];
	        this.Environment = source["Environment"];
	        this.ConsultationCNPJ = source["ConsultationCNPJ"];
	        this.CredentialCNPJ = source["CredentialCNPJ"];
	        this.CredentialNotAfter = this.convertValues(source["CredentialNotAfter"], null);
	        this.LastProcessedNSU = source["LastProcessedNSU"];
	        this.LastFoundNSU = source["LastFoundNSU"];
	        this.LastSyncAt = this.convertValues(source["LastSyncAt"], null);
	        this.LastRunStatus = source["LastRunStatus"];
	        this.LastRunStopReason = source["LastRunStopReason"];
	        this.TotalEmitidas = source["TotalEmitidas"];
	        this.TotalTomadas = source["TotalTomadas"];
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
	export class UpdateCompanyInput {
	    CNPJ: string;
	    Name: string;
	    Environment: string;
	    SyncStartPolicy: string;
	    SyncStartDate: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCompanyInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CNPJ = source["CNPJ"];
	        this.Name = source["Name"];
	        this.Environment = source["Environment"];
	        this.SyncStartPolicy = source["SyncStartPolicy"];
	        this.SyncStartDate = source["SyncStartDate"];
	    }
	}
	export class UpdateCredentialDataInput {
	    CredentialID: string;
	    Label: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCredentialDataInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CredentialID = source["CredentialID"];
	        this.Label = source["Label"];
	    }
	}
	export class UpdateCredentialPathInput {
	    CredentialID: string;
	    CertPath: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCredentialPathInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CredentialID = source["CredentialID"];
	        this.CertPath = source["CertPath"];
	    }
	}

}

export namespace desktopapi {
	
	export class BuildInfo {
	    version: string;
	    commit: string;
	    date: string;
	
	    static createFrom(source: any = {}) {
	        return new BuildInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.commit = source["commit"];
	        this.date = source["date"];
	    }
	}
	export class CompanySummary {
	    ID: string;
	    CNPJ: string;
	    CNPJRoot: string;
	    Name: string;
	    CredentialID: string;
	    CredentialLabel: string;
	    CredentialCertPath: string;
	    Environment: string;
	    LastFoundNSU?: number;
	    // Go type: time
	    LastSyncAt?: any;
	    SyncStartPolicy: string;
	    // Go type: time
	    SyncStartDate?: any;
	    // Go type: time
	    InitialSyncDoneAt?: any;
	    LastRunStatus: string;
	    LastRunStopReason: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new CompanySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.CNPJ = source["CNPJ"];
	        this.CNPJRoot = source["CNPJRoot"];
	        this.Name = source["Name"];
	        this.CredentialID = source["CredentialID"];
	        this.CredentialLabel = source["CredentialLabel"];
	        this.CredentialCertPath = source["CredentialCertPath"];
	        this.Environment = source["Environment"];
	        this.LastFoundNSU = source["LastFoundNSU"];
	        this.LastSyncAt = this.convertValues(source["LastSyncAt"], null);
	        this.SyncStartPolicy = source["SyncStartPolicy"];
	        this.SyncStartDate = this.convertValues(source["SyncStartDate"], null);
	        this.InitialSyncDoneAt = this.convertValues(source["InitialSyncDoneAt"], null);
	        this.LastRunStatus = source["LastRunStatus"];
	        this.LastRunStopReason = source["LastRunStopReason"];
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
	export class ConnectionTestResult {
	    certLoaded: boolean;
	    certSubject: string;
	    certExpiration: string;
	    mtlsAccepted: boolean;
	    endpointReached: boolean;
	    responseCode: string;
	    responseDetail: string;
	    statusExplanation: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.certLoaded = source["certLoaded"];
	        this.certSubject = source["certSubject"];
	        this.certExpiration = source["certExpiration"];
	        this.mtlsAccepted = source["mtlsAccepted"];
	        this.endpointReached = source["endpointReached"];
	        this.responseCode = source["responseCode"];
	        this.responseDetail = source["responseDetail"];
	        this.statusExplanation = source["statusExplanation"];
	    }
	}
	export class CredentialSummary {
	    ID: string;
	    Label: string;
	    CertPath: string;
	    OwnerCNPJ: string;
	    OwnerCNPJRoot: string;
	    FingerprintSHA256: string;
	    SubjectName: string;
	    // Go type: time
	    NotBefore?: any;
	    // Go type: time
	    NotAfter?: any;
	    // Go type: time
	    InspectedAt?: any;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new CredentialSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Label = source["Label"];
	        this.CertPath = source["CertPath"];
	        this.OwnerCNPJ = source["OwnerCNPJ"];
	        this.OwnerCNPJRoot = source["OwnerCNPJRoot"];
	        this.FingerprintSHA256 = source["FingerprintSHA256"];
	        this.SubjectName = source["SubjectName"];
	        this.NotBefore = this.convertValues(source["NotBefore"], null);
	        this.NotAfter = this.convertValues(source["NotAfter"], null);
	        this.InspectedAt = this.convertValues(source["InspectedAt"], null);
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
	export class DocumentEvent {
	    ID: string;
	    Type: string;
	    // Go type: time
	    EventAt?: any;
	    ReplacementChaveAcesso: string;
	    Description: string;
	    RawXMLPath: string;
	
	    static createFrom(source: any = {}) {
	        return new DocumentEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Type = source["Type"];
	        this.EventAt = this.convertValues(source["EventAt"], null);
	        this.ReplacementChaveAcesso = source["ReplacementChaveAcesso"];
	        this.Description = source["Description"];
	        this.RawXMLPath = source["RawXMLPath"];
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
	export class DocumentRow {
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
	    TotalRetentions: number;
	    Status: string;
	    LayoutVersion: string;
	    XMLPath: string;
	    RawHash: string;
	    ParseWarnings: string[];
	    NFSeNumber: string;
	    ServiceDescription: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	    RelationID: string;
	    CompanyID: string;
	    DocumentID: string;
	    CompanyRole: string;
	    VisibilityReason: string;
	    FirstSeenNSU?: number;
	    LastSeenNSU?: number;
	    // Go type: time
	    FirstSyncedAt: any;
	    // Go type: time
	    LastSyncedAt: any;
	    // Go type: time
	    ViewedAt?: any;
	
	    static createFrom(source: any = {}) {
	        return new DocumentRow(source);
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
	        this.TotalRetentions = source["TotalRetentions"];
	        this.Status = source["Status"];
	        this.LayoutVersion = source["LayoutVersion"];
	        this.XMLPath = source["XMLPath"];
	        this.RawHash = source["RawHash"];
	        this.ParseWarnings = source["ParseWarnings"];
	        this.NFSeNumber = source["NFSeNumber"];
	        this.ServiceDescription = source["ServiceDescription"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	        this.RelationID = source["RelationID"];
	        this.CompanyID = source["CompanyID"];
	        this.DocumentID = source["DocumentID"];
	        this.CompanyRole = source["CompanyRole"];
	        this.VisibilityReason = source["VisibilityReason"];
	        this.FirstSeenNSU = source["FirstSeenNSU"];
	        this.LastSeenNSU = source["LastSeenNSU"];
	        this.FirstSyncedAt = this.convertValues(source["FirstSyncedAt"], null);
	        this.LastSyncedAt = this.convertValues(source["LastSyncedAt"], null);
	        this.ViewedAt = this.convertValues(source["ViewedAt"], null);
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
	export class ExportDANFSeInput {
	    CNPJ: string;
	    ChaveAcesso: string;
	    OutPath: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportDANFSeInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CNPJ = source["CNPJ"];
	        this.ChaveAcesso = source["ChaveAcesso"];
	        this.OutPath = source["OutPath"];
	    }
	}
	export class ExportDocumentsInput {
	    CNPJ: string;
	    Competence: string;
	    Direction: string;
	    Format: string;
	    OutPath: string;
	    Incremental: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExportDocumentsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CNPJ = source["CNPJ"];
	        this.Competence = source["Competence"];
	        this.Direction = source["Direction"];
	        this.Format = source["Format"];
	        this.OutPath = source["OutPath"];
	        this.Incremental = source["Incremental"];
	    }
	}
	export class ExportResult {
	    OutPath: string;
	    Format: string;
	    Incremental: boolean;
	    ExportedCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.OutPath = source["OutPath"];
	        this.Format = source["Format"];
	        this.Incremental = source["Incremental"];
	        this.ExportedCount = source["ExportedCount"];
	    }
	}
	export class ExportXMLInput {
	    CNPJ: string;
	    ChaveAcesso: string;
	    OutPath: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportXMLInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CNPJ = source["CNPJ"];
	        this.ChaveAcesso = source["ChaveAcesso"];
	        this.OutPath = source["OutPath"];
	    }
	}

}

