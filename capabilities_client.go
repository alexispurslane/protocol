// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

// WorkspaceClientCapabilitiesWorkspaceEdit capabilities specific to "WorkspaceEdit"s.
type WorkspaceClientCapabilitiesWorkspaceEdit struct {
	// DocumentChanges is the client supports versioned document changes in `WorkspaceEdit`s
	DocumentChanges bool `json:"documentChanges,omitempty"`

	// FailureHandling is the failure handling strategy of a client if applying the workspace edit
	// fails.
	//
	// Mostly FailureHandlingKind.
	FailureHandling string `json:"failureHandling,omitempty"`

	// ResourceOperations is the resource operations the client supports. Clients should at least
	// support "create", "rename" and "delete" files and folders.
	ResourceOperations []string `json:"resourceOperations,omitempty"`

	// NormalizesLineEndings whether the client normalizes line endings to the client specific
	// setting.
	// If set to `true` the client will normalize line ending characters
	// in a workspace edit to the client specific new line character(s).
	//
	// @since 3.16.0.
	NormalizesLineEndings bool `json:"normalizesLineEndings,omitempty"`

	// ChangeAnnotationSupport whether the client in general supports change annotations on text edits,
	// create file, rename file and delete file changes.
	//
	// @since 3.16.0.
	ChangeAnnotationSupport *WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport `json:"changeAnnotationSupport,omitempty"`
}

// WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport is the ChangeAnnotationSupport of WorkspaceClientCapabilitiesWorkspaceEdit.
//
// @since 3.16.0.
type WorkspaceClientCapabilitiesWorkspaceEditChangeAnnotationSupport struct {
	// GroupsOnLabel whether the client groups edits with equal labels into tree nodes,
	// for instance all edits labeled with "Changes in Strings" would
	// be a tree node.
	GroupsOnLabel bool `json:"groupsOnLabel,omitempty"`
}

