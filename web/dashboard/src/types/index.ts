export interface Config {
  server: {
    port: number
    address: string
  }
  llm: {
    default_provider: string
    providers: Record<string, {
      api_key: string
      model: string
      base_url?: string
    }>
  }
  channels: {
    telegram?: {
      enabled: boolean
      bot_token: string
    }
    discord?: {
      enabled: boolean
      token: string
    }
  }
  storage: {
    data_dir: string
  }
}

export interface Status {
  version: string
  uptime: string
  server: {
    address: string
    port: number
  }
  llm: {
    provider: string
    model: string
    connected: boolean
  }
  channels: {
    telegram: boolean
    discord: boolean
  }
  skills: number
}

export interface Skill {
  name: string
  version: string
  description: string
  enabled: boolean
  tools: number
}

export interface PersonaFile {
  name: string
  content: string
  lastModified: string
}
