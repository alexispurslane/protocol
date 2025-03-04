// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"strconv"
)

// ServerCapabilities defines the capabilities provided by a language server.
type ServerCapabilities struct {
	// PositionEncoding the position encoding the server picked from the encodings offered by the client via the client capability `general.positionEncodings`. If the client didn't provide any position encodings the only valid value that a server can return is 'utf-16'. If omitted it defaults to 'utf-16'.
	PositionEncoding PositionEncodingKind `json:"positionEncoding,omitempty"`

	// TextDocumentSync defines how text documents are synced. Is either a detailed structure defining each notification or for backwards compatibility the TextDocumentSyncKind number.
	TextDocumentSync *OneOf[TextDocumentSyncOptions, TextDocumentSyncKind] `json:"textDocumentSync,omitempty"`

	// NotebookDocumentSync defines how notebook documents are synced.
	NotebookDocumentSync *OneOf[NotebookDocumentSyncOptions, NotebookDocumentSyncRegistrationOptions] `json:"notebookDocumentSync,omitempty"`

	// CompletionProvider the server provides completion support.
	CompletionProvider *CompletionOptions `json:"completionProvider,omitempty"`

	// HoverProvider the server provides hover support.
	HoverProvider *OneOf[bool, HoverOptions] `json:"hoverProvider,omitempty"`

	// SignatureHelpProvider the server provides signature help support.
	SignatureHelpProvider *SignatureHelpOptions `json:"signatureHelpProvider,omitempty"`

	// DeclarationProvider the server provides Goto Declaration support.
	DeclarationProvider *OneOf3[bool, DeclarationOptions, DeclarationRegistrationOptions] `json:"declarationProvider,omitempty"`

	// DefinitionProvider the server provides goto definition support.
	DefinitionProvider *OneOf[bool, DefinitionOptions] `json:"definitionProvider,omitempty"`

	// TypeDefinitionProvider the server provides Goto Type Definition support.
	TypeDefinitionProvider *OneOf3[bool, TypeDefinitionOptions, TypeDefinitionRegistrationOptions] `json:"typeDefinitionProvider,omitempty"`

	// ImplementationProvider the server provides Goto Implementation support.
	ImplementationProvider *OneOf3[bool, ImplementationOptions, ImplementationRegistrationOptions] `json:"implementationProvider,omitempty"`

	// ReferencesProvider the server provides find references support.
	ReferencesProvider *OneOf[bool, ReferenceOptions] `json:"referencesProvider,omitempty"`

	// DocumentHighlightProvider the server provides document highlight support.
	DocumentHighlightProvider *OneOf[bool, DocumentHighlightOptions] `json:"documentHighlightProvider,omitempty"`

	// DocumentSymbolProvider the server provides document symbol support.
	DocumentSymbolProvider *OneOf[bool, DocumentSymbolOptions] `json:"documentSymbolProvider,omitempty"`

	// CodeActionProvider the server provides code actions. CodeActionOptions may only be specified if the client states that it supports `codeActionLiteralSupport` in its initial `initialize` request.
	CodeActionProvider *OneOf[bool, CodeActionOptions] `json:"codeActionProvider,omitempty"`

	// CodeLensProvider the server provides code lens.
	CodeLensProvider *CodeLensOptions `json:"codeLensProvider,omitempty"`

	// DocumentLinkProvider the server provides document link support.
	DocumentLinkProvider *DocumentLinkOptions `json:"documentLinkProvider,omitempty"`

	// ColorProvider the server provides color provider support.
	ColorProvider *OneOf3[bool, DocumentColorOptions, DocumentColorRegistrationOptions] `json:"colorProvider,omitempty"`

	// WorkspaceSymbolProvider the server provides workspace symbol support.
	WorkspaceSymbolProvider *OneOf[bool, WorkspaceSymbolOptions] `json:"workspaceSymbolProvider,omitempty"`

	// DocumentFormattingProvider the server provides document formatting.
	DocumentFormattingProvider *OneOf[bool, DocumentFormattingOptions] `json:"documentFormattingProvider,omitempty"`

	// DocumentRangeFormattingProvider the server provides document range formatting.
	DocumentRangeFormattingProvider *OneOf[bool, DocumentRangeFormattingOptions] `json:"documentRangeFormattingProvider,omitempty"`

	// DocumentOnTypeFormattingProvider the server provides document formatting on typing.
	DocumentOnTypeFormattingProvider *DocumentOnTypeFormattingOptions `json:"documentOnTypeFormattingProvider,omitempty"`

	// RenameProvider the server provides rename support. RenameOptions may only be specified if the client states that it
	// supports `prepareSupport` in its initial `initialize` request.
	RenameProvider *OneOf[bool, RenameOptions] `json:"renameProvider,omitempty"`

	// FoldingRangeProvider the server provides folding provider support.
	FoldingRangeProvider *OneOf3[bool, FoldingRangeOptions, FoldingRangeRegistrationOptions] `json:"foldingRangeProvider,omitempty"`

	// SelectionRangeProvider the server provides selection range support.
	SelectionRangeProvider *OneOf3[bool, SelectionRangeOptions, SelectionRangeRegistrationOptions] `json:"selectionRangeProvider,omitempty"`

	// ExecuteCommandProvider the server provides execute command support.
	ExecuteCommandProvider *ExecuteCommandOptions `json:"executeCommandProvider,omitempty"`

	// CallHierarchyProvider the server provides call hierarchy support.
	CallHierarchyProvider *OneOf3[bool, CallHierarchyOptions, CallHierarchyRegistrationOptions] `json:"callHierarchyProvider,omitempty"`

	// LinkedEditingRangeProvider the server provides linked editing range support.
	LinkedEditingRangeProvider *OneOf3[bool, LinkedEditingRangeOptions, LinkedEditingRangeRegistrationOptions] `json:"linkedEditingRangeProvider,omitempty"`

	// SemanticTokensProvider the server provides semantic tokens support.
	SemanticTokensProvider *OneOf[SemanticTokensOptions, SemanticTokensRegistrationOptions] `json:"semanticTokensProvider,omitempty"`

	// MonikerProvider the server provides moniker support.
	MonikerProvider *OneOf3[bool, MonikerOptions, MonikerRegistrationOptions] `json:"monikerProvider,omitempty"`

	// TypeHierarchyProvider the server provides type hierarchy support.
	TypeHierarchyProvider *OneOf3[bool, TypeHierarchyOptions, TypeHierarchyRegistrationOptions] `json:"typeHierarchyProvider,omitempty"`

	// InlineValueProvider the server provides inline values.
	InlineValueProvider *OneOf3[bool, InlineValueOptions, InlineValueRegistrationOptions] `json:"inlineValueProvider,omitempty"`

	// InlayHintProvider the server provides inlay hints.
	InlayHintProvider *OneOf3[bool, InlayHintOptions, InlayHintRegistrationOptions] `json:"inlayHintProvider,omitempty"`

	// DiagnosticProvider the server has support for pull model diagnostics.
	DiagnosticProvider *OneOf[DiagnosticOptions, DiagnosticRegistrationOptions] `json:"diagnosticProvider,omitempty"`

	// InlineCompletionProvider inline completion options used during static registration.  3.18.0 @proposed.
	InlineCompletionProvider *OneOf[bool, InlineCompletionOptions] `json:"inlineCompletionProvider,omitempty"`

	// Workspace workspace specific server capabilities.
	Workspace *WorkspaceOptions `json:"workspace,omitempty"`

	// Experimental experimental server capabilities.
	Experimental any `json:"experimental,omitempty"`
}

