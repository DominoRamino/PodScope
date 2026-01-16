import { Search, Download, Wifi, WifiOff, Pause, Play } from 'lucide-react'

interface HeaderProps {
  connected: boolean
  flowCount: number
  pcapSize: number
  filter: string
  onFilterChange: (filter: string) => void
  onDownloadPCAP: () => void
  isPaused: boolean
  onTogglePause: () => void
}

export function Header({
  connected,
  flowCount,
  pcapSize,
  filter,
  onFilterChange,
  onDownloadPCAP,
  isPaused,
  onTogglePause,
}: HeaderProps) {
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
  }

  return (
    <header className="bg-slate-800 border-b border-slate-700 px-4 py-3">
      <div className="flex items-center justify-between">
        {/* Logo and Title */}
        <div className="flex items-center gap-3">
          <div className="text-2xl">🦈</div>
          <div>
            <h1 className="text-xl font-bold text-white">PodScope</h1>
            <p className="text-xs text-slate-400">Kubernetes Traffic Analyzer</p>
          </div>
        </div>

        {/* Search/Filter */}
        <div className="flex-1 max-w-xl mx-8">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              placeholder="Filter by IP, Pod, Service, URL, SNI..."
              value={filter}
              onChange={(e) => onFilterChange(e.target.value)}
              className="w-full bg-slate-700 border border-slate-600 rounded-lg pl-10 pr-4 py-2 text-sm text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-podscope-500 focus:border-transparent"
            />
          </div>
        </div>

        {/* Status and Actions */}
        <div className="flex items-center gap-6">
          {/* Connection Status */}
          <div className="flex items-center gap-2">
            {connected ? (
              <>
                <Wifi className="w-4 h-4 text-green-400" />
                <span className="text-sm text-green-400">Live</span>
              </>
            ) : (
              <>
                <WifiOff className="w-4 h-4 text-red-400" />
                <span className="text-sm text-red-400">Disconnected</span>
              </>
            )}
          </div>

          {/* Stats */}
          <div className="flex items-center gap-4 text-sm text-slate-400">
            <span>{flowCount} flows</span>
            <span>{formatBytes(pcapSize)} captured</span>
          </div>

          {/* Pause/Resume Button */}
          <button
            onClick={onTogglePause}
            className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
              isPaused
                ? 'bg-yellow-600 hover:bg-yellow-700 text-white'
                : 'bg-slate-700 hover:bg-slate-600 text-slate-300'
            }`}
            title={isPaused ? 'Resume streaming' : 'Pause streaming'}
          >
            {isPaused ? (
              <>
                <Play className="w-4 h-4" />
                Resume
              </>
            ) : (
              <>
                <Pause className="w-4 h-4" />
                Pause
              </>
            )}
          </button>

          {/* Download Button */}
          <button
            onClick={onDownloadPCAP}
            className="flex items-center gap-2 bg-podscope-600 hover:bg-podscope-700 px-4 py-2 rounded-lg text-sm font-medium transition-colors"
          >
            <Download className="w-4 h-4" />
            Download PCAP
          </button>
        </div>
      </div>
    </header>
  )
}
