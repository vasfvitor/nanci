import { vi } from 'vitest'

// Importing @wailsio/runtime wires global mouse listeners and calls the backend
// over HTTP. Unit tests never run inside Wails, so every test file gets this
// inert stand-in; tests of the platform wrappers mock it again themselves.
vi.mock('@wailsio/runtime', () => ({
  Application: { Quit: vi.fn(() => Promise.resolve()) },
  Call: {
    ByID: vi.fn(() => Promise.reject(new Error('Wails runtime is not available in unit tests'))),
  },
  Events: { On: vi.fn(() => () => undefined) },
  Screens: { GetAll: vi.fn(() => Promise.resolve([])) },
  Window: {
    Minimise: vi.fn(() => Promise.resolve()),
    ToggleMaximise: vi.fn(() => Promise.resolve()),
    Size: vi.fn(() => Promise.resolve({ width: 1280, height: 768 })),
    SetSize: vi.fn(() => Promise.resolve()),
    IsMaximised: vi.fn(() => Promise.resolve(false)),
  },
}))