// TextDocumentSyncOptions TextDocumentSync options.
type TextDocumentSyncOptions struct {
	// OpenClose open and close notifications are sent to the server. If omitted open close notification should not be sent.
	OpenClose bool `json:"openClose,omitempty"`

	// Change change notifications are sent to the server. See TextDocumentSyncKind.None, TextDocumentSyncKind.Full and TextDocumentSyncKind.Incremental. If omitted it defaults to TextDocumentSyncKind.None.
	Change TextDocumentSyncKind `json:"change,omitempty"`

	// WillSave if present will save notifications are sent to the server. If omitted the notification should not be
	// sent.
	WillSave bool `json:"willSave,omitempty"`

	// WillSaveWaitUntil if present will save wait until requests are sent to the server. If omitted the request should not be sent.
	WillSaveWaitUntil bool `json:"willSaveWaitUntil,omitempty"`

	// Save if present save notifications are sent to the server. If omitted the notification should not be sent.
	Save *OneOf[bool, SaveOptions] `json:"save,omitempty"`
}

// SaveOptions save options.
type SaveOptions struct {
	// IncludeText the client is supposed to include the content on save.
	IncludeText bool `json:"includeText,omitempty"`
}

// TextDocumentSyncKind defines how the host (editor) should sync document changes to the language server.
type TextDocumentSyncKind float64

