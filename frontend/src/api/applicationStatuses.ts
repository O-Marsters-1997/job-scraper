import type { ApplicationStatus } from '../types/applicationStatus'

const API_BASE = 'http://localhost:8080'

export async function fetchApplicationStatuses(): Promise<ApplicationStatus[]> {
  const res = await fetch(`${API_BASE}/application-statuses`, { credentials: 'include' })
  if (!res.ok) throw new Error(`Failed to fetch statuses: ${res.status}`)
  return res.json()
}

export async function createApplicationStatus(name: string, colour: string): Promise<ApplicationStatus> {
  const res = await fetch(`${API_BASE}/application-statuses`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, colour }),
  })
  if (!res.ok) throw new Error(`Failed to create status: ${res.status}`)
  return res.json()
}

export async function updateApplicationStatus(id: string, name: string, colour: string): Promise<ApplicationStatus> {
  const res = await fetch(`${API_BASE}/application-statuses/${id}`, {
    method: 'PATCH',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, colour }),
  })
  if (!res.ok) throw new Error(`Failed to update status: ${res.status}`)
  return res.json()
}

export async function deleteApplicationStatus(id: string): Promise<{ count?: number }> {
  const res = await fetch(`${API_BASE}/application-statuses/${id}`, {
    method: 'DELETE',
    credentials: 'include',
  })
  if (res.status === 409) {
    const body = await res.json()
    return { count: body.count }
  }
  if (!res.ok) throw new Error(`Failed to delete status: ${res.status}`)
  return {}
}
