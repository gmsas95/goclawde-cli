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
  get: () => api.get<Status>('/public/status').then(r => r.data),
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

export const jobsApi = {
  list: () => api.get<any[]>('/jobs').then(r => r.data),
  create: (data: { name: string; description: string; schedule: string }) => 
    api.post('/jobs', data).then(r => r.data),
  delete: (id: string) => api.delete(`/jobs/${id}`).then(r => r.data),
  toggle: (id: string, enabled: boolean) => 
    api.post(`/jobs/${id}/toggle`, { enabled }).then(r => r.data),
  run: (id: string) => api.post(`/jobs/${id}/run`).then(r => r.data),
}

export const clustersApi = {
  list: () => api.get<any[]>('/clusters').then(r => r.data),
  get: (id: string) => api.get<any>(`/clusters/${id}`).then(r => r.data),
  getMemories: (id: string) => api.get<any[]>(`/clusters/${id}/memories`).then(r => r.data),
}

export const activityApi = {
  list: () => api.get<any[]>('/activity').then(r => r.data),
}

export const logsApi = {
  list: (params?: { level?: string; limit?: number }) => 
    api.get<any[]>('/logs', { params }).then(r => r.data),
}

export default api
