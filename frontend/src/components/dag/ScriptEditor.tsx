import Editor, { type OnMount } from '@monaco-editor/react'
import { useEffect } from 'react'
import {
  ensureScriptCompletionProvider,
  updateScriptEditorParamNames,
} from './scriptEditorCompletion'

type ScriptEditorProps = {
  value?: string
  onChange?: (value: string) => void
  readOnly?: boolean
  height?: number
  paramNames?: string[]
}

export default function ScriptEditor({
  value,
  onChange,
  readOnly = false,
  height = 160,
  paramNames = [],
}: ScriptEditorProps) {
  useEffect(() => {
    updateScriptEditorParamNames(paramNames)
  }, [paramNames])

  const handleMount: OnMount = (monacoEditor, monaco) => {
    ensureScriptCompletionProvider(monaco)

    monacoEditor.onKeyDown((event) => {
      // 仅拦截会触发 React Flow 全局快捷键的按键，保留方向键给补全列表。
      if (event.keyCode === monaco.KeyCode.Space) {
        event.stopPropagation()
      }
    })
  }

  return (
    <div className="nopan nodrag nokey">
      <Editor
        height={height}
        defaultLanguage="shell"
        theme="vs-dark"
        value={value ?? ''}
        onChange={(next) => onChange?.(next ?? '')}
        onMount={handleMount}
        options={{
          readOnly,
          minimap: { enabled: false },
          fontSize: 13,
          scrollBeyondLastLine: false,
          wordWrap: 'on',
          fixedOverflowWidgets: true,
          quickSuggestions: { other: true, strings: true, comments: false },
          suggest: {
            showWords: false,
          },
        }}
      />
    </div>
  )
}
