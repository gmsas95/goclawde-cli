import axios from 'axios'
import type { Config, Status, Skill, PersonaFile } from '@/types'

const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
})

export const configApi = {
  get: () => api.get<Config>('/config').then(r => r.data),
  update: (config: Partial<Config>) => api.post<Config>('/config', config).then(r => r.data),
}

export const statusApi = {
  get: () => api.get<Status>('/status').then(r => r.data),
}

export const skillsApi = {
  list: () => api.get<Skill[]>('/skills').then(r => r.data),
  install: (repo: string) => api.post('/skills/install', { repo }).then(r => r.data),
  toggle: (name: string, enabled: boolean) => api.post(`/skills/${name}/toggle`, { enabled }).then(r => r.data),
}

export const personaApi = {
  get: (file: string) => api.get<PersonaFile>(`/persona/${file}`).then(r => r.data),
  update: (file: string, content: string) => api.post(`/persona/${file}`, { content }).then(r => r.data),
  list: () => api.get<string[]>('/persona').then(r => r.data),
}

export default api