// DidChangeConfigurationWorkspaceClientCapabilities capabilities specific to the "workspace/didChangeConfiguration" notification.
//
// @since 3.16.0.
type DidChangeConfigurationWorkspaceClientCapabilities struct {
	// DynamicRegistration whether the did change configuration notification supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DidChangeWatchedFilesWorkspaceClientCapabilities capabilities specific to the "workspace/didChangeWatchedFiles" notification.
//
// @since 3.16.0.
type DidChangeWatchedFilesWorkspaceClientCapabilities struct {
	// Did change watched files notification supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type SymbolKindCapabilities struct {
	// ValueSet is the symbol kind values the client supports. When this
	// property exists the client also guarantees that it will
	// handle values outside its set gracefully and falls back
	// to a default value when unknown.
	//
	// If this property is not present the client only supports
	// the symbol kinds from `File` to `Array` as defined in
	// the initial version of the protocol.
	ValueSet []SymbolKind `json:"valueSet,omitempty"`
}

type TagSupportCapabilities struct {
	// ValueSet is the tags supported by the client.
	ValueSet []SymbolTag `json:"valueSet,omitempty"`
}

// WorkspaceClientCapabilitiesFileOperations capabilities specific to the fileOperations.
//
// @since 3.16.0.
type WorkspaceClientCapabilitiesFileOperations struct {
	// DynamicRegistration whether the client supports dynamic registration for file
	// requests/notifications.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// DidCreate is the client has support for sending didCreateFiles notifications.
	DidCreate bool `json:"didCreate,omitempty"`

	// WillCreate is the client has support for sending willCreateFiles requests.
	WillCreate bool `json:"willCreate,omitempty"`

	// DidRename is the client has support for sending didRenameFiles notifications.
	DidRename bool `json:"didRename,omitempty"`

	// WillRename is the client has support for sending willRenameFiles requests.
	WillRename bool `json:"willRename,omitempty"`

	// DidDelete is the client has support for sending didDeleteFiles notifications.
	DidDelete bool `json:"didDelete,omitempty"`

	// WillDelete is the client has support for sending willDeleteFiles requests.
	WillDelete bool `json:"willDelete,omitempty"`
}

// CompletionTextDocumentClientCapabilities Capabilities specific to the "textDocument/completion".
type CompletionTextDocumentClientCapabilities struct {
	// Whether completion supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// The client supports the following `CompletionItem` specific
	// capabilities.
	CompletionItem *CompletionTextDocumentClientCapabilitiesItem `json:"completionItem,omitempty"`

	CompletionItemKind *CompletionTextDocumentClientCapabilitiesItemKind `json:"completionItemKind,omitempty"`

	// ContextSupport is the client supports to send additional context information for a
	// `textDocument/completion` request.
	ContextSupport bool `json:"contextSupport,omitempty"`
}

// CompletionTextDocumentClientCapabilitiesItem is the client supports the following "CompletionItem" specific
// capabilities.
type CompletionTextDocumentClientCapabilitiesItem struct {
	// SnippetSupport client supports snippets as insert text.
	//
	// A snippet can define tab stops and placeholders with `$1`, `$2`
	// and `${3:foo}`. `$0` defines the final tab stop, it defaults to
	// the end of the snippet. Placeholders with equal identifiers are linked,
	// that is typing in one will update others too.
	SnippetSupport bool `json:"snippetSupport,omitempty"`

	// CommitCharactersSupport client supports commit characters on a completion item.
	CommitCharactersSupport bool `json:"commitCharactersSupport,omitempty"`

	// DocumentationFormat client supports the follow content formats for the documentation
	// property. The order describes the preferred format of the client.
	DocumentationFormat []MarkupKind `json:"documentationFormat,omitempty"`

	// DeprecatedSupport client supports the deprecated property on a completion item.
	DeprecatedSupport bool `json:"deprecatedSupport,omitempty"`

	// PreselectSupport client supports the preselect property on a completion item.
	PreselectSupport bool `json:"preselectSupport,omitempty"`

	// TagSupport is the client supports the tag property on a completion item.
	//
	// Clients supporting tags have to handle unknown tags gracefully.
	// Clients especially need to preserve unknown tags when sending
	// a completion item back to the server in a resolve call.
	//
	// @since 3.15.0.
	TagSupport *CompletionTextDocumentClientCapabilitiesItemTagSupport `json:"tagSupport,omitempty"`

	// InsertReplaceSupport client supports insert replace edit to control different behavior if
	// a completion item is inserted in the text or should replace text.
	//
	// @since 3.16.0.
	InsertReplaceSupport bool `json:"insertReplaceSupport,omitempty"`

	// ResolveSupport indicates which properties a client can resolve lazily on a
	// completion item. Before version 3.16.0 only the predefined properties
	// `documentation` and `details` could be resolved lazily.
	//
	// @since 3.16.0.
	ResolveSupport *CompletionTextDocumentClientCapabilitiesItemResolveSupport `json:"resolveSupport,omitempty"`

	// InsertTextModeSupport is the client supports the `insertTextMode` property on
	// a completion item to override the whitespace handling mode
	// as defined by the client (see `insertTextMode`).
	//
	// @since 3.16.0.
	InsertTextModeSupport *CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport `json:"insertTextModeSupport,omitempty"`
}

// CompletionTextDocumentClientCapabilitiesItemTagSupport specific capabilities for the "TagSupport" in the "textDocument/completion" request.
//
// @since 3.15.0.
type CompletionTextDocumentClientCapabilitiesItemTagSupport struct {
	// ValueSet is the tags supported by the client.
	//
	// @since 3.15.0.
	ValueSet []CompletionItemTag `json:"valueSet,omitempty"`
}

// CompletionTextDocumentClientCapabilitiesItemResolveSupport specific capabilities for the ResolveSupport in the CompletionTextDocumentClientCapabilitiesItem.
//
// @since 3.16.0.
type CompletionTextDocumentClientCapabilitiesItemResolveSupport struct {
	// Properties is the properties that a client can resolve lazily.
	Properties []string `json:"properties"`
}

// CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport specific capabilities for the InsertTextModeSupport in the CompletionTextDocumentClientCapabilitiesItem.
//
// @since 3.16.0.
type CompletionTextDocumentClientCapabilitiesItemInsertTextModeSupport struct {
	// ValueSet is the tags supported by the client.
	//
	// @since 3.16.0.
	ValueSet []InsertTextMode `json:"valueSet,omitempty"`
}

// CompletionTextDocumentClientCapabilitiesItemKind specific capabilities for the "CompletionItemKind" in the "textDocument/completion" request.
type CompletionTextDocumentClientCapabilitiesItemKind struct {
	// The completion item kind values the client supports. When this
	// property exists the client also guarantees that it will
	// handle values outside its set gracefully and falls back
	// to a default value when unknown.
	//
	// If this property is not present the client only supports
	// the completion items kinds from `Text` to `Reference` as defined in
	// the initial version of the protocol.
	//
	ValueSet []CompletionItemKind `json:"valueSet,omitempty"`
}

// HoverTextDocumentClientCapabilities capabilities specific to the "textDocument/hover".
type HoverTextDocumentClientCapabilities struct {
	// DynamicRegistration whether hover supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// ContentFormat is the client supports the follow content formats for the content
	// property. The order describes the preferred format of the client.
	ContentFormat []MarkupKind `json:"contentFormat,omitempty"`
}

// SignatureHelpTextDocumentClientCapabilities capabilities specific to the "textDocument/signatureHelp".
type SignatureHelpTextDocumentClientCapabilities struct {
	// DynamicRegistration whether signature help supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// SignatureInformation is the client supports the following "SignatureInformation"
	// specific properties.
	SignatureInformation *TextDocumentClientCapabilitiesSignatureInformation `json:"signatureInformation,omitempty"`

	// ContextSupport is the client supports to send additional context information for a "textDocument/signatureHelp" request.
	//
	// A client that opts into contextSupport will also support the "retriggerCharacters" on "SignatureHelpOptions".
	//
	// @since 3.15.0.
	ContextSupport bool `json:"contextSupport,omitempty"`
}

// TextDocumentClientCapabilitiesSignatureInformation is the client supports the following "SignatureInformation"
// specific properties.
type TextDocumentClientCapabilitiesSignatureInformation struct {
	// DocumentationFormat is the client supports the follow content formats for the documentation
	// property. The order describes the preferred format of the client.
	DocumentationFormat []MarkupKind `json:"documentationFormat,omitempty"`

	// ParameterInformation is the Client capabilities specific to parameter information.
	ParameterInformation *TextDocumentClientCapabilitiesParameterInformation `json:"parameterInformation,omitempty"`

	// ActiveParameterSupport is the client supports the `activeParameter` property on
	// `SignatureInformation` literal.
	//
	// @since 3.16.0.
	ActiveParameterSupport bool `json:"activeParameterSupport,omitempty"`
}

// TextDocumentClientCapabilitiesParameterInformation is the client capabilities specific to parameter information.
type TextDocumentClientCapabilitiesParameterInformation struct {
	// LabelOffsetSupport is the client supports processing label offsets instead of a
	// simple label string.
	//
	// @since 3.14.0.
	LabelOffsetSupport bool `json:"labelOffsetSupport,omitempty"`
}

// DeclarationTextDocumentClientCapabilities capabilities specific to the "textDocument/declaration".
type DeclarationTextDocumentClientCapabilities struct {
	// DynamicRegistration whether declaration supports dynamic registration. If this is set to `true`
	// the client supports the new `(TextDocumentRegistrationOptions & StaticRegistrationOptions)`
	// return value for the corresponding server capability as well.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// LinkSupport is the client supports additional metadata in the form of declaration links.
	//
	// @since 3.14.0.
	LinkSupport bool `json:"linkSupport,omitempty"`
}

// DefinitionTextDocumentClientCapabilities capabilities specific to the "textDocument/definition".
//
// @since 3.14.0.
type DefinitionTextDocumentClientCapabilities struct {
	// DynamicRegistration whether definition supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// LinkSupport is the client supports additional metadata in the form of definition links.
	LinkSupport bool `json:"linkSupport,omitempty"`
}

// TypeDefinitionTextDocumentClientCapabilities capabilities specific to the "textDocument/typeDefinition".
//
// @since 3.6.0.
type TypeDefinitionTextDocumentClientCapabilities struct {
	// DynamicRegistration whether typeDefinition supports dynamic registration. If this is set to `true`
	// the client supports the new "(TextDocumentRegistrationOptions & StaticRegistrationOptions)"
	// return value for the corresponding server capability as well.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// LinkSupport is the client supports additional metadata in the form of definition links.
	//
	// @since 3.14.0
	LinkSupport bool `json:"linkSupport,omitempty"`
}

// ImplementationTextDocumentClientCapabilities capabilities specific to the "textDocument/implementation".
//
// @since 3.6.0.
type ImplementationTextDocumentClientCapabilities struct {
	// DynamicRegistration whether implementation supports dynamic registration. If this is set to `true`
	// the client supports the new "(TextDocumentRegistrationOptions & StaticRegistrationOptions)"
	// return value for the corresponding server capability as well.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	// LinkSupport is the client supports additional metadata in the form of definition links.
	//
	// @since 3.14.0
	LinkSupport bool `json:"linkSupport,omitempty"`
}

// ReferencesTextDocumentClientCapabilities capabilities specific to the "textDocument/references".
type ReferencesTextDocumentClientCapabilities struct {
	// DynamicRegistration whether references supports dynamic registration.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentSymbolClientCapabilitiesTagSupport TagSupport in the DocumentSymbolClientCapabilities.
//
// @since 3.16.0.
type DocumentSymbolClientCapabilitiesTagSupport struct {
	// ValueSet is the tags supported by the client.
	ValueSet []SymbolTag `json:"valueSet"`
}

// CodeActionClientCapabilitiesLiteralSupport is the client support code action literals as a valid response of the "textDocument/codeAction" request.
type CodeActionClientCapabilitiesLiteralSupport struct {
	// CodeActionKind is the code action kind is support with the following value
	// set.
	CodeActionKind *CodeActionClientCapabilitiesKind `json:"codeActionKind"`
}

// CodeActionClientCapabilitiesKind is the code action kind is support with the following value set.
type CodeActionClientCapabilitiesKind struct {
	// ValueSet is the code action kind values the client supports. When this
	// property exists the client also guarantees that it will
	// handle values outside its set gracefully and falls back
	// to a default value when unknown.
	ValueSet []CodeActionKind `json:"valueSet"`
}

// CodeActionClientCapabilitiesResolveSupport ResolveSupport in the CodeActionClientCapabilities.
//
// @since 3.16.0.
type CodeActionClientCapabilitiesResolveSupport struct {
	// Properties is the properties that a client can resolve lazily.
	Properties []string `json:"properties"`
}

// PublishDiagnosticsClientCapabilitiesTagSupport is the client capacity of TagSupport.
//
// @since 3.15.0.
type PublishDiagnosticsClientCapabilitiesTagSupport struct {
	// ValueSet is the tags supported by the client.
	ValueSet []DiagnosticTag `json:"valueSet"`
}

// SemanticTokensWorkspaceClientCapabilitiesRequests capabilities specific to the "textDocument/semanticTokens/xxx" request.
//
// @since 3.16.0.
type SemanticTokensWorkspaceClientCapabilitiesRequests struct {
	// Range is the client will send the "textDocument/semanticTokens/range" request
	// if the server provides a corresponding handler.
	Range bool `json:"range,omitempty"`

	// Full is the client will send the "textDocument/semanticTokens/full" request
	// if the server provides a corresponding handler. The client will send the
	// `textDocument/semanticTokens/full/delta` request if the server provides a
	// corresponding handler.
	Full any `json:"full,omitempty"`
}

// ShowMessageRequestClientCapabilitiesMessageActionItem capabilities specific to the "MessageActionItem" type.
//
// @since 3.16.0.
type ShowMessageRequestClientCapabilitiesMessageActionItem struct {
	// AdditionalPropertiesSupport whether the client supports additional attributes which
	// are preserved and sent back to the server in the
	// request's response.
	AdditionalPropertiesSupport bool `json:"additionalPropertiesSupport,omitempty"`
}
