import { useEffect, useRef, useCallback } from 'react'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { X, Maximize2, Minimize2 } from 'lucide-react'
import '@xterm/xterm/css/xterm.css'

interface TerminalProps {
  namespace: string
  podName: string
  container?: string
  onClose: () => void
  isMaximized?: boolean
  onToggleMaximize?: () => void
}

export function Terminal({
  namespace,
  podName,
  container,
  onClose,
  isMaximized,
  onToggleMaximize,
}: TerminalProps) {
  const terminalRef = useRef<HTMLDivElement>(null)
  const xtermRef = useRef<XTerm | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)

  const sendResize = useCallback((cols: number, rows: number) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(
        JSON.stringify({
          type: 'resize',
          cols,
          rows,
        })
      )
    }
  }, [])

  useEffect(() => {
    if (!terminalRef.current) return

    // Initialize xterm
    const term = new XTerm({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1e293b',
        foreground: '#e2e8f0',
        cursor: '#22d3ee',
        cursorAccent: '#1e293b',
        selectionBackground: '#475569',
        black: '#1e293b',
        red: '#f87171',
        green: '#4ade80',
        yellow: '#facc15',
        blue: '#60a5fa',
        magenta: '#c084fc',
        cyan: '#22d3ee',
        white: '#f1f5f9',
        brightBlack: '#475569',
        brightRed: '#fca5a5',
        brightGreen: '#86efac',
        brightYellow: '#fde047',
        brightBlue: '#93c5fd',
        brightMagenta: '#d8b4fe',
        brightCyan: '#67e8f9',
        brightWhite: '#f8fafc',
      },
      allowProposedApi: true,
    })

    const fitAddon = new FitAddon()
    const webLinksAddon = new WebLinksAddon()

    term.loadAddon(fitAddon)
    term.loadAddon(webLinksAddon)
    term.open(terminalRef.current)

    xtermRef.current = term
    fitAddonRef.current = fitAddon

    // Fit terminal to container
    setTimeout(() => fitAddon.fit(), 0)

    // Connect WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const params = new URLSearchParams({ namespace, pod: podName })
    if (container) params.set('container', container)

    const wsUrl = `${protocol}//${window.location.host}/api/terminal/ws?${params}`
    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => {
      term.writeln('\x1b[32mConnected to agent terminal\x1b[0m')
      term.writeln(`\x1b[90mPod: ${namespace}/${podName}\x1b[0m`)
      term.writeln('')

      // Send initial size
      sendResize(term.cols, term.rows)
    }

    ws.onmessage = (event) => {
      if (event.data instanceof Blob) {
        event.data.text().then((text) => term.write(text))
      } else {
        term.write(event.data)
      }
    }

    ws.onerror = () => {
      term.writeln('\x1b[31mWebSocket error\x1b[0m')
    }

    ws.onclose = () => {
      term.writeln('\x1b[33m\r\nConnection closed\x1b[0m')
    }

    // Send input to WebSocket
    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data }))
      }
    })

    // Handle resize with debouncing to prevent infinite loops
    let resizeTimeout: number | null = null
    const resizeObserver = new ResizeObserver(() => {
      if (resizeTimeout) {
        clearTimeout(resizeTimeout)
      }
      resizeTimeout = window.setTimeout(() => {
        if (fitAddonRef.current && xtermRef.current) {
          fitAddonRef.current.fit()
          sendResize(xtermRef.current.cols, xtermRef.current.rows)
        }
      }, 100) // Debounce 100ms
    })

    if (terminalRef.current) {
      resizeObserver.observe(terminalRef.current)
    }

    // Cleanup
    return () => {
      if (resizeTimeout) {
        clearTimeout(resizeTimeout)
      }
      resizeObserver.disconnect()
      ws.close()
      term.dispose()
    }
  }, [namespace, podName, container, sendResize])

  return (
    <div className="flex flex-col h-full bg-slate-800 border-t border-slate-700">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2 bg-slate-900 border-b border-slate-700">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-white">Terminal</span>
          <span className="text-xs text-slate-400">
            {namespace}/{podName}
          </span>
        </div>
        <div className="flex items-center gap-1">
          {onToggleMaximize && (
            <button
              onClick={onToggleMaximize}
              className="p-1.5 hover:bg-slate-700 rounded transition-colors"
              title={isMaximized ? 'Minimize' : 'Maximize'}
            >
              {isMaximized ? (
                <Minimize2 className="w-4 h-4 text-slate-400" />
              ) : (
                <Maximize2 className="w-4 h-4 text-slate-400" />
              )}
            </button>
          )}
          <button
            onClick={onClose}
            className="p-1.5 hover:bg-slate-700 rounded transition-colors"
            title="Close terminal"
          >
            <X className="w-4 h-4 text-slate-400" />
          </button>
        </div>
      </div>

      {/* Terminal */}
      <div ref={terminalRef} className="flex-1 p-2 overflow-hidden" />
    </div>
  )
}
