// BPF filter presets for common use cases
export interface BPFPreset {
  label: string
  filter: string
  description: string
}

export const bpfPresets: BPFPreset[] = [
  {
    label: 'HTTP/HTTPS Only',
    filter: 'tcp port 80 or tcp port 443 or tcp port 8080 or tcp port 8443',
    description: 'Capture only HTTP and HTTPS traffic on common ports',
  },
  {
    label: 'Exclude DNS',
    filter: 'not port 53',
    description: 'Exclude DNS traffic to reduce noise',
  },
  {
    label: 'TCP SYN Only',
    filter: 'tcp[tcpflags] & tcp-syn != 0',
    description: 'Capture only TCP connection initiations',
  },
  {
    label: 'High Ports (>1024)',
    filter: 'portrange 1025-65535',
    description: 'Capture only traffic on ephemeral/high ports',
  },
  {
    label: 'HTTPS Only',
    filter: 'tcp port 443',
    description: 'Capture only HTTPS traffic',
  },
  {
    label: 'Clear Filter',
    filter: '',
    description: 'Remove filter and capture all traffic',
  },
]
