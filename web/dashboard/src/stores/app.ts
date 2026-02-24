import { create } from 'zustand'
import type { Config, Status, Skill } from '@/types'

interface AppState {
  // Config
  config: Config | null
  setConfig: (config: Config) => void
  
  // Status
  status: Status | null
  setStatus: (status: Status) => void
  
  // Skills
  skills: Skill[]
  setSkills: (skills: Skill[]) => void
  
  // UI State
  sidebarOpen: boolean
  toggleSidebar: () => void
  
  // Real-time updates
  lastUpdate: Date | null
  setLastUpdate: (date: Date) => void
}

export const useStore = create<AppState>((set) => ({
  config: null,
  setConfig: (config) => set({ config }),
  
  status: null,
  setStatus: (status) => set({ status }),
  
  skills: [],
  setSkills: (skills) => set({ skills }),
  
  sidebarOpen: true,
  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
  
  lastUpdate: null,
  setLastUpdate: (date) => set({ lastUpdate: date }),
}))
