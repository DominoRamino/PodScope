import { useEffect, useRef, useCallback } from 'react'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { X, Maximize2, Minimize2, Terminal as TerminalIcon } from 'lucide-react'
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

    // Initialize xterm with custom theme matching our design
    const term = new XTerm({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: '"JetBrains Mono", "Fira Code", Menlo, Monaco, "Courier New", monospace',
      lineHeight: 1.4,
      letterSpacing: 0,
      theme: {
        background: '#080810',
        foreground: '#e2e8f0',
        cursor: '#00ffd5',
        cursorAccent: '#080810',
        selectionBackground: 'rgba(0, 255, 213, 0.2)',
        selectionForeground: '#ffffff',
        black: '#0c0c18',
        red: '#ff4757',
        green: '#00ffa3',
        yellow: '#ffd000',
        blue: '#00d4ff',
        magenta: '#c084fc',
        cyan: '#00ffd5',
        white: '#f1f5f9',
        brightBlack: '#1a1a30',
        brightRed: '#ff6b7a',
        brightGreen: '#5cffbc',
        brightYellow: '#ffe066',
        brightBlue: '#66e0ff',
        brightMagenta: '#d8b4fe',
        brightCyan: '#5cfffc',
        brightWhite: '#ffffff',
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
      term.writeln('\x1b[38;2;0;255;213m● Connected to agent terminal\x1b[0m')
      term.writeln(`\x1b[90m  Pod: ${namespace}/${podName}\x1b[0m`)
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
      term.writeln('\x1b[38;2;255;71;87m● WebSocket error\x1b[0m')
    }

    ws.onclose = () => {
      term.writeln('\x1b[38;2;255;208;0m\r\n● Connection closed\x1b[0m')
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
      }, 100)
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
    <div className={`flex flex-col h-full bg-void-950 ${isMaximized ? '' : 'border-t border-glow-400/10'}`}>
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 bg-void-900/80 backdrop-blur-xl border-b border-glow-400/10">
        <div className="flex items-center gap-3">
          <div className="w-7 h-7 rounded-lg bg-glow-400/10 flex items-center justify-center">
            <TerminalIcon className="w-3.5 h-3.5 text-glow-400" />
          </div>
          <div>
            <span className="text-sm font-medium text-white">Terminal</span>
            <span className="text-xs text-gray-500 ml-3 font-mono">
              {namespace}/{podName}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-1">
          {onToggleMaximize && (
            <button
              onClick={onToggleMaximize}
              className="btn-ghost p-2"
              title={isMaximized ? 'Minimize' : 'Maximize'}
            >
              {isMaximized ? (
                <Minimize2 className="w-4 h-4" />
              ) : (
                <Maximize2 className="w-4 h-4" />
              )}
            </button>
          )}
          <button
            onClick={onClose}
            className="btn-ghost p-2 hover:text-status-error hover:bg-status-error/10"
            title="Close terminal"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Terminal container */}
      <div
        ref={terminalRef}
        className="flex-1 p-3 overflow-hidden"
        style={{
          background: 'linear-gradient(180deg, #080810 0%, #050508 100%)'
        }}
      />
    </div>
  )
}
