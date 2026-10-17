package nfse

// Export kinds tracked in the NFS-e export marks. Each kind keeps its own
// marks, so exporting the XML does not mark the spreadsheet as exported.
const (
	ExportKindXML    = "xml"
	ExportKindCSV    = "csv"
	ExportKindXLSX   = "xlsx"
	ExportKindDANFSe = "danfse"
)
