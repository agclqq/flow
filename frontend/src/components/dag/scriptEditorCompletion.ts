import type * as Monaco from 'monaco-editor'
import { SYSTEM_PARAM_KEYS } from '../../constants/systemParams'

const BRACE_VAR_TRIGGER = /\{\{([a-zA-Z0-9_]*)$/
const DOLLAR_VAR_TRIGGER = /\$(\{?)([a-zA-Z0-9_]*)$/

let paramNames: string[] = []
let registered = false

function buildBraceVariableInsertText(
  name: string,
  model: Monaco.editor.ITextModel,
  position: Monaco.IPosition,
): string {
  const lineSuffix = model.getValueInRange({
    startLineNumber: position.lineNumber,
    startColumn: position.column,
    endLineNumber: position.lineNumber,
    endColumn: model.getLineMaxColumn(position.lineNumber),
  })

  // Monaco 输入 {{ 时常自动补上 }}，补全项只插入变量名，避免重复闭合括号。
  if (lineSuffix.startsWith('}}')) {
    return name
  }
  if (lineSuffix.startsWith('}')) {
    return `${name}}`
  }
  return `${name}}}`
}

function buildDollarVariableInsertText(
  name: string,
  model: Monaco.editor.ITextModel,
  position: Monaco.IPosition,
  braced: boolean,
): string {
  if (!braced) {
    return name
  }
  const lineSuffix = model.getValueInRange({
    startLineNumber: position.lineNumber,
    startColumn: position.column,
    endLineNumber: position.lineNumber,
    endColumn: model.getLineMaxColumn(position.lineNumber),
  })
  if (lineSuffix.startsWith('}')) {
    return name
  }
  return `${name}}`
}

export function updateScriptEditorParamNames(names: string[]) {
  paramNames = names
}

function buildVariableNames(): string[] {
  return [...new Set([...paramNames, ...SYSTEM_PARAM_KEYS].filter(Boolean))]
}

function braceCompletions(
  monaco: typeof Monaco,
  model: Monaco.editor.ITextModel,
  position: Monaco.IPosition,
  linePrefix: string,
): Monaco.languages.CompletionList {
  const match = linePrefix.match(BRACE_VAR_TRIGGER)
  if (!match) {
    return { suggestions: [] }
  }

  const partial = match[1] ?? ''
  const openIndex = linePrefix.lastIndexOf('{{')
  const range = {
    startLineNumber: position.lineNumber,
    endLineNumber: position.lineNumber,
    startColumn: openIndex + 3,
    endColumn: position.column,
  }

  return {
    suggestions: buildVariableNames()
      .filter((name) => name.startsWith(partial))
      .map((name) => ({
        label: name,
        kind: monaco.languages.CompletionItemKind.Variable,
        insertText: buildBraceVariableInsertText(name, model, position),
        range,
        detail: `{{${name}}}`,
      })),
  }
}

function dollarCompletions(
  monaco: typeof Monaco,
  model: Monaco.editor.ITextModel,
  position: Monaco.IPosition,
  linePrefix: string,
): Monaco.languages.CompletionList {
  const match = linePrefix.match(DOLLAR_VAR_TRIGGER)
  if (!match) {
    return { suggestions: [] }
  }

  const braced = match[1] === '{'
  const partial = match[2] ?? ''
  const dollarIndex = linePrefix.lastIndexOf('$')
  const range = {
    startLineNumber: position.lineNumber,
    endLineNumber: position.lineNumber,
    startColumn: dollarIndex + 2 + (braced ? 1 : 0),
    endColumn: position.column,
  }

  return {
    suggestions: buildVariableNames()
      .filter((name) => name.startsWith(partial))
      .map((name) => ({
        label: name,
        kind: monaco.languages.CompletionItemKind.Variable,
        insertText: buildDollarVariableInsertText(name, model, position, braced),
        range,
        detail: braced ? `\${${name}}` : `$${name}`,
      })),
  }
}

export function ensureScriptCompletionProvider(monaco: typeof Monaco) {
  if (registered) {
    return
  }
  registered = true

  monaco.languages.registerCompletionItemProvider('shell', {
    triggerCharacters: ['{', '$'],
    provideCompletionItems: (model, position) => {
      const linePrefix = model.getValueInRange({
        startLineNumber: position.lineNumber,
        startColumn: 1,
        endLineNumber: position.lineNumber,
        endColumn: position.column,
      })

      const brace = braceCompletions(monaco, model, position, linePrefix)
      if (brace.suggestions.length > 0) {
        return brace
      }
      return dollarCompletions(monaco, model, position, linePrefix)
    },
  })
}