const (
	// TextDocumentSyncKindNone documents should not be synced at all.
	TextDocumentSyncKindNone TextDocumentSyncKind = 0

	// TextDocumentSyncKindFull documents are synced by always sending the full content
	// of the document.
	TextDocumentSyncKindFull TextDocumentSyncKind = 1

	// TextDocumentSyncKindIncremental documents are synced by sending the full content on open.
	// After that only incremental updates to the document are
	// send.
	TextDocumentSyncKindIncremental TextDocumentSyncKind = 2
)

// String implements fmt.Stringer.
func (k TextDocumentSyncKind) String() string {
	switch k {
	case TextDocumentSyncKindNone:
		return "None"
	case TextDocumentSyncKindFull:
		return "Full"
	case TextDocumentSyncKindIncremental:
		return "Incremental"
	default:
		return strconv.FormatFloat(float64(k), 'f', -10, 64)
	}
}

// CompletionOptions completion options.
type CompletionOptions struct {
	// mixins
	WorkDoneProgressOptions

	// TriggerCharacters most tools trigger completion request automatically without explicitly requesting it using a keyboard shortcut (e.g. Ctrl+Space). Typically they do so when the user starts to type an identifier. For example if the user types `c` in a JavaScript file code complete will automatically pop up present `console` besides others as a completion item. Characters that make up identifiers don't need to be listed here. If code complete should automatically be trigger on characters not being valid inside an identifier (for example `.` in JavaScript) list them in `triggerCharacters`.
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`

	// AllCommitCharacters the list of all possible characters that commit a completion. This field can be used if clients don't support individual commit characters per completion item. See `ClientCapabilities.textDocument.completion.completionItem.commitCharactersSupport` If a server provides both `allCommitCharacters` and commit characters on an individual completion item the ones on the completion item win.
	AllCommitCharacters []string `json:"allCommitCharacters,omitempty"`

	// ResolveProvider the server provides support to resolve additional information for a completion item.
	ResolveProvider bool `json:"resolveProvider,omitempty"`

	// CompletionItem the server supports the following `CompletionItem` specific capabilities.
	CompletionItem *ServerCompletionItemOptions `json:"completionItem,omitempty"`
}

// HoverOptions hover options.
type HoverOptions struct {
	// mixins
	WorkDoneProgressOptions
}

// SignatureHelpOptions server Capabilities for a SignatureHelpRequest.
type SignatureHelpOptions struct {
	// mixins
	WorkDoneProgressOptions

	// TriggerCharacters list of characters that trigger signature help automatically.
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`

	// RetriggerCharacters list of characters that re-trigger signature help. These trigger characters are only active when signature help is already showing. All trigger characters are also counted as re-trigger characters.
	RetriggerCharacters []string `json:"retriggerCharacters,omitempty"`
}

// DefinitionOptions server Capabilities for a DefinitionRequest.
type DefinitionOptions struct {
	// mixins
	WorkDoneProgressOptions
}

// ReferenceOptions reference options.
type ReferenceOptions struct {
	// mixins
	WorkDoneProgressOptions
}

// DocumentHighlightOptions provider options for a DocumentHighlightRequest.
type DocumentHighlightOptions struct {
	// mixins
	WorkDoneProgressOptions
}

// DocumentSymbolOptions provider options for a DocumentSymbolRequest.
type DocumentSymbolOptions struct {
	// mixins
	WorkDoneProgressOptions

	// Label a human-readable string that is shown when multiple outlines trees are shown for the same document.
	Label string `json:"label,omitempty"`
}

type TypeDefinitionOptions struct {
	// mixins
	WorkDoneProgressOptions
}

type TypeDefinitionRegistrationOptions struct {
	// extends
	TextDocumentRegistrationOptions
	TypeDefinitionOptions
	// mixins
	StaticRegistrationOptions
}

// ImplementationOptions registration option of Implementation server capability.
//
// @since 3.15.0.
type ImplementationOptions struct {
	WorkDoneProgressOptions
}

