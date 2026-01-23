import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi, beforeEach, afterEach, describe, it, expect } from 'vitest'
import { Header } from '../../components/Header'

// Default props for Header component
const createDefaultProps = () => ({
  connected: true,
  flowCount: 0,
  filteredCount: 0,
  pcapSize: 0,
  filter: '',
  onFilterChange: vi.fn(),
  filterOptions: {
    searchText: '',
    showOnlyHTTP: true,
    showDNS: false,
    showAllPorts: false,
  },
  onFilterOptionsChange: vi.fn(),
  onDownloadPCAP: vi.fn(),
  isPaused: false,
  onTogglePause: vi.fn(),
})

describe('Header component', () => {
  beforeEach(() => {
    // Mock fetch for BPF filter and PCAP reset endpoints
    vi.spyOn(globalThis, 'fetch').mockImplementation(() => {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ message: 'success' }),
      } as Response)
    })

    // Mock window.confirm for reset PCAP button
    vi.spyOn(window, 'confirm').mockReturnValue(true)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('connection status indicator', () => {
    it('shows green Live status when connected', () => {
      const props = createDefaultProps()
      props.connected = true

      render(<Header {...props} />)

      const liveText = screen.getByText('Live')
      expect(liveText).toBeInTheDocument()
      expect(liveText).toHaveClass('text-green-400')
    })

    it('shows red Disconnected status when not connected', () => {
      const props = createDefaultProps()
      props.connected = false

      render(<Header {...props} />)

      const disconnectedText = screen.getByText('Disconnected')
      expect(disconnectedText).toBeInTheDocument()
      expect(disconnectedText).toHaveClass('text-red-400')
    })

    it('does not show Live when disconnected', () => {
      const props = createDefaultProps()
      props.connected = false

      render(<Header {...props} />)

      expect(screen.queryByText('Live')).not.toBeInTheDocument()
    })

    it('does not show Disconnected when connected', () => {
      const props = createDefaultProps()
      props.connected = true

      render(<Header {...props} />)

      expect(screen.queryByText('Disconnected')).not.toBeInTheDocument()
    })
  })

  describe('flow count display', () => {
    it('displays flow count of 0', () => {
      const props = createDefaultProps()
      props.flowCount = 0

      render(<Header {...props} />)

      expect(screen.getByText('0 flows')).toBeInTheDocument()
    })

    it('displays flow count of 1', () => {
      const props = createDefaultProps()
      props.flowCount = 1

      render(<Header {...props} />)

      expect(screen.getByText('1 flows')).toBeInTheDocument()
    })

    it('displays large flow count correctly', () => {
      const props = createDefaultProps()
      props.flowCount = 12345

      render(<Header {...props} />)

      expect(screen.getByText('12345 flows')).toBeInTheDocument()
    })
  })

  describe('PCAP size display', () => {
    it('displays 0 bytes captured', () => {
      const props = createDefaultProps()
      props.pcapSize = 0

      render(<Header {...props} />)

      expect(screen.getByText('0 B captured')).toBeInTheDocument()
    })

    it('displays bytes captured for small sizes', () => {
      const props = createDefaultProps()
      props.pcapSize = 512

      render(<Header {...props} />)

      expect(screen.getByText('512 B captured')).toBeInTheDocument()
    })

    it('displays KB for kilobyte sizes', () => {
      const props = createDefaultProps()
      props.pcapSize = 1024

      render(<Header {...props} />)

      expect(screen.getByText('1.00 KB captured')).toBeInTheDocument()
    })

    it('displays MB for megabyte sizes', () => {
      const props = createDefaultProps()
      props.pcapSize = 1024 * 1024

      render(<Header {...props} />)

      expect(screen.getByText('1.00 MB captured')).toBeInTheDocument()
    })

    it('displays formatted KB with decimals', () => {
      const props = createDefaultProps()
      props.pcapSize = 2560 // 2.5 KB

      render(<Header {...props} />)

      expect(screen.getByText('2.50 KB captured')).toBeInTheDocument()
    })
  })

  describe('search input', () => {
    it('renders search input with placeholder', () => {
      const props = createDefaultProps()

      render(<Header {...props} />)

      const searchInput = screen.getByPlaceholderText(/Filter by IP/)
      expect(searchInput).toBeInTheDocument()
    })

    it('displays current filter value', () => {
      const props = createDefaultProps()
      props.filter = 'test-filter'

      render(<Header {...props} />)

      const searchInput = screen.getByPlaceholderText(/Filter by IP/) as HTMLInputElement
      expect(searchInput.value).toBe('test-filter')
    })

    it('calls onFilterChange when typing', async () => {
      const user = userEvent.setup()
      const props = createDefaultProps()

      render(<Header {...props} />)

      const searchInput = screen.getByPlaceholderText(/Filter by IP/)
      await user.type(searchInput, 'a')

      expect(props.onFilterChange).toHaveBeenCalled()
    })

    it('calls onFilterChange with input value on change', async () => {
      const user = userEvent.setup()
      const props = createDefaultProps()

      render(<Header {...props} />)

      const searchInput = screen.getByPlaceholderText(/Filter by IP/)
      await user.type(searchInput, 'x')

      // onFilterChange is called with the new character since input is controlled
      expect(props.onFilterChange).toHaveBeenCalledWith('x')
    })
  })

  describe('pause button', () => {
    it('shows Pause text when not paused', () => {
      const props = createDefaultProps()
      props.isPaused = false

      render(<Header {...props} />)

      const pauseButton = screen.getByRole('button', { name: /pause/i })
      expect(pauseButton).toBeInTheDocument()
      expect(pauseButton).toHaveTextContent('Pause')
    })

    it('shows Resume text when paused', () => {
      const props = createDefaultProps()
      props.isPaused = true

      render(<Header {...props} />)

      const resumeButton = screen.getByRole('button', { name: /resume/i })
      expect(resumeButton).toBeInTheDocument()
      expect(resumeButton).toHaveTextContent('Resume')
    })

    it('does not show Resume when not paused', () => {
      const props = createDefaultProps()
      props.isPaused = false

      render(<Header {...props} />)

      // Check button content doesn't include Resume
      const pauseButton = screen.getByRole('button', { name: /pause/i })
      expect(pauseButton).not.toHaveTextContent('Resume')
    })

    it('does not show Pause text when paused', () => {
      const props = createDefaultProps()
      props.isPaused = true

      render(<Header {...props} />)

      // The button with name Resume should not have Pause as text content
      // (avoiding conflict with button role name)
      const resumeButton = screen.getByRole('button', { name: /resume/i })
      expect(resumeButton.textContent).not.toMatch(/^Pause$/)
    })

    it('calls onTogglePause when clicked', async () => {
      const user = userEvent.setup()
      const props = createDefaultProps()
      props.isPaused = false

      render(<Header {...props} />)

      const pauseButton = screen.getByRole('button', { name: /pause/i })
      await user.click(pauseButton)

      expect(props.onTogglePause).toHaveBeenCalledTimes(1)
    })

    it('calls onTogglePause when Resume button clicked', async () => {
      const user = userEvent.setup()
      const props = createDefaultProps()
      props.isPaused = true

      render(<Header {...props} />)

      const resumeButton = screen.getByRole('button', { name: /resume/i })
      await user.click(resumeButton)

      expect(props.onTogglePause).toHaveBeenCalledTimes(1)
    })

    it('has different styling when paused vs not paused', () => {
      const propsNotPaused = createDefaultProps()
      propsNotPaused.isPaused = false

      const { rerender } = render(<Header {...propsNotPaused} />)

      const pauseButton = screen.getByRole('button', { name: /pause/i })
      expect(pauseButton).toHaveClass('bg-slate-700')

      // Rerender with paused = true
      const propsPaused = createDefaultProps()
      propsPaused.isPaused = true
      rerender(<Header {...propsPaused} />)

      const resumeButton = screen.getByRole('button', { name: /resume/i })
      expect(resumeButton).toHaveClass('bg-yellow-600')
    })
  })

  describe('filter option buttons', () => {
    it('renders HTTP/HTTPS Only button', () => {
      const props = createDefaultProps()

      render(<Header {...props} />)

      expect(screen.getByText(/HTTP\/HTTPS Only/)).toBeInTheDocument()
    })

    it('renders Show DNS button', () => {
      const props = createDefaultProps()

      render(<Header {...props} />)

      expect(screen.getByText(/Show DNS/)).toBeInTheDocument()
    })

    it('renders Show All Ports button', () => {
      const props = createDefaultProps()

      render(<Header {...props} />)

      expect(screen.getByText(/Show All Ports/)).toBeInTheDocument()
    })

    it('calls onFilterOptionsChange when HTTP Only clicked', async () => {
      const user = userEvent.setup()
      const props = createDefaultProps()

      render(<Header {...props} />)

      const httpButton = screen.getByText(/HTTP\/HTTPS Only/)
      await user.click(httpButton)

      expect(props.onFilterOptionsChange).toHaveBeenCalled()
    })
  })

  describe('download button', () => {
    it('renders Download PCAP button', () => {
      const props = createDefaultProps()

      render(<Header {...props} />)

      expect(screen.getByRole('button', { name: /download pcap/i })).toBeInTheDocument()
    })

    it('calls onDownloadPCAP when clicked', async () => {
      const user = userEvent.setup()
      const props = createDefaultProps()

      render(<Header {...props} />)

      const downloadButton = screen.getByRole('button', { name: /download pcap/i })
      await user.click(downloadButton)

      expect(props.onDownloadPCAP).toHaveBeenCalledTimes(1)
    })
  })

  describe('header content', () => {
    it('displays PodScope title', () => {
      const props = createDefaultProps()

      render(<Header {...props} />)

      expect(screen.getByText('PodScope')).toBeInTheDocument()
    })

    it('displays subtitle', () => {
      const props = createDefaultProps()

      render(<Header {...props} />)

      expect(screen.getByText('Kubernetes Traffic Analyzer')).toBeInTheDocument()
    })
  })
})
