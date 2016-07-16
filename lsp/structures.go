package lsp

type Position struct {
	/**
	 * Line position in a document (zero-based).
	 */
	Line int

	/**
	 * Character offset on a line in a document (zero-based).
	 */
	Character int
}

type Range struct {
	/**
	 * The range's start position.
	 */
	Start Position

	/**
	 * The range's end position.
	 */
	End Position
}

type Location struct {
	URI   string
	Range Range
}

type Diagnostic struct {
	/**
	 * The range at which the message applies.
	 */
	Range Range

	/**
	 * The diagnostic's severity. Can be omitted. If omitted it is up to the
	 * client to interpret diagnostics as error, warning, info or hint.
	 */
	Severity int

	/**
	 * The diagnostic's code. Can be omitted.
	 */
	Code string

	/**
	 * A human-readable string describing the source of this
	 * diagnostic, e.g. 'typescript' or 'super lint'.
	 */
	Source string

	/**
	 * The diagnostic's message.
	 */
	Message string
}

type DiagnosticSeverity int

const (
	Error       DiagnosticSeverity = 1
	Warning                        = 2
	Information                    = 3
	Hint                           = 4
)

type Command struct {
	/**
	 * Title of the command, like `save`.
	 */
	Title string
	/**
	 * The identifier of the actual command handler.
	 */
	Command string
	/**
	 * Arguments that the command handler should be
	 * invoked with.
	 */
	Arguments []interface{}
}

type TextEdit struct {
	/**
	 * The range of the text document to be manipulated. To insert
	 * text into a document create a range where start === end.
	 */
	Range Range

	/**
	 * The string to be inserted. For delete operations use an
	 * empty string.
	 */
	NewText string
}

type WorkspaceEdit struct {
	/**
	 * Holds changes to existing resources.
	 */
	Changes map[string][]TextEdit
}

type TextDocumentIdentifier struct {
	/**
	 * The text document's URI.
	 */
	URI string
}

type TextDocumentItem struct {
	/**
	 * The text document's URI.
	 */
	URI string

	/**
	 * The text document's language identifier.
	 */
	LanguageID string

	/**
	 * The version number of this document (it will strictly increase after each
	 * change, including undo/redo).
	 */
	Version int

	/**
	 * The content of the opened text document.
	 */
	Text string
}

type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	/**
	 * The version number of this document.
	 */
	Version int
}

type TextDocumentPositionParams struct {
	/**
	 * The text document.
	 */
	TextDocument TextDocumentIdentifier

	/**
	 * The position inside the text document.
	 */
	Position Position
}