// ImplementationRegistrationOptions registration option of Implementation server capability.
//
// @since 3.15.0.
type ImplementationRegistrationOptions struct {
	TextDocumentRegistrationOptions
	ImplementationOptions
	StaticRegistrationOptions
}

// CodeActionOptions provider options for a CodeActionRequest.
type CodeActionOptions struct {
	// mixins
	WorkDoneProgressOptions

	// CodeActionKinds codeActionKinds that this server may return. The list of kinds may be generic, such as `CodeActionKind.Refactor`, or the server may list out every specific kind they provide.
	CodeActionKinds []CodeActionKind `json:"codeActionKinds,omitempty"`

	// Documentation static documentation for a class of code actions. Documentation from the provider should be shown in
	// the code actions menu if either: - Code actions of `kind` are requested by the editor. In this
	// case, the editor will show the documentation that most closely matches the requested code action kind. For example, if a provider has documentation for both `Refactor` and `RefactorExtract`, when the user requests code actions for `RefactorExtract`, the editor will use the documentation for `RefactorExtract` instead of the documentation for `Refactor`. - Any code actions of `kind` are returned by the provider. At most one documentation entry should be shown per provider. 3.18.0 @proposed.
	Documentation []CodeActionKindDocumentation `json:"documentation,omitempty"`

	// ResolveProvider the server provides support to resolve additional information for a code action.
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// CodeLensOptions code Lens provider options of a CodeLensRequest.
type CodeLensOptions struct {
	// mixins
	WorkDoneProgressOptions

	// ResolveProvider code lens has a resolve provider as well.
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// DocumentLinkOptions provider options for a DocumentLinkRequest.
type DocumentLinkOptions struct {
	// mixins
	WorkDoneProgressOptions

	// ResolveProvider document links have a resolve provider as well.
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

type DocumentColorOptions struct {
	// mixins
	WorkDoneProgressOptions
}

type DocumentColorRegistrationOptions struct {
	// extends
	TextDocumentRegistrationOptions
	DocumentColorOptions
	// mixins
	StaticRegistrationOptions
}

// WorkspaceSymbolOptions server capabilities for a WorkspaceSymbolRequest.
type WorkspaceSymbolOptions struct {
	// mixins
	WorkDoneProgressOptions

	// ResolveProvider the server provides support to resolve additional information for a workspace symbol.
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// DocumentFormattingOptions provider options for a DocumentFormattingRequest.
type DocumentFormattingOptions struct {
	// mixins
	WorkDoneProgressOptions
}

// DocumentRangeFormattingOptions provider options for a DocumentRangeFormattingRequest.
type DocumentRangeFormattingOptions struct {
	// mixins
	WorkDoneProgressOptions

	// RangesSupport whether the server supports formatting multiple ranges at once.  3.18.0 @proposed.
	RangesSupport bool `json:"rangesSupport,omitempty"`
}

// DocumentOnTypeFormattingOptions provider options for a DocumentOnTypeFormattingRequest.
type DocumentOnTypeFormattingOptions struct {
	// FirstTriggerCharacter a character on which formatting should be triggered, like `{`.
	FirstTriggerCharacter string `json:"firstTriggerCharacter"`

	// MoreTriggerCharacter more trigger characters.
	MoreTriggerCharacter []string `json:"moreTriggerCharacter,omitempty"`
}

// RenameOptions provider options for a RenameRequest.
type RenameOptions struct {
	// mixins
	WorkDoneProgressOptions

	// PrepareProvider renames should be checked and tested before being executed.  version .
	PrepareProvider bool `json:"prepareProvider,omitempty"`
}

type FoldingRangeOptions struct {
	// mixins
	WorkDoneProgressOptions
}

type FoldingRangeRegistrationOptions struct {
	// extends
	TextDocumentRegistrationOptions
	FoldingRangeOptions
	// mixins
	StaticRegistrationOptions
}

// ExecuteCommandOptions the server capabilities of a ExecuteCommandRequest.
type ExecuteCommandOptions struct {
	// mixins
	WorkDoneProgressOptions

	// Commands the commands to be executed on the server.
	Commands []string `json:"commands"`
}

// CallHierarchyOptions call hierarchy options used during static registration.
//
// @since 3.16.0
type CallHierarchyOptions struct {
	// mixins
	WorkDoneProgressOptions
}

// CallHierarchyRegistrationOptions call hierarchy options used during static or dynamic registration.
//
// @since 3.16.0
type CallHierarchyRegistrationOptions struct {
	// extends
	TextDocumentRegistrationOptions
	CallHierarchyOptions
	// mixins
	StaticRegistrationOptions
}

type LinkedEditingRangeOptions struct {
	// mixins
	WorkDoneProgressOptions
}

type LinkedEditingRangeRegistrationOptions struct {
	// extends
	TextDocumentRegistrationOptions
	LinkedEditingRangeOptions
	// mixins
	StaticRegistrationOptions
}

// SemanticTokensOptions.
//
// @since 3.16.0
type SemanticTokensOptions struct {
	// mixins
	WorkDoneProgressOptions

	// Legend the legend used by the server.
	//
	// @since 3.16.0
	Legend SemanticTokensLegend `json:"legend"`

	// Range server supports providing semantic tokens for a specific range of a document.
	//
	// @since 3.16.0
	Range *OneOf[bool, Range] `json:"range,omitempty"`

	// Full server supports providing semantic tokens for a full document.
	//
	// @since 3.16.0
	Full *OneOf[bool, SemanticTokensFullDelta] `json:"full,omitempty"`
}

// SemanticTokensRegistrationOptions.
//
// @since 3.16.0
type SemanticTokensRegistrationOptions struct {
	// extends
	TextDocumentRegistrationOptions
	SemanticTokensOptions
	// mixins
	StaticRegistrationOptions
}

// ServerCapabilitiesWorkspace specific server capabilities.
type ServerCapabilitiesWorkspace struct {
	// WorkspaceFolders is the server supports workspace folder.
	//
	// @since 3.6.0.
	WorkspaceFolders *ServerCapabilitiesWorkspaceFolders `json:"workspaceFolders,omitempty"`

	// FileOperations is the server is interested in file notifications/requests.
	//
	// @since 3.16.0.
	FileOperations *ServerCapabilitiesWorkspaceFileOperations `json:"fileOperations,omitempty"`
}

// ServerCapabilitiesWorkspaceFolders is the server supports workspace folder.
//
// @since 3.6.0.
type ServerCapabilitiesWorkspaceFolders struct {
	// Supported is the server has support for workspace folders
	Supported bool `json:"supported,omitempty"`

	// ChangeNotifications whether the server wants to receive workspace folder
	// change notifications.
	//
	// If a strings is provided the string is treated as a ID
	// under which the notification is registered on the client
	// side. The ID can be used to unregister for these events
	// using the `client/unregisterCapability` request.
	ChangeNotifications any `json:"changeNotifications,omitempty"` // string | boolean
}

// ServerCapabilitiesWorkspaceFileOperations is the server is interested in file notifications/requests.
//
// @since 3.16.0.
type ServerCapabilitiesWorkspaceFileOperations struct {
	// DidCreate is the server is interested in receiving didCreateFiles
	// notifications.
	DidCreate *FileOperationRegistrationOptions `json:"didCreate,omitempty"`

	// WillCreate is the server is interested in receiving willCreateFiles requests.
	WillCreate *FileOperationRegistrationOptions `json:"willCreate,omitempty"`

	// DidRename is the server is interested in receiving didRenameFiles
	// notifications.
	DidRename *FileOperationRegistrationOptions `json:"didRename,omitempty"`

	// WillRename is the server is interested in receiving willRenameFiles requests.
	WillRename *FileOperationRegistrationOptions `json:"willRename,omitempty"`

	// DidDelete is the server is interested in receiving didDeleteFiles file
	// notifications.
	DidDelete *FileOperationRegistrationOptions `json:"didDelete,omitempty"`

	// WillDelete is the server is interested in receiving willDeleteFiles file
	// requests.
	WillDelete *FileOperationRegistrationOptions `json:"willDelete,omitempty"`
}

// FileOperationRegistrationOptions the options to register for file operations.
//
// @since 3.16.0
type FileOperationRegistrationOptions struct {
	// Filters the actual filters.
	//
	// @since 3.16.0
	Filters []FileOperationFilter `json:"filters"`
}

type MonikerOptions struct {
	// mixins
	WorkDoneProgressOptions
}

type MonikerRegistrationOptions struct {
	// extends
	TextDocumentRegistrationOptions
	MonikerOptions
}
