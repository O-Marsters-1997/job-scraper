const API_BASE = 'http://localhost:8080'

export function login(username: string, password: string) {
  return fetch(`${API_BASE}/auth/login`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  })
}

export function signup(username: string, password: string) {
  return fetch(`${API_BASE}/auth/signup`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  })
}

export function logout() {
  return fetch(`${API_BASE}/auth/logout`, {
    method: 'POST',
    credentials: 'include',
  })
}

export function getMe() {
  return fetch(`${API_BASE}/auth/me`, {
    credentials: 'include',
  })
}
