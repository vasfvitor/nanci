// readStorage and writeStorage reach localStorage for the preferences that
// survive a restart. A storage that throws (disabled, full or blocked) reads
// as empty and drops the write, so the app runs on its defaults.

// readStorage returns the stored value of key, or null when there is none
// or the storage cannot be read.
export function readStorage(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

// writeStorage stores value under key, and removes key when value is null.
export function writeStorage(key: string, value: string | null) {
  try {
    if (value === null) {
      localStorage.removeItem(key)
    } else {
      localStorage.setItem(key, value)
    }
  } catch {
    // The choice lasts for the session only.
  }
}
